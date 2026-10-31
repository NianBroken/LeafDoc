#!/usr/bin/env bash
# Verify Compose restarts the LeafDoc application after its container process exits.
set -euo pipefail

project=leafdoc
service=app
expected_executable=/app/leafdoc
container_ids=()

mapfile -t container_ids < <(docker ps --quiet --filter "label=com.docker.compose.project=$project" --filter "label=com.docker.compose.service=$service")
if [ "${#container_ids[@]}" -ne 1 ]; then
    echo 'Expected exactly one running LeafDoc application container.' >&2
    exit 1
fi

container_id="${container_ids[0]}"
restart_policy="$(docker inspect --format '{{.HostConfig.RestartPolicy.Name}}' "$container_id")"
restart_count="$(docker inspect --format '{{.RestartCount}}' "$container_id")"
container_pid="$(docker inspect --format '{{.State.Pid}}' "$container_id")"
executable="$(readlink "/proc/$container_pid/exe")"
cgroup="$(cat "/proc/$container_pid/cgroup")"

if [ "$restart_policy" != unless-stopped ] || [ "$executable" != "$expected_executable" ] || [[ "$cgroup" != *"$container_id"* ]]; then
    echo 'The selected process is not the protected LeafDoc Compose application.' >&2
    exit 1
fi

# Killing this verified container process simulates an unexpected application exit.
kill -KILL "$container_pid"
for attempt in {1..120}; do
    status="$(docker inspect --format '{{.State.Status}} {{.State.Health.Status}} {{.RestartCount}}' "$container_id")"
    read -r state health recovered_count <<< "$status"
    if [ "$state" = running ] && [ "$health" = healthy ] && [ "$recovered_count" -gt "$restart_count" ]; then
        echo 'LeafDoc application recovered after an unexpected process exit.'
        exit 0
    fi
    sleep 2
done

echo "LeafDoc application did not recover after its process exited: $status" >&2
exit 1
