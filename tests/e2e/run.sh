#!/usr/bin/env bash

set -euo pipefail

current_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$current_dir/../.." && pwd)"
# The wrapper creates and drops one owned test database before any app process starts.
if [ -z "${LEAFDOC_E2E_DATABASE_OWNED:-}" ]; then
  exec node "$current_dir/run-mysql.mjs" "$@"
fi
if [[ ! "$LEAFDOC_E2E_DATABASE_OWNED" =~ ^[0-9a-f]{24}$ ]] ||
   [ "${LEAFDOC_MYSQL_DATABASE:-}" != "leafdoc_test_migration_$LEAFDOC_E2E_DATABASE_OWNED" ]; then
  echo "The browser runner requires its own isolated MySQL database." >&2
  exit 1
fi
app_port="$(node -p 'require(process.argv[1]).backend' "$repo_root/internal/core/serviceports/ports.json")"
app_url="http://localhost:${app_port}"
run_mode="${E2E_RUN_MODE:-docker}"
server_pid=""
server_log=""
server_bin=""
local_data_dir=""
docker_data_volume=""

print_runner_diagnostics() {
  echo "--- E2E runtime diagnostics ---"

  if [ "$run_mode" = "docker" ]; then
    if docker ps -a --format '{{.Names}}' | grep -q '^wiki-e2e-tests$'; then
      echo "--- docker logs (last 200 lines) ---"
      docker logs --tail 200 wiki-e2e-tests 2>&1 || true
    else
      echo "No Docker container logs available."
    fi
    return
  fi

  if [ -n "$server_log" ] && [ -f "$server_log" ]; then
    echo "--- local server log (last 200 lines) ---"
    tail -n 200 "$server_log" || true
  else
    echo "No local server log available."
  fi
}

build_frontend_for_local_e2e() {
  if [ "${E2E_SKIP_UI_BUILD:-0}" = "1" ]; then
    echo "⚡ Skipping UI build for local E2E run..."
  else
    echo "🔨 Building frontend for local E2E run..."
    (
      cd "$repo_root/ui"
      npm run build
    )
  fi

  if [ ! -f "$repo_root/ui/dist/index.html" ]; then
    echo "❌ Frontend build output is missing at ui/dist/index.html"
    exit 1
  fi

  rm -rf "$repo_root/internal/http/dist"
  mkdir -p "$repo_root/internal/http/dist"
  cp -R "$repo_root/ui/dist/." "$repo_root/internal/http/dist/"
  touch "$repo_root/internal/http/dist/.gitkeep"
}

start_docker() {
  echo "🟢 Starting Docker container..."
  docker build \
    --build-arg RATE_LIMIT_BYPASS=true \
    -t wiki-e2e-tests \
    "$repo_root"

  if docker ps -a --format '{{.Names}}' | grep -q '^wiki-e2e-tests$'; then
    echo "⚠️ Removing existing container..."
    docker rm -f wiki-e2e-tests >/dev/null 2>&1 || true
  fi

  docker_data_volume="wiki-e2e-tests-data-${RANDOM}${RANDOM}"
  docker volume create "$docker_data_volume" >/dev/null

  docker run -d \
    -p "$app_port:$app_port" \
    --name wiki-e2e-tests \
    -e LEAFDOC_MYSQL_HOST -e LEAFDOC_MYSQL_PORT -e LEAFDOC_MYSQL_DATABASE \
    -e LEAFDOC_MYSQL_USER -e LEAFDOC_MYSQL_PASSWORD \
    -e LEAFDOC_ADMIN_USERNAME -e LEAFDOC_ADMIN_PASSWORD -e LEAFDOC_JWT_SECRET \
    -v "$docker_data_volume":/app/data \
    wiki-e2e-tests \
    --allow-insecure=true \
    --enable-revision=true \
    --enable-link-refactor=true \
    --revision-coalesce-window=0

  echo "✅ Container started on $app_url"
}

stop_docker() {
  echo "🛑 Stopping Docker container..."
  docker stop wiki-e2e-tests >/dev/null 2>&1 || true
  docker rm wiki-e2e-tests >/dev/null 2>&1 || true
  docker rmi wiki-e2e-tests >/dev/null 2>&1 || true
  if [ -n "$docker_data_volume" ]; then
    docker volume rm "$docker_data_volume" >/dev/null 2>&1 || true
  fi
}

