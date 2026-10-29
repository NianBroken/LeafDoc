#!/usr/bin/env bash
# Runs the backend and Vite on the shared fixed ports; Ctrl-C stops both.
# Vite proxies API and resource requests to the backend. Credentials and
# MySQL configuration come from the environment or the private root .env.
# Arguments go to the backend, e.g.
# `make dev ARGS=--enable-revision=true`. bash, not sh: dash turns job
# control (set -m below) off without a terminal and would leave Vite running.
set -eu

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
UI_DIR="$ROOT/ui"

for tool in go node npm; do
    if ! command -v "$tool" >/dev/null 2>&1; then
        echo "$tool not found; 'make doctor' shows what is missing." >&2
        exit 1
    fi
done

FRONTEND_PORT="$(node -p 'require(process.argv[1]).frontend' "$ROOT/internal/core/serviceports/ports.json")"
BACKEND_PORT="$(node -p 'require(process.argv[1]).backend' "$ROOT/internal/core/serviceports/ports.json")"

# Export private assignments for the CLI without publishing their values.
if [ -f "$ROOT/.env" ]; then
    set -a
    . "$ROOT/.env"
    set +a
fi

# Development sessions receive a fresh signing key when none is configured.
if [ -z "${LEAFDOC_JWT_SECRET:-}" ]; then
    LEAFDOC_JWT_SECRET="$(node -e 'process.stdout.write(require("node:crypto").randomBytes(64).toString("base64"))')"
    export LEAFDOC_JWT_SECRET
fi

node "$ROOT/scripts/release-service-ports.mjs" frontend backend

if [ ! -d "$UI_DIR/node_modules" ]; then
    echo "Frontend dependencies are missing; run 'make setup' first." >&2
    exit 1
fi

# Build instead of `go run`: signalling `go run` reaches only the wrapper, not
# the server it starts (see tests/e2e/run.sh), and compile errors show up before
# Vite starts.
tmp="$(mktemp -d)"
bin="$tmp/leafdoc$(cd "$ROOT" && go env GOEXE)"
echo "Building the backend..."
if ! (cd "$ROOT" && go build -o "$bin" ./cmd/leafdoc); then
    rm -rf "$tmp"
    exit 1
fi

backend=""
frontend=""
cleanup() {
    trap - EXIT INT TERM
    for pid in $frontend $backend; do
        # The whole process group (npm -> vite -> esbuild), else the process.
        kill -TERM -- "-$pid" 2>/dev/null || kill -TERM "$pid" 2>/dev/null || true
    done
    for pid in $frontend $backend; do
        wait "$pid" 2>/dev/null || true
    done
    rm -rf "$tmp"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# Job control puts each background job into its own process group, so cleanup
# can stop everything it spawned. Off again before the wait loop: with it on,
# the loop's foreground `sleep` would get the terminal and with it Ctrl-C.
# Both read /dev/null, a background process group reading the terminal stops.
set -m
(cd "$ROOT" && exec "$bin" --port="$BACKEND_PORT" --allow-insecure=true --enable-revision --enable-api-key-management --snapshot "$@") </dev/null &
backend=$!
(cd "$UI_DIR" && exec npm run dev) </dev/null &
frontend=$!
set +m

echo "Backend on http://localhost:$BACKEND_PORT, frontend on http://localhost:$FRONTEND_PORT (Ctrl-C stops both)."
while kill -0 "$backend" 2>/dev/null && kill -0 "$frontend" 2>/dev/null; do
    sleep 1
done
echo "The backend or the frontend exited; stopping the other." >&2
exit 1
