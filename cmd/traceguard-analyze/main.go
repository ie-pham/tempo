// traceguard-analyze queries Tempo's metrics endpoint for trace performance data
// produced during integration tests, comparing a PR commit against a baseline.
//
// Four analyses are performed in parallel:
//  1. Max span duration (min/max/avg of max_over_time samples) per service
//  2. p95 latency (avg of quantile_over_time(0.95) samples) per service
//  3. Error rate (error span rate / total span rate) per service
//  4. New/removed operation names (span:name set diff between commits)
//
// # Remote mode (default)
//
// Requires the integration-test harness to have forwarded OTEL env vars into
// every Tempo container so that Tempo tags its own internal traces with
// resource.service.version = <commit SHA> (see integration/util/harness.go).
//
// # Example invocations
//
//	export GRAFANA_TEMPO_QUERY_URL=https://tempo-prod-05-prod-us-central-0.grafana.net/tempo
//	export GRAFANA_BASIC_AUTH="Basic user:pw"
//	export GIT_COMMIT=abc1234
//	export BASE_COMMIT=def5678
//	export PR_NUMBER=42
//	export GITHUB_REPOSITORY=grafana/tempo
//	export GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
//	go run ./cmd/traceguard-analyze
package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
)

// ── Constants ────────────────────────────────────────────────────────────────

const (
	defaultMinSamples = 30
	searchWindowHours = 6
)

// ── Config ───────────────────────────────────────────────────────────────────

type Config struct {
	Mode         string // "remote"
	TempoURL     string // e.g. https://tempo.example.com (no trailing slash)
	Auth string // Basic auth token for Tempo
	GitCommit    string // PR commit SHA
	BaseCommit   string // baseline commit SHA (optional; skips diff when absent)
	PRNumber     string
	GitHubRepo   string // "owner/repo"
	GitHubToken  string
	MinSamples   int
}

func loadConfig() (Config, error) {
	cfg := Config{
		Mode:         getEnv("MODE", "remote"),
		TempoURL:     strings.TrimRight(os.Getenv("GRAFANA_TEMPO_QUERY_URL"), "/"),
		Auth:         os.Getenv("GRAFANA_BASIC_AUTH"),
		GitCommit:    os.Getenv("GIT_COMMIT"),
		BaseCommit:   os.Getenv("BASE_COMMIT"),
		PRNumber:     os.Getenv("PR_NUMBER"),
		GitHubRepo:   os.Getenv("GITHUB_REPOSITORY"),
		GitHubToken:  os.Getenv("GITHUB_TOKEN"),
		MinSamples:   envInt("MIN_SAMPLES", defaultMinSamples),
	}

	if cfg.Mode == "remote" {
		var missing []string
		if cfg.TempoURL == "" {
			missing = append(missing, "GRAFANA_TEMPO_QUERY_URL")
		}
		if cfg.Auth == "" {
			missing = append(missing, "GRAFANA_AUTH")
		}
		if cfg.GitCommit == "" {
			missing = append(missing, "GIT_COMMIT")
		}
		if len(missing) > 0 {
			return cfg, fmt.Errorf("required env vars missing in remote mode: %s", strings.Join(missing, ", "))
		}
	}

	if cfg.PRNumber != "" && cfg.GitHubToken == "" {
		fmt.Fprintln(os.Stderr, "warning: PR_NUMBER is set but GITHUB_TOKEN is missing — PR comment will be skipped")
	}

	return cfg, nil
}

// ── Domain types ─────────────────────────────────────────────────────────────

// ServiceKey identifies a unique (service.version, rootServiceName) combination.
type ServiceKey struct {
	Version string
	Service string
}

// DurationStats holds min/max/avg of max_over_time(span:duration) samples, in ms.
type DurationStats struct {
	MinMs       float64
	MaxMs       float64
	AvgMs       float64
	SampleCount int
}

// P95Stats holds the time-averaged p95 latency from quantile_over_time samples, in ms.
type P95Stats struct {
	AvgP95Ms    float64
	SampleCount int
}

// ErrorRateStats holds the error rate percentage for one (version, service).
type ErrorRateStats struct {
	ErrorRatePct float64 // 0–100
}

// AnalysisResults bundles all four analyses.
type AnalysisResults struct {
	Duration            map[ServiceKey]DurationStats
	P95                 map[ServiceKey]P95Stats
	ErrorRate           map[ServiceKey]ErrorRateStats
	SpanNames           map[string]map[string]struct{} // version → set of span names
	NewSpanContexts     map[string]string              // span name → tree (spans new in PR)
	RemovedSpanContexts map[string]string              // span name → tree (spans removed in PR)
}

