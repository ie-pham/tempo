#!/usr/bin/env bash
# verify_remote.sh — step-by-step remote verification for traceguard-analyze.
#
# Mirrors verify_local.sh but targets a remote Grafana Cloud Tempo instance
# instead of a local Docker Compose stack.  Grafana Alloy runs on the host as
# a local OTLP collector; Tempo containers push traces to Alloy (no auth
# needed), and Alloy forwards them to Grafana Cloud with Basic auth.
#
# ── What you need ─────────────────────────────────────────────────────────────
#
# To EMIT traces (Tempo containers → Alloy → Grafana Cloud):
#
#   OTEL_EXPORTER_OTLP_ENDPOINT
#       HTTP OTLP gateway URL for your Grafana Cloud stack, e.g.:
#         https://otlp-gateway-prod-us-east-0.grafana.net/otlp
#       Find it at: grafana.com → your stack → OpenTelemetry → OTLP endpoint
#       Used by Alloy when forwarding traces to Grafana Cloud.
#
#   GRAFANA_USER
#       Your Grafana Cloud instance ID (numeric), e.g.: 1234
#       Used by Alloy as the Basic-auth username.
#
#   GRAFANA_SERVICE_TOKEN
#       A Grafana Cloud API token with "traces:write" + "traces:read" scope.
#       Used by Alloy as the Basic-auth password and by the analyzer for queries.
#
# To QUERY traces (traceguard-analyze reads from Grafana Cloud Tempo):
#
#   GRAFANA_TEMPO_QUERY_URL
#       Your Tempo query base URL, e.g.:
#         https://tempo-prod-05-prod-us-central-0.grafana.net/tempo
#       Find it at: grafana.com → your stack → Tempo details
#
# ── Usage ─────────────────────────────────────────────────────────────────────
#
#   bash tests/traceguard/verify_remote.sh <command>
#
# Commands:
#   check     Validate env vars and ping the remote Tempo endpoint
#   baseline  Run the baseline integration tests and wait for trace ingestion
#   pr        Run the PR integration tests and wait for trace ingestion
#   analyze   Build the analyzer, diff the two runs, validate the report
#   cleanup   Remove local state files (the remote stack is untouched)
#   all       Run all steps in sequence (check → baseline → pr → analyze → cleanup)
#
# Typical workflow with a code change between the two runs:
#
#   export OTEL_EXPORTER_OTLP_ENDPOINT=https://otlp-gateway-prod-us-east-0.grafana.net/otlp
#   export GRAFANA_BASIC_AUTH="Basic <base64>"
#   export GRAFANA_TEMPO_QUERY_URL=https://tempo-prod-05-prod-us-central-0.grafana.net/tempo
#   export GRAFANA_SERVICE_TOKEN=glsa_xxxxxxxxxxxxxxxxxxxx
#   export GRAFANA_USER=1234
#
#   bash tests/traceguard/verify_remote.sh check
#   bash tests/traceguard/verify_remote.sh baseline
#
#   # … make your changes …
#
#   bash tests/traceguard/verify_remote.sh pr
#   bash tests/traceguard/verify_remote.sh analyze
#   bash tests/traceguard/verify_remote.sh cleanup
#
# Prerequisites:
#   • Go toolchain in PATH
#   • curl in PATH
#   • Grafana Alloy in PATH  (https://grafana.com/docs/alloy/latest/get-started/install/)
#   • Run from the repository root
#
# Environment variables (all optional unless marked required):
#
#   OTEL_EXPORTER_OTLP_ENDPOINT   [required] HTTP OTLP gateway URL (used by Alloy)
#   GRAFANA_USER                  [required] Grafana Cloud instance ID
#   GRAFANA_SERVICE_TOKEN         [required] API token for Alloy auth + Tempo queries
#   GRAFANA_BASIC_AUTH            [required] Basic <base64> for Tempo query polling
#   GRAFANA_TEMPO_QUERY_URL       [required] Tempo query base URL
#   OTLP_HOST                     host reachable from test containers (default host.docker.internal)
#                                 Override on Linux: OTLP_HOST=$(ip route show default | awk '{print $3}')
#   ALLOY_CONFIG                  path to Alloy config file (default tests/traceguard/config.alloy)
#   MIN_SAMPLES                   minimum trace count threshold (default 10)
#   ARTIFACTS_DIR                 directory for state and report files
#                                 (default /tmp/traceguard-remote-verify)
#   TIMEOUT_SECS                  seconds to wait for trace ingestion (default 300)
#   BASELINE_TARGET               make target for baseline run (default test-e2e-api)
#   PR_TARGET                     make target for PR run       (default test-e2e-api)
#
# Exit code: 0 = success, non-zero = failure.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$REPO_ROOT"

