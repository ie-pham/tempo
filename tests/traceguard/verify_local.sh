#!/usr/bin/env bash
# verify_local.sh — step-by-step local verification for traceguard-analyze.
#
# Run each step independently so you can make code changes between the
# baseline and PR test runs and see a real diff in the report.
#
# Usage:
#   bash tests/traceguard/verify_local.sh <command>
#
# Commands:
#   start     Start the Docker Compose stack and write state to ARTIFACTS_DIR
#   baseline  Run the baseline integration tests and wait for trace ingestion
#   pr        Run the PR integration tests and wait for trace ingestion
#   analyze   Build the analyzer, diff the two runs, validate the report
#   stop      Tear down the Docker Compose stack
#   all       Run all steps in sequence (start → baseline → pr → analyze → stop)
#
# Typical workflow with a code change between the two runs:
#
#   bash tests/traceguard/verify_local.sh start
#   bash tests/traceguard/verify_local.sh baseline
#
#   # … make your changes, then rebuild the Tempo image …
#   make docker-tempo docker-tempo-query
#
#   bash tests/traceguard/verify_local.sh pr
#   bash tests/traceguard/verify_local.sh analyze
#   bash tests/traceguard/verify_local.sh stop
#
# Prerequisites:
#   • Go toolchain in PATH
#   • Docker daemon running
#   • Run from the repository root
#
# Environment variables (all optional):
#
#   MIN_SAMPLES       minimum trace count threshold (default 10)
#   ARTIFACTS_DIR     directory for state and generated files
#                     (default /tmp/traceguard-local-verify)
#   TIMEOUT_SECS      seconds to wait for trace ingestion per commit (default 180)
#   BASELINE_TARGET   make target for the baseline run (default test-e2e-limits)
#   PR_TARGET         make target for the PR run       (default test-e2e-limits)
#   OTLP_HOST         host reachable from test containers (default host.docker.internal)
#                     Override on Linux: OTLP_HOST=$(ip route show default | awk '{print $3}')
#
# Exit code: 0 = success, non-zero = failure.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$REPO_ROOT"

# ── Static configuration ───────────────────────────────────────────────────────
ARTIFACTS_DIR="${ARTIFACTS_DIR:-/tmp/traceguard-local-verify}"
MIN_SAMPLES="${MIN_SAMPLES:-10}"
TIMEOUT_SECS="${TIMEOUT_SECS:-180}"
BASELINE_TARGET="${BASELINE_TARGET:-test-e2e-api}"
PR_TARGET="${PR_TARGET:-test-e2e-api}"

POLL_INTERVAL=10
COMPOSE_FILE="$REPO_ROOT/example/docker-compose/single-binary/docker-compose.yaml"
COMPOSE_PROJECT="traceguard-verify"
TEMPO_QUERY_URL="http://localhost:3200"
OTLP_HOST="${OTLP_HOST:-host.docker.internal}"
OTLP_ENDPOINT="http://${OTLP_HOST}:4318"

STATE_FILE="$ARTIFACTS_DIR/state.env"

# ── Helpers ────────────────────────────────────────────────────────────────────