// flatSpan is a span with its service name, flattened for tree building.
type flatSpan struct {
	SpanID      string
	ParentID    string // empty string for root spans
	Name        string
	ServiceName string
	DurationMs  float64
}

// ── main ─────────────────────────────────────────────────────────────────────

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if cfg.BaseCommit == "" {
		fmt.Println("► BASE_COMMIT not set — auto-detecting baseline from Tempo…")
		cfg.BaseCommit = detectBaseCommit(cfg)
		if cfg.BaseCommit != "" {
			fmt.Printf("  detected base: %s\n", cfg.BaseCommit)
		} else {
			fmt.Fprintln(os.Stderr, "  warning: no main.* or baseline.* version found in Tempo; running without baseline")
		}
	}

	fmt.Printf("mode: remote  commit=%s  base=%s\n", cfg.GitCommit, cfg.BaseCommit)
	fmt.Println("► Running 4 analyses in parallel…")

	var (
		dur       map[ServiceKey]DurationStats
		p95       map[ServiceKey]P95Stats
		errorRate map[ServiceKey]ErrorRateStats
		spanNames map[string]map[string]struct{}

		errDur, errP95, errErrRate, errSpan error
		wg                                  sync.WaitGroup
	)

	wg.Add(4)
	go func() { defer wg.Done(); dur, errDur = fetchDurationStats(cfg) }()
	go func() { defer wg.Done(); p95, errP95 = fetchP95Stats(cfg) }()
	go func() { defer wg.Done(); errorRate, errErrRate = fetchErrorRates(cfg) }()
	go func() { defer wg.Done(); spanNames, errSpan = fetchSpanNames(cfg) }()
	wg.Wait()

	for label, e := range map[string]error{
		"duration stats": errDur,
		"p95 latency":    errP95,
		"error rates":    errErrRate,
		"span names":     errSpan,
	} {
		if e != nil {
			fmt.Fprintf(os.Stderr, "warning: %s query failed: %v\n", label, e)
		}
	}

	fmt.Println("► Fetching span context trees for changed operations…")
	newCtxs, removedCtxs := fetchSpanContexts(cfg, spanNames)

	results := AnalysisResults{
		Duration:            dur,
		P95:                 p95,
		ErrorRate:           errorRate,
		SpanNames:           spanNames,
		NewSpanContexts:     newCtxs,
		RemovedSpanContexts: removedCtxs,
	}

	report := generateReport(cfg, results)
	fmt.Println(report)

	// Write to GitHub Actions step summary if running in CI.
	if summaryFile := os.Getenv("GITHUB_STEP_SUMMARY"); summaryFile != "" {
		if f, err := os.OpenFile(summaryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			fmt.Fprintln(f, report)
			f.Close()
		}
	}

	// Post as a PR comment if configured.
	if cfg.PRNumber != "" && cfg.GitHubRepo != "" && cfg.GitHubToken != "" {
		fmt.Printf("► Posting PR comment to %s#%s…\n", cfg.GitHubRepo, cfg.PRNumber)
		if err := postPRComment(cfg, report); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to post PR comment: %v\n", err)
		}
	}
}

// ── Shared fetch helper ───────────────────────────────────────────────────────

// fetchMetricsSeries runs a TraceQL metrics query against /api/metrics/query_range
// over the configured search window and returns the parsed response.
func fetchMetricsSeries(cfg Config, q string) (*tempopb.QueryRangeResponse, error) {
	now := time.Now()
	params := url.Values{}
	params.Set("q", q)
	params.Set("start", strconv.FormatInt(now.Add(-searchWindowHours*time.Hour).Unix(), 10))
	params.Set("end", strconv.FormatInt(now.Unix(), 10))
	params.Set("step", "60")

	body, err := tempoGET(cfg, cfg.TempoURL+"/api/metrics/query_range?"+params.Encode())
	if err != nil {
		return nil, err
	}

	resp := &tempopb.QueryRangeResponse{}
	if err := jsonpb.Unmarshal(bytes.NewReader(body), resp); err != nil {
		return nil, fmt.Errorf("parse response (raw: %s): %w", truncate(string(body), 300), err)
	}
	return resp, nil
}

// isTempoService returns true for Tempo's own internal services, excluding
// vulture (load tester) and the -all catch-all binary label.
func isTempoService(name string) bool {
	return strings.Contains(name, "tempo") &&
		!strings.Contains(name, "vulture") &&
		!strings.Contains(name, "-all")
}

