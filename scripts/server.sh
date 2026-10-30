#!/usr/bin/env bash
# Installs and manages the static release on Ubuntu 18.04, 20.04 and 22.04.
# Secrets remain in system configuration and user data survives uninstall.
set -euo pipefail
command_name="${1:-status}"
install_dir=/usr/local/lib/leafdoc
config_dir=/etc/leafdoc
data_dir=/var/lib/leafdoc
unit_dir=/etc/systemd/system
require_root() {
    if [ "$(id -u)" -ne 0 ]; then echo 'Run as root.' >&2; exit 1; fi
}
case "$command_name" in
install)
    require_root
    if [ "$#" -ne 7 ]; then echo 'Usage: server.sh install BINARY CADDY ENV_FILE CERT KEY HTTPS_HOST' >&2; exit 1; fi
    binary="$(realpath "$2")"
    gateway="$(realpath "$3")"
    environment="$(realpath "$4")"
    certificate="$(realpath "$5")"
    private_key="$(realpath "$6")"
    hostname="$7"
    case "$hostname" in *[!a-zA-Z0-9.:-]*|'') echo 'Invalid HTTPS host.' >&2; exit 1;; esac
    . /etc/os-release
    case "$ID:$VERSION_ID" in ubuntu:18.04|ubuntu:20.04|ubuntu:22.04) ;; *) echo 'Supported: Ubuntu 18.04, 20.04, 22.04.' >&2; exit 1;; esac
    if ! command -v lsof >/dev/null || ! command -v curl >/dev/null || [ ! -s /etc/ssl/certs/ca-certificates.crt ]; then
        apt-get update
        DEBIAN_FRONTEND=noninteractive apt-get install -y lsof ca-certificates curl
    fi
    if ! id leafdoc >/dev/null 2>&1; then useradd --system --home-dir "$data_dir" --shell /usr/sbin/nologin leafdoc; fi
    install -d -m 755 "$install_dir"
    install -d -m 750 -o root -g leafdoc "$config_dir"
    install -d -m 750 -o leafdoc -g leafdoc "$data_dir"
    # QUIC needs sufficient kernel socket buffers on all supported Ubuntu kernels.
    printf 'net.core.rmem_max=7500000\nnet.core.wmem_max=7500000\n' > /etc/sysctl.d/60-leafdoc-quic.conf
    sysctl -p /etc/sysctl.d/60-leafdoc-quic.conf
    install -m 755 "$binary" "$install_dir/leafdoc"
    install -m 755 "$gateway" "$install_dir/caddy"
    install -m 640 -o root -g leafdoc "$environment" "$config_dir/environment"
    install -m 644 "$certificate" "$config_dir/server.crt"
    install -m 640 -o root -g leafdoc "$private_key" "$config_dir/server.key"
    frontend="$("$install_dir/leafdoc" service-port frontend)"
    backend="$("$install_dir/leafdoc" service-port backend)"
    admin="$("$install_dir/leafdoc" service-port gatewayAdmin)"
    cat > "$config_dir/Caddyfile" <<EOF
{
    auto_https off
    admin 127.0.0.1:$admin
}
https://$hostname:$frontend {
    bind 0.0.0.0
    tls $config_dir/server.crt $config_dir/server.key
    reverse_proxy 127.0.0.1:$backend
}
EOF
    "$install_dir/caddy" fmt --overwrite "$config_dir/Caddyfile"
    cat > "$unit_dir/leafdoc.service" <<EOF
# The service manager reclaims permanent listeners with system privileges.
[Unit]
Description=LeafDoc backend
Wants=network-online.target
After=network-online.target
StartLimitIntervalSec=60
StartLimitBurst=5
[Service]
Type=simple
User=leafdoc
Group=leafdoc
PermissionsStartOnly=true
WorkingDirectory=$data_dir
EnvironmentFile=$config_dir/environment
ExecStartPre=$install_dir/leafdoc release-ports backend metrics
ExecStart=$install_dir/leafdoc --host=127.0.0.1 --data-dir=$data_dir --enable-revision --snapshot
Restart=on-failure
RestartSec=5
TimeoutStopSec=30
UMask=0027
[Install]
WantedBy=multi-user.target
EOF
    cat > "$unit_dir/leafdoc-gateway.service" <<EOF
# The gateway exposes HTTPS and a local administration listener.
[Unit]
Description=LeafDoc HTTPS gateway
Wants=network-online.target leafdoc.service
After=network-online.target leafdoc.service
StartLimitIntervalSec=60
StartLimitBurst=5
[Service]
Type=simple
User=leafdoc
Group=leafdoc
PermissionsStartOnly=true
WorkingDirectory=$data_dir
Environment=HOME=$data_dir
ExecStartPre=$install_dir/leafdoc release-ports frontend gatewayAdmin
ExecStart=$install_dir/caddy run --config $config_dir/Caddyfile
Restart=on-failure
RestartSec=5
TimeoutStopSec=30
UMask=0027
[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable leafdoc.service leafdoc-gateway.service
    systemctl restart leafdoc.service leafdoc-gateway.service
    ready=false
    for attempt in {1..120}; do
        if curl --fail --silent "http://127.0.0.1:$backend/api/health" >/dev/null; then ready=true; break; fi
        sleep 1
    done
    if [ "$ready" != true ]; then journalctl -u leafdoc.service -n 60 --no-pager; exit 1; fi
    echo "HTTPS frontend: https://$hostname:$frontend"
    ;;
start|stop|restart)
    require_root
    systemctl "$command_name" leafdoc.service leafdoc-gateway.service
    ;;
status)
    systemctl status leafdoc.service leafdoc-gateway.service --no-pager
    ;;
uninstall)
    require_root
    systemctl disable --now leafdoc-gateway.service leafdoc.service
    rm -f "$unit_dir/leafdoc.service" "$unit_dir/leafdoc-gateway.service"
    systemctl daemon-reload
    rm -f "$install_dir/leafdoc" "$install_dir/caddy"
    rmdir "$install_dir"
    echo "Data and private configuration retained in $data_dir and $config_dir."
    ;;
*)
    echo 'Commands: install, start, stop, restart, status, uninstall' >&2
    exit 1
    ;;
esac