# ── Static configuration ───────────────────────────────────────────────────────
ARTIFACTS_DIR="${ARTIFACTS_DIR:-/tmp/traceguard-remote-verify}"
MIN_SAMPLES="${MIN_SAMPLES:-10}"
TIMEOUT_SECS="${TIMEOUT_SECS:-300}"
BASELINE_TARGET="${BASELINE_TARGET:-test-e2e-api}"
PR_TARGET="${PR_TARGET:-test-e2e-api}"
OTLP_HOST="${OTLP_HOST:-host.docker.internal}"
ALLOY_CONFIG="${ALLOY_CONFIG:-$REPO_ROOT/tests/traceguard/config.alloy}"

POLL_INTERVAL=15
STATE_FILE="$ARTIFACTS_DIR/state.env"
ALLOY_PID_FILE="$ARTIFACTS_DIR/alloy.pid"

# ── Helpers ────────────────────────────────────────────────────────────────────

require_env() {
    local var="$1"
    if [ -z "${!var:-}" ]; then
        echo "ERROR: $var is not set."
        echo "       See the header of this script for setup instructions."
        exit 1
    fi
}

check_required_env() {
    require_env OTEL_EXPORTER_OTLP_ENDPOINT
    require_env GRAFANA_USER
    require_env GRAFANA_SERVICE_TOKEN
    require_env GRAFANA_BASIC_AUTH
    require_env GRAFANA_TEMPO_QUERY_URL
}

load_state() {
    if [ ! -f "$STATE_FILE" ]; then
        echo "ERROR: no state file found at $STATE_FILE"
        echo "       Run 'check' first."
        exit 1
    fi
    # shellcheck source=/dev/null
    source "$STATE_FILE"
}

save_state() {
    mkdir -p "$ARTIFACTS_DIR"
    cat > "$STATE_FILE" <<EOF
BASE_COMMIT="${BASE_COMMIT}"
GIT_COMMIT="${GIT_COMMIT}"
EOF
    echo "     State → $STATE_FILE"
}

REQUIRED_SERVICES=("tempo-distributor" "tempo-live-store")