// serviceVersionLabels extracts the (resource.service.version, rootServiceName)
// label pair from a series. Returns empty strings if either is missing.
func serviceVersionLabels(series *tempopb.TimeSeries) (version, service string) {
	for _, l := range series.Labels {
		switch l.Key {
		case "resource.service.version":
			version = l.Value.GetStringValue()
		case "rootServiceName":
			service = l.Value.GetStringValue()
		}
	}
	return
}

// avgSamples returns the average of all sample values in the series.
func avgSamples(series *tempopb.TimeSeries) (avg float64, n int) {
	for _, s := range series.Samples {
		avg += s.Value
	}
	n = len(series.Samples)
	if n > 0 {
		avg /= float64(n)
	}
	return
}

// ── Analysis 1: max duration ──────────────────────────────────────────────────

// fetchDurationStats runs `{} | max_over_time(span:duration) by (resource.service.version, rootServiceName)`
// and computes min/max/avg of the per-window max values. Values are in seconds in
// the API response; we convert to milliseconds.
func fetchDurationStats(cfg Config) (map[ServiceKey]DurationStats, error) {
	resp, err := fetchMetricsSeries(cfg, `{} | max_over_time(span:duration) by (resource.service.version, rootServiceName)`)
	if err != nil {
		return nil, fmt.Errorf("duration stats: %w", err)
	}

	result := make(map[ServiceKey]DurationStats, len(resp.Series))
	for _, series := range resp.Series {
		version, service := serviceVersionLabels(series)
		if version == "" || service == "" || !isTempoService(service) {
			continue
		}
		if len(series.Samples) == 0 {
			continue
		}

		minVal := math.MaxFloat64
		maxVal := -math.MaxFloat64
		sum := 0.0
		for _, s := range series.Samples {
			if s.Value < minVal {
				minVal = s.Value
			}
			if s.Value > maxVal {
				maxVal = s.Value
			}
			sum += s.Value
		}
		n := float64(len(series.Samples))
		// span:duration is in seconds in Tempo's metrics API (Prometheus convention).
		result[ServiceKey{version, service}] = DurationStats{
			MinMs:       minVal * 1000,
			MaxMs:       maxVal * 1000,
			AvgMs:       (sum / n) * 1000,
			SampleCount: len(series.Samples),
		}
	}
	return result, nil
}

// ── Analysis 2: p95 latency ───────────────────────────────────────────────────

// fetchP95Stats runs `{} | quantile_over_time(0.95, span:duration) by (resource.service.version, rootServiceName)`
// and averages the per-window p95 values to produce a stable overall p95 estimate.
func fetchP95Stats(cfg Config) (map[ServiceKey]P95Stats, error) {
	resp, err := fetchMetricsSeries(cfg, `{} | quantile_over_time(0.95, span:duration) by (resource.service.version, rootServiceName)`)
	if err != nil {
		return nil, fmt.Errorf("p95 stats: %w", err)
	}

	result := make(map[ServiceKey]P95Stats, len(resp.Series))
	for _, series := range resp.Series {
		version, service := serviceVersionLabels(series)
		if version == "" || service == "" || !isTempoService(service) {
			continue
		}
		avg, n := avgSamples(series)
		if n == 0 {
			continue
		}
		result[ServiceKey{version, service}] = P95Stats{
			AvgP95Ms:    avg * 1000, // seconds → ms
			SampleCount: n,
		}
	}
	return result, nil
}

// ── Analysis 3: error rate ────────────────────────────────────────────────────

// fetchAvgRateSeries runs a rate() query and returns the average rate per
// (version, service) key. Only Tempo service keys are included.
func fetchAvgRateSeries(cfg Config, q string) (map[ServiceKey]float64, error) {
	resp, err := fetchMetricsSeries(cfg, q)
	if err != nil {
		return nil, err
	}

	result := make(map[ServiceKey]float64, len(resp.Series))
	for _, series := range resp.Series {
		version, service := serviceVersionLabels(series)
		if version == "" || service == "" || !isTempoService(service) {
			continue
		}
		avg, n := avgSamples(series)
		if n == 0 {
			continue
		}
		result[ServiceKey{version, service}] = avg
	}
	return result, nil
}