load_state() {
    if [ ! -f "$STATE_FILE" ]; then
        echo "ERROR: no state file found at $STATE_FILE"
        echo "       Run 'start' first."
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
                    -G \
                    --data-urlencode "q={ resource.service.version = \"$commit\" && resource.service.name = \"$svc\" }" \
                    --data-urlencode "limit=$MIN_SAMPLES" \
                    "$TEMPO_QUERY_URL/api/search" 2>/dev/null
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

cmd_start() {
    echo "══════════════════════════════════════════════════════"
    echo "  TraceGuard — start"
    echo "══════════════════════════════════════════════════════"

    mkdir -p "$ARTIFACTS_DIR"

    # Generate unique commit tags for this session and persist them.
    local run_id="verify-local-$(date +%Y%m%d-%H%M%S)-$$"
    BASE_COMMIT="base-${run_id}"
    GIT_COMMIT="pr-${run_id}"
    save_state

    echo "  base commit : $BASE_COMMIT"
    echo "  PR commit   : $GIT_COMMIT"
    echo "  OTLP ingest : $OTLP_ENDPOINT"
    echo "  query URL   : $TEMPO_QUERY_URL"
    echo "  artifacts   : $ARTIFACTS_DIR"
    echo ""

    # Tear down any stale stack from a previous run.
    docker compose -f "$COMPOSE_FILE" -p "$COMPOSE_PROJECT" down --remove-orphans 2>/dev/null || true

    echo "► Starting Docker Compose stack…"
    docker compose -f "$COMPOSE_FILE" -p "$COMPOSE_PROJECT" up -d

    echo "     Stack started. Waiting for Tempo readiness…"
    local deadline=$(( $(date +%s) + 90 ))
    until curl -sf "$TEMPO_QUERY_URL/ready" >/dev/null 2>&1; do
        if [ "$(date +%s)" -ge "$deadline" ]; then
            echo "ERROR: Tempo did not become ready within 90 s."
            docker compose -f "$COMPOSE_FILE" -p "$COMPOSE_PROJECT" logs tempo
            exit 1
        fi
        sleep 2
    done

    echo "     Tempo is ready at $TEMPO_QUERY_URL"
    echo ""
    echo "Next step: bash tests/traceguard/verify_local.sh baseline"
}

cmd_baseline() {
    load_state
    echo "══════════════════════════════════════════════════════"
    echo "  TraceGuard — baseline run"
    echo "══════════════════════════════════════════════════════"
    echo "  commit tag    : $BASE_COMMIT"
    echo "  make target   : $BASELINE_TARGET"
    echo "  OTLP target   : $OTLP_ENDPOINT"
    echo ""

    echo "► Running baseline integration tests…"
    OTEL_EXPORTER_OTLP_ENDPOINT="$OTLP_ENDPOINT" \
    OTEL_EXPORTER_OTLP_PROTOCOL="http/protobuf" \
    OTEL_RESOURCE_ATTRIBUTES="service.version=$BASE_COMMIT" \
        make "$BASELINE_TARGET"

    echo ""
    echo "► Waiting for baseline trace ingestion (up to ${TIMEOUT_SECS}s)…"
    poll_until_ready "$BASE_COMMIT" "base"
    poll_until_ready "$BASE_COMMIT" "base"

    echo ""
    echo "Baseline complete. You can now make code changes and rebuild:"
    echo "  make docker-tempo docker-tempo-query"
    echo ""
    echo "Next step: bash tests/traceguard/verify_local.sh pr"
}

cmd_pr() {
    load_state
    echo "══════════════════════════════════════════════════════"
    echo "  TraceGuard — PR run"
    echo "══════════════════════════════════════════════════════"
    echo "  commit tag    : $GIT_COMMIT"
    echo "  make target   : $PR_TARGET"
    echo "  OTLP target   : $OTLP_ENDPOINT"
    echo ""

    echo "► Running PR integration tests…"
    OTEL_EXPORTER_OTLP_ENDPOINT="$OTLP_ENDPOINT" \
    OTEL_EXPORTER_OTLP_PROTOCOL="http/protobuf" \
    OTEL_RESOURCE_ATTRIBUTES="service.version=$GIT_COMMIT" \
        make "$PR_TARGET"

    echo ""
    echo "► Waiting for PR trace ingestion (up to ${TIMEOUT_SECS}s)…"
    poll_until_ready "$GIT_COMMIT" "PR"
    poll_until_ready "$GIT_COMMIT" "PR"

    echo ""
    echo "Next step: bash tests/traceguard/verify_local.sh analyze"
}

cmd_analyze() {
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

    # GRAFANA_SERVICE_TOKEN must be non-empty to pass config validation;
    # the local Tempo instance does not enforce auth so any value works.
    local raw_output
    raw_output=$(
        MODE=remote \
        GIT_COMMIT="$GIT_COMMIT" \
        BASE_COMMIT="$BASE_COMMIT" \
        GRAFANA_TEMPO_QUERY_URL="$TEMPO_QUERY_URL" \
        GRAFANA_SERVICE_TOKEN="local-no-auth" \
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
    echo "Next step: bash tests/traceguard/verify_local.sh stop"
}

cmd_stop() {
    echo "══════════════════════════════════════════════════════"
    echo "  TraceGuard — stop"
    echo "══════════════════════════════════════════════════════"
    echo ""
    echo "► Tearing down Docker Compose stack…"
    docker compose -f "$COMPOSE_FILE" -p "$COMPOSE_PROJECT" down --remove-orphans
    echo "     Done."

    if [ -f "$STATE_FILE" ]; then
        rm "$STATE_FILE"
        echo "     State file removed."
    fi
}

cmd_all() {
    cmd_start
    echo ""
    cmd_baseline
    echo ""
    cmd_pr
    echo ""
    cmd_analyze
    echo ""
    cmd_stop
}

# ── Dispatch ───────────────────────────────────────────────────────────────────

COMMAND="${1:-all}"

case "$COMMAND" in
    start)    cmd_start    ;;
    baseline) cmd_baseline ;;
    pr)       cmd_pr       ;;
    analyze)  cmd_analyze  ;;
    stop)     cmd_stop     ;;
    all)      cmd_all      ;;
    *)
        echo "Unknown command: $COMMAND"
        echo "Usage: $0 {start|baseline|pr|analyze|stop|all}"
        exit 1
        ;;
esac
