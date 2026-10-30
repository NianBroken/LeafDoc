#!/usr/bin/env sh
set -eu

# Enable debug mode if DEBUG environment variable is set to "true"
if [ "${DEBUG:-false}" = "true" ]; then
  set -x
fi

log() {
  # Simple log format with timestamp
  printf '[entrypoint] %s\n' "$*" >&2
}

log "Starting entrypoint script..."

# Check if --host argument is not provided and the LEAFDOC_HOST environment variable is not set.
# if both are missing, set LEAFDOC_HOST to "0.0.0.0"
has_host_arg=false
for arg in "$@"; do
  case "$arg" in
    --host|--host=* )
      has_host_arg=true
      break
      ;;
  esac
done

if [ "$has_host_arg" = false ] && [ -z "${LEAFDOC_HOST:-}" ]; then
  log "No --host argument or LEAFDOC_HOST environment variable found. Setting LEAFDOC_HOST to '0.0.0.0'"
  export LEAFDOC_HOST="0.0.0.0"
fi

# start the main application and pass all arguments to it
log "Executing /app/leafdoc"
exec /app/leafdoc "$@"