// fetchErrorRates computes the error rate percentage per (version, service) by
// dividing the average error span rate by the average total span rate.
func fetchErrorRates(cfg Config) (map[ServiceKey]ErrorRateStats, error) {
	var totalRates, errorRates map[ServiceKey]float64
	var totalErr, errRateErr error
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		totalRates, totalErr = fetchAvgRateSeries(cfg, `{} | rate() by (resource.service.version, rootServiceName)`)
	}()
	go func() {
		defer wg.Done()
		errorRates, errRateErr = fetchAvgRateSeries(cfg, `{ status = error } | rate() by (resource.service.version, rootServiceName)`)
	}()
	wg.Wait()

	if totalErr != nil {
		return nil, fmt.Errorf("total rate query: %w", totalErr)
	}
	if errRateErr != nil {
		return nil, fmt.Errorf("error rate query: %w", errRateErr)
	}

	result := make(map[ServiceKey]ErrorRateStats, len(totalRates))
	for key, total := range totalRates {
		if total == 0 {
			continue
		}
		errPct := 0.0
		if errRate, ok := errorRates[key]; ok {
			errPct = (errRate / total) * 100
		}
		result[key] = ErrorRateStats{ErrorRatePct: errPct}
	}
	return result, nil
}

// ── Analysis 4: span name diff ────────────────────────────────────────────────

// fetchSpanNames runs `{} | rate() by (resource.service.version, span:name)` and
// returns the set of span names observed for the base and PR commits.
func fetchSpanNames(cfg Config) (map[string]map[string]struct{}, error) {
	resp, err := fetchMetricsSeries(cfg, `{rootServiceName =~ "tempo.*" && rootServiceName !~ "tempo-all|tempo-vulture"} | rate() by (resource.service.version, span:name)`)
	if err != nil {
		return nil, fmt.Errorf("span names: %w", err)
	}

	result := make(map[string]map[string]struct{})
	for _, series := range resp.Series {
		var version, spanName string
		for _, l := range series.Labels {
			switch l.Key {
			case "resource.service.version":
				version = l.Value.GetStringValue()
			case "name": // Tempo returns span:name as "name" in metrics label responses
				spanName = l.Value.GetStringValue()
			}
		}
		if version == "" || spanName == "" {
			continue
		}
		// Only track names for the two commits we're comparing.
		if version != cfg.GitCommit && version != cfg.BaseCommit {
			continue
		}
		if len(series.Samples) == 0 {
			continue
		}
		if result[version] == nil {
			result[version] = make(map[string]struct{})
		}
		result[version][spanName] = struct{}{}
	}
	return result, nil
}

// ── Report generation ─────────────────────────────────────────────────────────