poll_until_ready() {
    local commit="$1"
    local label="$2"
    local deadline=$(( $(date +%s) + TIMEOUT_SECS ))

    while [ "$(date +%s)" -lt "$deadline" ]; do
        local all_ready=1
        for svc in "${REQUIRED_SERVICES[@]}"; do
            local http_code
            http_code=$(
                curl -s -o "$ARTIFACTS_DIR/poll_${svc}.json" -w "%{http_code}" \
                    -H "Accept: application/json" \
                    -H "Authorization: ${GRAFANA_BASIC_AUTH}" \
                    -G \
                    --data-urlencode "q={ resource.service.version = \"$commit\" && resource.service.name = \"$svc\" }" \
                    --data-urlencode "limit=$MIN_SAMPLES" \
                    "${GRAFANA_TEMPO_QUERY_URL}/api/search" 2>/dev/null
            ) || true

            local found=0
            if [ "$http_code" = "200" ]; then
                found=$(python3 -c "
import json
d = json.load(open('$ARTIFACTS_DIR/poll_${svc}.json'))
print(len(d.get('traces', [])))
" 2>/dev/null || echo "0")
            else
                echo "     [$label] $svc: Tempo returned HTTP $http_code"
            fi

            echo "     [$label] $svc: $found / $MIN_SAMPLES traces"
            if [ "$found" -lt "$MIN_SAMPLES" ]; then
                all_ready=0
            fi
        done

        if [ "$all_ready" = "1" ]; then
            echo "     [$label] All services reached threshold."
            return 0
        fi

        sleep "$POLL_INTERVAL"
    done

    echo "WARNING: [$label] not all services reached $MIN_SAMPLES traces after ${TIMEOUT_SECS}s."
    echo "         Proceeding anyway (analyzer has low-confidence logic)."
}

# ── Commands ───────────────────────────────────────────────────────────────────

cmd_check() {
    echo "══════════════════════════════════════════════════════"
    echo "  TraceGuard — check"
    echo "══════════════════════════════════════════════════════"

    check_required_env

    echo "  OTLP endpoint   : ${OTEL_EXPORTER_OTLP_ENDPOINT}"
    echo "  Tempo query URL : ${GRAFANA_TEMPO_QUERY_URL}"
    echo "  Alloy config    : ${ALLOY_CONFIG}"
    echo "  artifacts       : ${ARTIFACTS_DIR}"
    echo ""

    if ! command -v alloy &>/dev/null; then
        echo "ERROR: 'alloy' not found in PATH."
        echo "       Install Grafana Alloy: https://grafana.com/docs/alloy/latest/get-started/install/"
        exit 1
    fi

    echo "► Pinging remote Tempo /ready…"
    local http_code
    http_code=$(
        curl -s -o /dev/null -w "%{http_code}" \
            -H "Authorization: ${GRAFANA_BASIC_AUTH}" \
            "${GRAFANA_TEMPO_QUERY_URL}/api/search?q=%7B%7D" 2>/dev/null
    ) || true
    if [ "$http_code" != "200" ]; then
        echo "ERROR: ${GRAFANA_TEMPO_QUERY_URL}/api/search?q=%7B%7D returned HTTP ${http_code}."
        echo "       Check GRAFANA_TEMPO_QUERY_URL and GRAFANA_SERVICE_TOKEN."
        exit 1
    fi
    echo "     Remote Tempo is reachable (HTTP 200)."
    echo ""

    mkdir -p "$ARTIFACTS_DIR"

    # Generate unique commit tags for this session and persist them.
    local run_id="verify-remote-$(date +%Y%m%d-%H%M%S)-$$"
    BASE_COMMIT="baseline-${run_id}"
    GIT_COMMIT="pr-${run_id}"
    save_state

    echo "► Starting Grafana Alloy (local OTLP collector → Grafana Cloud)…"
    OTEL_EXPORTER_OTLP_ENDPOINT="${OTEL_EXPORTER_OTLP_ENDPOINT}" \
    GRAFANA_USER="${GRAFANA_USER}" \
    GRAFANA_SERVICE_TOKEN="${GRAFANA_SERVICE_TOKEN}" \
        alloy run "${ALLOY_CONFIG}" > "$ARTIFACTS_DIR/alloy.log" 2>&1 &
    echo $! > "$ALLOY_PID_FILE"
    echo "     Alloy PID → $(cat "$ALLOY_PID_FILE")  log → $ARTIFACTS_DIR/alloy.log"

    echo "     Waiting for Alloy to be ready on :4318…"
    local deadline=$(( $(date +%s) + 30 ))
    until curl -sf http://localhost:4318 >/dev/null 2>&1 || [ "$(date +%s)" -ge "$deadline" ]; do
        if ! kill -0 "$(cat "$ALLOY_PID_FILE")" 2>/dev/null; then
            echo "ERROR: Alloy process exited unexpectedly. Check $ARTIFACTS_DIR/alloy.log"
            exit 1
        fi
        sleep 1
    done
    if ! curl -sf http://localhost:4318 >/dev/null 2>&1; then
        echo "WARNING: Alloy did not respond on :4318 within 30s — traces may not be forwarded."
        echo "         Check $ARTIFACTS_DIR/alloy.log for errors."
    else
        echo "     Alloy is ready."
    fi

    echo ""
    echo "  base commit : $BASE_COMMIT"
    echo "  PR commit   : $GIT_COMMIT"
    echo "  OTLP via    : http://${OTLP_HOST}:4318 → Alloy → Grafana Cloud"
    echo ""
    echo "Next step: bash tests/traceguard/verify_remote.sh baseline"
}

cmd_baseline() {
    check_required_env
    load_state
    echo "══════════════════════════════════════════════════════"
    echo "  TraceGuard — baseline run"
    echo "══════════════════════════════════════════════════════"
    echo "  commit tag    : $BASE_COMMIT"
    echo "  make target   : $BASELINE_TARGET"
    echo "  OTLP target   : http://${OTLP_HOST}:4318 (Alloy)"
    echo ""

    echo "► Running baseline integration tests…"
    OTEL_EXPORTER_OTLP_ENDPOINT="http://${OTLP_HOST}:4318" \
    OTEL_EXPORTER_OTLP_PROTOCOL="http/protobuf" \
    OTEL_RESOURCE_ATTRIBUTES="service.version=$BASE_COMMIT" \
        make "$BASELINE_TARGET"

    echo ""
    echo "► Waiting for baseline trace ingestion (up to ${TIMEOUT_SECS}s)…"
    poll_until_ready "$BASE_COMMIT" "base"

    echo ""
    echo "Baseline complete. You can now make code changes, then:"
    echo "Next step: bash tests/traceguard/verify_remote.sh pr"
}

cmd_pr() {
    check_required_env
    load_state
    echo "══════════════════════════════════════════════════════"
    echo "  TraceGuard — PR run"
    echo "══════════════════════════════════════════════════════"
    echo "  commit tag    : $GIT_COMMIT"
    echo "  make target   : $PR_TARGET"
    echo "  OTLP target   : http://${OTLP_HOST}:4318 (Alloy)"
    echo ""

    echo "► Running PR integration tests…"
    OTEL_EXPORTER_OTLP_ENDPOINT="http://${OTLP_HOST}:4318" \
    OTEL_EXPORTER_OTLP_PROTOCOL="http/protobuf" \
    OTEL_RESOURCE_ATTRIBUTES="service.version=$GIT_COMMIT" \
        make "$PR_TARGET"

    echo ""
    echo "► Waiting for PR trace ingestion (up to ${TIMEOUT_SECS}s)…"
    poll_until_ready "$GIT_COMMIT" "PR"

    echo ""
    echo "Next step: bash tests/traceguard/verify_remote.sh analyze"
}

cmd_analyze() {
    check_required_env
    load_state
    echo "══════════════════════════════════════════════════════"
    echo "  TraceGuard — analyze"
    echo "══════════════════════════════════════════════════════"
    echo "  base commit : $BASE_COMMIT"
    echo "  PR commit   : $GIT_COMMIT"
    echo ""

    echo "► Building traceguard-analyze…"
    go build -o "$ARTIFACTS_DIR/traceguard-analyze" ./cmd/traceguard-analyze
    echo "     Binary → $ARTIFACTS_DIR/traceguard-analyze"

    echo ""
    echo "► Running traceguard-analyze (PR vs base diff)…"

    local raw_output
    raw_output=$(
        MODE=remote \
        GIT_COMMIT="$GIT_COMMIT" \
        BASE_COMMIT="$BASE_COMMIT" \
        GRAFANA_TEMPO_QUERY_URL="${GRAFANA_TEMPO_QUERY_URL}" \
        GRAFANA_SERVICE_TOKEN="${GRAFANA_SERVICE_TOKEN}" \
        GRAFANA_USER="${GRAFANA_USER}" \
        MIN_SAMPLES="$MIN_SAMPLES" \
        "$ARTIFACTS_DIR/traceguard-analyze" 2>&1
    ) || {
        echo "WARNING: analyzer exited non-zero. Output:"
        echo "$raw_output"
    }

    echo "$raw_output"

    echo ""
    echo "► Validating report…"
    awk '/^## TraceGuard/{found=1} found{print}' <<< "$raw_output" > "$ARTIFACTS_DIR/report.md"

    if [ ! -s "$ARTIFACTS_DIR/report.md" ]; then
        echo "ERROR: no '## TraceGuard' heading found in analyzer output."
        exit 1
    fi

    echo "     Report → $ARTIFACTS_DIR/report.md"

    echo ""
    echo "══════════════════════════════════════════════════════"
    echo "  PASS — analysis complete"
    echo "══════════════════════════════════════════════════════"
    echo ""
    echo "Next step: bash tests/traceguard/verify_remote.sh cleanup"
}

cmd_cleanup() {
    echo "══════════════════════════════════════════════════════"
    echo "  TraceGuard — cleanup"
    echo "══════════════════════════════════════════════════════"
    echo ""
    echo "  (No containers to stop — remote stack is untouched.)"
    echo ""

    if [ -f "$ALLOY_PID_FILE" ]; then
        local alloy_pid
        alloy_pid=$(cat "$ALLOY_PID_FILE")
        if kill -0 "$alloy_pid" 2>/dev/null; then
            echo "► Stopping Grafana Alloy (PID $alloy_pid)…"
            kill "$alloy_pid"
            echo "     Alloy stopped."
        fi
        rm "$ALLOY_PID_FILE"
    fi

    if [ -f "$STATE_FILE" ]; then
        rm "$STATE_FILE"
        echo "     State file removed: $STATE_FILE"
    else
        echo "     No state file found — nothing to remove."
    fi
}

cmd_all() {
    cmd_check
    echo ""
    cmd_baseline
    echo ""
    cmd_pr
    echo ""
    cmd_analyze
    echo ""
    cmd_cleanup
}

# ── Dispatch ───────────────────────────────────────────────────────────────────

COMMAND="${1:-all}"

case "$COMMAND" in
    check)   cmd_check   ;;
    baseline) cmd_baseline ;;
    pr)       cmd_pr       ;;
    analyze)  cmd_analyze  ;;
    cleanup)  cmd_cleanup  ;;
    all)      cmd_all      ;;
    *)
        echo "Unknown command: $COMMAND"
        echo "Usage: $0 {check|baseline|pr|analyze|cleanup|all}"
        exit 1
        ;;
esac