start_local() {
  echo "🟢 Starting local LeafDoc process..."
  build_frontend_for_local_e2e

  local_data_dir="$(mktemp -d /tmp/leafdoc-e2e-data.XXXXXX)"
  server_log="$(mktemp /tmp/leafdoc-e2e-server.XXXXXX.log)"
  server_bin="$(mktemp /tmp/leafdoc-e2e-bin.XXXXXX)"

  # Build the binary first, then run it directly in the background. Using
  # `go run` here would make `$!` the PID of the `go run` wrapper, not of the
  # compiled server it execs as a child — so `stop_local`'s `kill` would leave
  # the real server orphaned, holding the port after the script exits.
  echo "🔨 Building leafdoc binary for local E2E run..."
  (
    cd "$repo_root"
    go build \
      -ldflags="-X github.com/NianBroken/LeafDoc/internal/http.EmbedFrontend=true -X github.com/NianBroken/LeafDoc/internal/http.Environment=production -X github.com/NianBroken/LeafDoc/internal/wiki/auth.DisableRefreshTokenRateLimit=true" \
      -o "$server_bin" \
      ./cmd/leafdoc
  )

  (
    cd "$repo_root"
    exec "$server_bin" \
      --host 127.0.0.1 \
      --port "$app_port" \
      --data-dir "$local_data_dir" \
      --allow-insecure=true \
      --enable-revision=true \
      --enable-link-refactor=true \
      --revision-coalesce-window=0
  ) >"$server_log" 2>&1 &

  server_pid=$!
  echo "✅ Local process started on $app_url (pid $server_pid)"
}

stop_local() {
  echo "🛑 Stopping local LeafDoc process..."
  if [ -n "$server_pid" ] && kill -0 "$server_pid" >/dev/null 2>&1; then
    kill "$server_pid" >/dev/null 2>&1 || true
    wait "$server_pid" >/dev/null 2>&1 || true
  fi
  if [ -n "$local_data_dir" ] && [ -d "$local_data_dir" ]; then
    rm -rf "$local_data_dir"
  fi

  if [ -n "$server_log" ] && [ -f "$server_log" ]; then
    rm -f "$server_log"
  fi

  if [ -n "$server_bin" ] && [ -f "$server_bin" ]; then
    rm -f "$server_bin"
  fi
}

stop_runner() {
  if [ "$run_mode" = "docker" ]; then
    stop_docker
  else
    stop_local
  fi
}

cleanup_runner() {
  local exit_code=$?

  if [ "$exit_code" -ne 0 ]; then
    print_runner_diagnostics
  fi

  stop_runner
  exit "$exit_code"
}

run_playwright_tests() {
  echo "Running Playwright tests..."
  (
    cd "$current_dir"
    local reporter="${E2E_PLAYWRIGHT_REPORTER:-line}"
    local workers="${E2E_PLAYWRIGHT_WORKERS:-1}"

    if command -v stdbuf >/dev/null 2>&1; then
      E2E_BASE_URL="$app_url" \
      E2E_ADMIN_USER="${E2E_ADMIN_USER:-admin}" \
      E2E_ADMIN_PASSWORD="${E2E_ADMIN_PASSWORD:-admine2epassword}" \
      PLAYWRIGHT_FORCE_TTY=1 \
      stdbuf -oL -eL npx playwright test --workers="$workers" --reporter="$reporter" "$@"
    else
      E2E_BASE_URL="$app_url" \
      E2E_ADMIN_USER="${E2E_ADMIN_USER:-admin}" \
      E2E_ADMIN_PASSWORD="${E2E_ADMIN_PASSWORD:-admine2epassword}" \
      PLAYWRIGHT_FORCE_TTY=1 \
      npx playwright test --workers="$workers" --reporter="$reporter" "$@"
    fi
  )
}

wait_until_reachable() {
  local max_attempts=60
  local attempt=0

  until curl -s "$app_url" >/dev/null; do
    printf '.'
    sleep 2
    attempt=$((attempt + 1))
    if [ "$attempt" -ge "$max_attempts" ]; then
      echo
      echo "❌ LeafDoc is not reachable after 2 minutes."
      if [ "$run_mode" = "local" ] && [ -f "$server_log" ]; then
        echo "--- local server log ---"
        tail -n 200 "$server_log" || true
      fi
      exit 1
    fi
  done

  echo
  echo "✅ LeafDoc is reachable."
}

if nc -z localhost "$app_port" >/dev/null 2>&1; then
  if docker ps -a --format '{{.Names}}' | grep -q '^wiki-e2e-tests$'; then
    echo "⚠️ Port $app_port already in use by an existing E2E container – restarting it..."
    stop_docker
  else
    echo "❌ Fixed port $app_port remained occupied after reclaiming its owner. No alternate port is permitted."
    exit 1
  fi
fi

trap cleanup_runner EXIT
if [ "$run_mode" = "docker" ]; then
  start_docker
else
  start_local
fi

wait_until_reachable
run_playwright_tests "$@"