func generateReport(cfg Config, results AnalysisResults) string {
	hasBase := cfg.BaseCommit != ""

	// Collect all unique service names across all result maps.
	serviceSet := make(map[string]struct{})
	for k := range results.Duration {
		serviceSet[k.Service] = struct{}{}
	}
	for k := range results.P95 {
		serviceSet[k.Service] = struct{}{}
	}
	for k := range results.ErrorRate {
		serviceSet[k.Service] = struct{}{}
	}
	services := make([]string, 0, len(serviceSet))
	for svc := range serviceSet {
		services = append(services, svc)
	}
	sort.Strings(services)

	var b strings.Builder

	fmt.Fprintf(&b, "## TraceGuard Analysis\n\n")
	fmt.Fprintf(&b, "**PR commit:** `%s`", cfg.GitCommit)
	if cfg.BaseCommit != "" {
		fmt.Fprintf(&b, "   **Base commit:** `%s`", cfg.BaseCommit)
	}
	fmt.Fprintf(&b, "\n\n")

	if len(services) == 0 {
		fmt.Fprintf(&b, "_No trace data found for the configured commits._\n\n")
		fmt.Fprintf(&b, "_Generated by [TraceGuard](https://github.com/grafana/tempo/tree/main/cmd/traceguard-analyze)_\n")
		return b.String()
	}

	for _, svc := range services {
		prKey := ServiceKey{cfg.GitCommit, svc}
		baseKey := ServiceKey{cfg.BaseCommit, svc}

		fmt.Fprintf(&b, "---\n\n### Service: `%s`\n\n", svc)

		// ── Duration ─────────────────────────────────────────────────────────
		prDur, hasPRDur := results.Duration[prKey]
		baseDur, hasBaseDur := results.Duration[baseKey]

		prDurConfident := !hasPRDur || prDur.SampleCount >= cfg.MinSamples
		baseDurConfident := !hasBaseDur || baseDur.SampleCount >= cfg.MinSamples
		if (hasPRDur || hasBaseDur) && prDurConfident && baseDurConfident {
			fmt.Fprintf(&b, "**Duration** (`max_over_time`)\n\n")
			writeCompareTable(&b, cfg, hasBase,
				hasPRDur, hasBaseDur,
				func() { // both
					fmt.Fprintf(&b, "| Metric | Base (`%s`) | PR (`%s`) | Delta |\n", shortSHA(cfg.BaseCommit), shortSHA(cfg.GitCommit))
					fmt.Fprintf(&b, "|--------|-------------|----------|-------|\n")
					fmt.Fprintf(&b, "| min (ms) | %.2f | %.2f | %s |\n", baseDur.MinMs, prDur.MinMs, formatDelta(prDur.MinMs-baseDur.MinMs, baseDur.MinMs))
					fmt.Fprintf(&b, "| max (ms) | %.2f | %.2f | %s |\n", baseDur.MaxMs, prDur.MaxMs, formatDelta(prDur.MaxMs-baseDur.MaxMs, baseDur.MaxMs))
					fmt.Fprintf(&b, "| avg (ms) | %.2f | %.2f | %s |\n", baseDur.AvgMs, prDur.AvgMs, formatDelta(prDur.AvgMs-baseDur.AvgMs, baseDur.AvgMs))
					fmt.Fprintf(&b, "| samples | %d | %d | — |\n", baseDur.SampleCount, prDur.SampleCount)
				},
				func() { // PR only
					fmt.Fprintf(&b, "| min (ms) | max (ms) | avg (ms) | samples |\n")
					fmt.Fprintf(&b, "|----------|----------|----------|---------|\n")
					fmt.Fprintf(&b, "| %.2f | %.2f | %.2f | %d |\n", prDur.MinMs, prDur.MaxMs, prDur.AvgMs, prDur.SampleCount)
				},
				func() { // base only
					fmt.Fprintf(&b, "| min (ms) | max (ms) | avg (ms) | samples |\n")
					fmt.Fprintf(&b, "|----------|----------|----------|---------|\n")
					fmt.Fprintf(&b, "| %.2f | %.2f | %.2f | %d |\n", baseDur.MinMs, baseDur.MaxMs, baseDur.AvgMs, baseDur.SampleCount)
				},
			)
			fmt.Fprintf(&b, "\n")
		}

		// ── p95 latency ───────────────────────────────────────────────────────
		prP95, hasPRP95 := results.P95[prKey]
		baseP95, hasBaseP95 := results.P95[baseKey]

		if hasPRP95 || hasBaseP95 {
			fmt.Fprintf(&b, "**p95 Latency** (`quantile_over_time(0.95)`)\n\n")
			writeCompareTable(&b, cfg, hasBase,
				hasPRP95, hasBaseP95,
				func() {
					delta := prP95.AvgP95Ms - baseP95.AvgP95Ms
					fmt.Fprintf(&b, "| Base (`%s`) | PR (`%s`) | Delta |\n", shortSHA(cfg.BaseCommit), shortSHA(cfg.GitCommit))
					fmt.Fprintf(&b, "|-------------|----------|-------|\n")
					warn := ""
					if baseP95.AvgP95Ms > 0 && delta/baseP95.AvgP95Ms > 0.20 {
						warn = " **[!]**"
					}
					fmt.Fprintf(&b, "| %.2f ms | %.2f ms | %s%s |\n", baseP95.AvgP95Ms, prP95.AvgP95Ms, formatDelta(delta, baseP95.AvgP95Ms), warn)
				},
				func() {
					fmt.Fprintf(&b, "| p95 (ms) |\n|----------|\n| %.2f |\n", prP95.AvgP95Ms)
				},
				func() {
					fmt.Fprintf(&b, "| p95 (ms) |\n|----------|\n| %.2f |\n", baseP95.AvgP95Ms)
				},
			)
			fmt.Fprintf(&b, "\n")
		}

		// ── Error rate ────────────────────────────────────────────────────────
		prErr, hasPRErr := results.ErrorRate[prKey]
		baseErr, hasBaseErr := results.ErrorRate[baseKey]

		if hasPRErr || hasBaseErr {
			fmt.Fprintf(&b, "**Error Rate**\n\n")
			writeCompareTable(&b, cfg, hasBase,
				hasPRErr, hasBaseErr,
				func() {
					delta := prErr.ErrorRatePct - baseErr.ErrorRatePct
					fmt.Fprintf(&b, "| Base (`%s`) | PR (`%s`) | Delta |\n", shortSHA(cfg.BaseCommit), shortSHA(cfg.GitCommit))
					fmt.Fprintf(&b, "|-------------|----------|-------|\n")
					warn := ""
					if delta > 1 {
						warn = " **[!]**"
					}
					sign := "+"
					if delta < 0 {
						sign = ""
					}
					fmt.Fprintf(&b, "| %.2f%% | %.2f%% | %s%.2f pp%s |\n", baseErr.ErrorRatePct, prErr.ErrorRatePct, sign, delta, warn)
				},
				func() {
					fmt.Fprintf(&b, "| error rate |\n|------------|\n| %.2f%% |\n", prErr.ErrorRatePct)
				},
				func() {
					fmt.Fprintf(&b, "| error rate |\n|------------|\n| %.2f%% |\n", baseErr.ErrorRatePct)
				},
			)
			fmt.Fprintf(&b, "\n")
		}
	}

	// ── Span name diff ────────────────────────────────────────────────────────
	if hasBase && results.SpanNames != nil {
		prNames := results.SpanNames[cfg.GitCommit]
		baseNames := results.SpanNames[cfg.BaseCommit]

		var newOps, removedOps []string
		for name := range prNames {
			if _, inBase := baseNames[name]; !inBase {
				newOps = append(newOps, name)
			}
		}
		for name := range baseNames {
			if _, inPR := prNames[name]; !inPR {
				removedOps = append(removedOps, name)
			}
		}
		sort.Strings(newOps)
		sort.Strings(removedOps)

		if len(newOps) > 0 || len(removedOps) > 0 {
			fmt.Fprintf(&b, "---\n\n### Changed Operations\n\n")
			if len(newOps) > 0 {
				fmt.Fprintf(&b, "**New in PR** (not seen in base):\n\n")
				writeOpsWithContext(&b, newOps, results.NewSpanContexts, "new operation")
			}
			if len(removedOps) > 0 {
				fmt.Fprintf(&b, "**Removed in PR** (seen in base, not in PR):\n\n")
				writeOpsWithContext(&b, removedOps, results.RemovedSpanContexts, "removed operation")
			}
		}
	}

	fmt.Fprintf(&b, "---\n\n")
	fmt.Fprintf(&b, "_Generated by [TraceGuard](https://github.com/grafana/tempo/tree/main/cmd/traceguard-analyze)_\n")

	return b.String()
}

// writeCompareTable calls one of three render functions depending on whether
// PR data, base data, or both are available. Emits a "no data" note otherwise.
func writeCompareTable(
	b *strings.Builder,
	cfg Config,
	hasBase, hasPR, hasBaseData bool,
	both, prOnly, baseOnly func(),
) {
	switch {
	case hasPR && hasBase && hasBaseData:
		both()
	case hasPR:
		prOnly()
		if hasBase {
			fmt.Fprintf(b, "_No data for base commit._\n")
		}
	case hasBase && hasBaseData:
		baseOnly()
		fmt.Fprintf(b, "_No data for PR commit._\n")
	}
}

// formatDelta returns a "+N.2ms (+P%)" string. Percentage is omitted when baseVal is 0.
func formatDelta(delta, baseVal float64) string {
	sign := "+"
	if delta < 0 {
		sign = ""
	}
	if baseVal != 0 {
		pct := delta / baseVal * 100
		return fmt.Sprintf("%s%.2f ms (%s%.0f%%)", sign, delta, sign, pct)
	}
	return fmt.Sprintf("%s%.2f ms", sign, delta)
}

// shortSHA returns the first 8 characters of a commit SHA.
func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

// ── Base commit detection ─────────────────────────────────────────────────────

var baselineVersionRe = regexp.MustCompile(`^(main|baseline)`)

// detectBaseCommit queries Tempo for all active service versions and returns
// the one matching "main.*" or "baseline.*" with the most recent sample timestamp.
// Returns empty string if none is found.
func detectBaseCommit(cfg Config) string {
	resp, err := fetchMetricsSeries(cfg, `{} | rate() by(resource.service.version)`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: base commit auto-detection failed: %v\n", err)
		return ""
	}

	// Track the latest sample timestamp seen per matching version.
	latestTS := make(map[string]int64)
	for _, series := range resp.Series {
		var version string
		for _, l := range series.Labels {
			if l.Key == "resource.service.version" {
				version = l.Value.GetStringValue()
				break
			}
		}
		if version == "" || !baselineVersionRe.MatchString(version) {
			continue
		}
		for _, s := range series.Samples {
			if s.TimestampMs > latestTS[version] {
				latestTS[version] = s.TimestampMs
			}
		}
	}

	var bestVersion string
	var bestTS int64 = -1
	for v, ts := range latestTS {
		if ts > bestTS {
			bestTS = ts
			bestVersion = v
		}
	}
	return bestVersion
}

// ── Span context ─────────────────────────────────────────────────────────────

const maxContextsPerSection = 5

// fetchSpanContexts computes the new/removed operation diff from spanNames and
// fetches a trace context tree for up to maxContextsPerSection spans in each
// direction, in parallel.
func fetchSpanContexts(cfg Config, spanNames map[string]map[string]struct{}) (newCtxs, removedCtxs map[string]string) {
	if cfg.BaseCommit == "" || spanNames == nil {
		return nil, nil
	}

	prNames := spanNames[cfg.GitCommit]
	baseNames := spanNames[cfg.BaseCommit]

	var newOps, removedOps []string
	for name := range prNames {
		if _, inBase := baseNames[name]; !inBase {
			newOps = append(newOps, name)
		}
	}
	for name := range baseNames {
		if _, inPR := prNames[name]; !inPR {
			removedOps = append(removedOps, name)
		}
	}
	sort.Strings(newOps)
	sort.Strings(removedOps)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); newCtxs = fetchSpanContextsBatch(cfg, newOps, cfg.GitCommit) }()
	go func() { defer wg.Done(); removedCtxs = fetchSpanContextsBatch(cfg, removedOps, cfg.BaseCommit) }()
	wg.Wait()
	return newCtxs, removedCtxs
}

// fetchSpanContextsBatch fetches span context trees for up to maxContextsPerSection
// span names in parallel, filtering traces by commitSHA.
func fetchSpanContextsBatch(cfg Config, names []string, commitSHA string) map[string]string {
	if commitSHA == "" || len(names) == 0 {
		return nil
	}
	if len(names) > maxContextsPerSection {
		names = names[:maxContextsPerSection]
	}

	type result struct {
		name string
		tree string
	}
	ch := make(chan result, len(names))
	for _, name := range names {
		go func(n string) {
			ch <- result{n, fetchOneSpanContext(cfg, n, commitSHA)}
		}(name)
	}

	out := make(map[string]string, len(names))
	for range names {
		if r := <-ch; r.tree != "" {
			out[r.name] = r.tree
		}
	}
	return out
}

// fetchOneSpanContext searches for a trace containing spanName at commitSHA,
// fetches the full trace, and returns a tree-structured context string.
func fetchOneSpanContext(cfg Config, spanName, commitSHA string) string {
	q := fmt.Sprintf(`{ span:name = %q && resource.service.version = %q }`, spanName, commitSHA)
	traceID, err := searchTraceByQuery(cfg, q)
	if err != nil || traceID == "" {
		return ""
	}
	spans, err := fetchTraceSpans(cfg, traceID)
	if err != nil || len(spans) == 0 {
		return ""
	}
	return renderSpanContext(spans, spanName)
}

// searchTraceByQuery runs a TraceQL search and returns the first trace ID found.
func searchTraceByQuery(cfg Config, q string) (string, error) {
	now := time.Now()
	params := url.Values{}
	params.Set("q", q)
	params.Set("limit", "1")
	params.Set("start", strconv.FormatInt(now.Add(-searchWindowHours*time.Hour).Unix(), 10))
	params.Set("end", strconv.FormatInt(now.Unix(), 10))

	body, err := tempoGET(cfg, cfg.TempoURL+"/api/search?"+params.Encode())
	if err != nil {
		return "", err
	}
	resp := &tempopb.SearchResponse{}
	if err := jsonpb.Unmarshal(bytes.NewReader(body), resp); err != nil {
		return "", fmt.Errorf("parse search response: %w", err)
	}
	if len(resp.Traces) == 0 {
		return "", nil
	}
	return resp.Traces[0].TraceID, nil
}

// fetchTraceSpans fetches a trace by ID and returns all spans flattened with
// service names extracted from resource attributes.
func fetchTraceSpans(cfg Config, traceID string) ([]flatSpan, error) {
	body, err := tempoGETProto(cfg, cfg.TempoURL+"/api/traces/"+traceID)
	if err != nil {
		return nil, err
	}
	trace := &tempopb.Trace{}
	if err := proto.Unmarshal(body, trace); err != nil {
		return nil, fmt.Errorf("unmarshal trace %s: %w", traceID, err)
	}

	var spans []flatSpan
	for _, rs := range trace.ResourceSpans {
		svcName := ""
		if rs.Resource != nil {
			for _, attr := range rs.Resource.Attributes {
				if attr.Key == "service.name" {
					svcName = attr.Value.GetStringValue()
					break
				}
			}
		}
		for _, ss := range rs.ScopeSpans {
			for _, s := range ss.Spans {
				dur := 0.0
				if s.EndTimeUnixNano > s.StartTimeUnixNano {
					dur = float64(s.EndTimeUnixNano-s.StartTimeUnixNano) / 1e6
				}
				spans = append(spans, flatSpan{
					SpanID:      hex.EncodeToString(s.SpanId),
					ParentID:    hex.EncodeToString(s.ParentSpanId),
					Name:        s.Name,
					ServiceName: svcName,
					DurationMs:  dur,
				})
			}
		}
	}
	return spans, nil
}

// renderSpanContext finds the first span matching targetName and renders a tree
// showing: the parent, all siblings (with target highlighted), and the target's
// direct children.
func renderSpanContext(spans []flatSpan, targetName string) string {
	// Index spans by ID for fast lookup.
	byID := make(map[string]*flatSpan, len(spans))
	for i := range spans {
		byID[spans[i].SpanID] = &spans[i]
	}

	// Build parent→children index.
	childrenOf := make(map[string][]int, len(spans))
	for i, s := range spans {
		childrenOf[s.ParentID] = append(childrenOf[s.ParentID], i)
	}

	// Find the first span with the target name.
	targetIdx := -1
	for i, s := range spans {
		if s.Name == targetName {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		return ""
	}

	target := &spans[targetIdx]
	parent := byID[target.ParentID]

	var b strings.Builder
	if parent == nil {
		// Target is a root span: show it and its children.
		writeSpanLine(&b, "", target, true)
		writeChildLines(&b, childrenOf, spans, target.SpanID, "")
	} else {
		// Show the parent at the top level.
		writeSpanLine(&b, "", parent, false)
		// Show all siblings, highlighting the target.
		sibs := childrenOf[parent.SpanID]
		for k, idx := range sibs {
			isLast := k == len(sibs)-1
			branch, cont := "├── ", "│   "
			if isLast {
				branch, cont = "└── ", "    "
			}
			sp := &spans[idx]
			isTarget := sp.SpanID == target.SpanID
			writeSpanLine(&b, branch, sp, isTarget)
			if isTarget {
				writeChildLines(&b, childrenOf, spans, sp.SpanID, cont)
			}
		}
	}
	return b.String()
}

// writeOpsWithContext lists operation names and, for those that have a fetched
// context tree, renders the tree in a code block so reviewers can see the
// span's position in the trace. label is shown in the tree marker (e.g. "new operation").
// A note is appended when more operations exist than were fetched (capped at maxContextsPerSection).
func writeOpsWithContext(b *strings.Builder, ops []string, contexts map[string]string, label string) {
	for _, name := range ops {
		fmt.Fprintf(b, "- `%s`\n", name)
		if tree, ok := contexts[name]; ok {
			// Annotate the marker in the tree with the label.
			tree = strings.ReplaceAll(tree, "← changed", "← "+label)
			fmt.Fprintf(b, "\n  ```\n")
			for _, line := range strings.Split(strings.TrimRight(tree, "\n"), "\n") {
				fmt.Fprintf(b, "  %s\n", line)
			}
			fmt.Fprintf(b, "  ```\n\n")
		}
	}
	if len(ops) > maxContextsPerSection {
		fmt.Fprintf(b, "_Context shown for first %d of %d operations._\n", maxContextsPerSection, len(ops))
	}
	fmt.Fprintf(b, "\n")
}

func writeSpanLine(b *strings.Builder, prefix string, s *flatSpan, isTarget bool) {
	svc := s.ServiceName
	if svc == "" {
		svc = "?"
	}
	marker := ""
	if isTarget {
		marker = "  ← changed"
	}
	fmt.Fprintf(b, "%s[%s] %s (%.0fms)%s\n", prefix, svc, s.Name, s.DurationMs, marker)
}

func writeChildLines(b *strings.Builder, childrenOf map[string][]int, spans []flatSpan, parentID, indent string) {
	kids := childrenOf[parentID]
	for k, idx := range kids {
		isLast := k == len(kids)-1
		branch := indent + "├── "
		if isLast {
			branch = indent + "└── "
		}
		writeSpanLine(b, branch, &spans[idx], false)
	}
}

// ── HTTP ──────────────────────────────────────────────────────────────────────

func tempoGET(cfg Config, rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if cfg.Auth != "" {
		req.Header.Set("Authorization", cfg.Auth)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(b), 200))
	}
	return b, nil
}

// tempoGETProto is like tempoGET but sets Accept: application/protobuf, which
// is required by /api/traces/{id} since OTLP JSON lost backwards compatibility.
func tempoGETProto(cfg Config, rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if cfg.Auth != "" {
		req.Header.Set("Authorization", cfg.Auth)
	}
	req.Header.Set("Accept", "application/protobuf")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(b), 200))
	}
	return b, nil
}

// ── GitHub ───────────────────────────────────────────────────────────────────

func postPRComment(cfg Config, body string) error {
	parts := strings.SplitN(cfg.GitHubRepo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("GITHUB_REPOSITORY must be in owner/repo format, got %q", cfg.GitHubRepo)
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%s/comments",
		parts[0], parts[1], cfg.PRNumber)

	payload, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+cfg.GitHubToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API HTTP %d: %s", resp.StatusCode, truncate(string(b), 300))
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
