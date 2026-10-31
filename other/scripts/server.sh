#!/usr/bin/env bash
# Ubuntu 原生服务安装器，程序与数据库均由 systemd 管理，配置和业务数据独立保留。
set -euo pipefail
. "$(dirname "$0")/lib/deployment.sh"
script_dir="$(cd -- "$(dirname -- "$0")" && pwd -P)"
action="${1:-status}"
install_dir=/usr/local/lib/leafdoc
config_dir="${LEAFDOC_CONFIG_DIR:-/etc/leafdoc}"
data_dir=/var/lib/leafdoc
mysql_dir=/usr/local/mysql
mysql_data=/var/lib/leafdoc-mysql
units=(leafdoc-mysql.service leafdoc.service leafdoc-gateway.service)
case "$action" in
install)
    leafdoc_require_root
    [ "$#" = 5 ] || { echo 'Usage: server.sh install BINARY CADDY MYSQL_ARCHIVE CONFIG_DIRECTORY' >&2; exit 1; }
    binary="$(realpath "$2")"; caddy="$(realpath "$3")"; archive="$(realpath "$4")"; config_dir="$(realpath "$5")"
    . /etc/os-release
    case "$ID:$VERSION_ID" in ubuntu:18.04|ubuntu:20.04|ubuntu:22.04) ;; *) echo 'Supported Ubuntu releases: 18.04, 20.04, 22.04.' >&2; exit 1;; esac
    test -f "$binary" && test -f "$caddy" && test -f "$archive"
    leafdoc_load_environment "$config_dir/environment"
    apt-get update
    DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl libaio1 libnuma1 libncurses5 libtinfo5
    if ! id leafdoc >/dev/null 2>&1; then useradd --system --home-dir "$data_dir" --shell /usr/sbin/nologin leafdoc; fi
    if ! id leafdoc-mysql >/dev/null 2>&1; then useradd --system --home-dir "$mysql_data" --shell /usr/sbin/nologin leafdoc-mysql; fi
    install -d -m 755 "$install_dir"
    install -d -m 750 -o leafdoc -g leafdoc "$data_dir"
    install -d -m 750 -o leafdoc-mysql -g leafdoc-mysql "$mysql_data"
    install -m 755 "$binary" "$install_dir/leafdoc"
    install -m 755 "$caddy" "$install_dir/caddy"
    if [ ! -x "$mysql_dir/bin/mysqld" ]; then
        install -d -m 755 "$mysql_dir"
        tar -xf "$archive" -C "$mysql_dir" --strip-components=1
    fi
    # 依赖缺失直接失败并保留诊断，不能进入部分可用状态。
    if ldd "$mysql_dir/bin/mysqld" | grep 'not found'; then exit 1; fi
    chown -R root:leafdoc "$config_dir"
    chmod 750 "$config_dir" "$config_dir/tls"
    chmod 640 "$config_dir/environment" "$config_dir"/tls/*.key
    frontend="$("$install_dir/leafdoc" service-port frontend)"
    backend="$("$install_dir/leafdoc" service-port backend)"
    cat > "$config_dir/mysql.cnf" <<EOF
# Dedicated LeafDoc database, accessible only inside this Ubuntu server.
[mysqld]
basedir=$mysql_dir
datadir=$mysql_data
socket=$mysql_data/mysql.sock
pid-file=$mysql_data/mysql.pid
bind-address=127.0.0.1
port=$LEAFDOC_MYSQL_PORT
mysqlx=OFF
log-error=$mysql_data/mysql.log
innodb_buffer_pool_size=128M
character-set-server=utf8mb4
collation-server=utf8mb4_unicode_ci
EOF
    if [ ! -d "$mysql_data/mysql" ]; then
        "$mysql_dir/bin/mysqld" --defaults-file="$config_dir/mysql.cnf" --initialize --user=leafdoc-mysql
    fi
    cat > /etc/systemd/system/leafdoc-mysql.service <<EOF
# Dedicated native MySQL service.
[Unit]
Description=LeafDoc MySQL
After=network.target
[Service]
Type=simple
User=leafdoc-mysql
Group=leafdoc-mysql
ExecStart=$mysql_dir/bin/mysqld --defaults-file=$config_dir/mysql.cnf
Restart=on-failure
RestartSec=5
TimeoutStopSec=120
LimitNOFILE=10000
[Install]
WantedBy=multi-user.target
EOF
    # 数据库账号读取该文件所需的组权限不授予应用私有配置。
    chown root:leafdoc-mysql "$config_dir/mysql.cnf"
    chmod 644 "$config_dir/mysql.cnf"
    chmod 755 "$config_dir"
    systemctl daemon-reload
    systemctl enable --now leafdoc-mysql.service
    if [ ! -f "$config_dir/.database-initialized" ]; then
        error_log="$mysql_data/mysql.log"
        for attempt in {1..30}; do
            temporary_password="$(sed -n 's/.*A temporary password is generated for root@localhost: //p' "$error_log" | tail -n 1)"
            [ -n "$temporary_password" ] && break
            sleep 1
        done
        [ -n "${temporary_password:-}" ] || { journalctl -u leafdoc-mysql.service -n 60 --no-pager; exit 1; }
        for attempt in {1..120}; do
            [ -S "$mysql_data/mysql.sock" ] && break
            sleep 1
        done
        MYSQL_PWD="$temporary_password" "$mysql_dir/bin/mysql" --defaults-file="$config_dir/mysql.cnf" --protocol=SOCKET --socket="$mysql_data/mysql.sock" -u root --connect-expired-password <<SQL
ALTER USER 'root'@'localhost' IDENTIFIED BY '$LEAFDOC_MYSQL_ROOT_PASSWORD';
CREATE DATABASE IF NOT EXISTS \`$LEAFDOC_MYSQL_DATABASE\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS '$LEAFDOC_MYSQL_USER'@'127.0.0.1' IDENTIFIED BY '$LEAFDOC_MYSQL_PASSWORD';
GRANT ALL ON \`$LEAFDOC_MYSQL_DATABASE\`.* TO '$LEAFDOC_MYSQL_USER'@'127.0.0.1';
SQL
        unset temporary_password MYSQL_PWD
        touch "$config_dir/.database-initialized"
        chmod 600 "$config_dir/.database-initialized"
    fi
    ready=false
    for attempt in {1..120}; do
        if "$mysql_dir/bin/mysqladmin" --no-defaults --socket="$mysql_data/mysql.sock" ping >/dev/null 2>&1; then ready=true; break; fi
        sleep 1
    done
    [ "$ready" = true ] || { journalctl -u leafdoc-mysql.service -n 60 --no-pager; exit 1; }
    # 凭据由生成器给出十六进制随机值，标识符只接受字母、数字和下划线。
    [[ "$LEAFDOC_MYSQL_DATABASE:$LEAFDOC_MYSQL_USER" =~ ^[a-zA-Z0-9_]+:[a-zA-Z0-9_]+$ ]]
    [[ "$LEAFDOC_MYSQL_PASSWORD:$LEAFDOC_MYSQL_ROOT_PASSWORD" =~ ^[a-f0-9]+:[a-f0-9]+$ ]]
    cat > /etc/systemd/system/leafdoc.service <<EOF
# Native LeafDoc application with bounded database startup recovery.
[Unit]
Description=LeafDoc backend
Wants=network-online.target
Requires=leafdoc-mysql.service
After=network-online.target leafdoc-mysql.service
[Service]
Type=simple
User=leafdoc
Group=leafdoc
WorkingDirectory=$data_dir
EnvironmentFile=$config_dir/environment
ExecStart=$install_dir/leafdoc --host=127.0.0.1 --data-dir=$data_dir --enable-revision --snapshot
Restart=on-failure
RestartSec=5
TimeoutStopSec=45
UMask=0027
NoNewPrivileges=true
[Install]
WantedBy=multi-user.target
EOF
    "$install_dir/caddy" fmt --overwrite "$config_dir/Caddyfile"
    "$install_dir/caddy" validate --config "$config_dir/Caddyfile"
    cat > /etc/systemd/system/leafdoc-gateway.service <<EOF
# HTTPS gateway keeps certificate storage in the dedicated data directory.
[Unit]
Description=LeafDoc HTTPS gateway
Wants=network-online.target leafdoc.service
After=network-online.target leafdoc.service
[Service]
Type=simple
User=leafdoc
Group=leafdoc
Environment=HOME=$data_dir
WorkingDirectory=$data_dir
ExecStart=$install_dir/caddy run --config $config_dir/Caddyfile
Restart=on-failure
RestartSec=5
TimeoutStopSec=30
AmbientCapabilities=CAP_NET_BIND_SERVICE
UMask=0027
NoNewPrivileges=true
[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable "${units[@]}"
    systemctl restart leafdoc.service leafdoc-gateway.service
    python3 "$script_dir/verify-deployment.py" --url "https://localhost:$frontend" --ca "$config_dir/tls/local-ca.crt" --environment "$config_dir/environment"
    ;;
start|stop|restart)
    leafdoc_require_root
    systemctl "$action" "${units[@]}"
    ;;
status) systemctl status "${units[@]}" --no-pager;;
uninstall)
    leafdoc_require_root
    systemctl disable --now leafdoc-gateway.service leafdoc.service leafdoc-mysql.service
    for unit in "${units[@]}"; do rm -f "/etc/systemd/system/$unit"; done
    systemctl daemon-reload
    echo 'Configuration, programs and databases retained for recovery.'
    ;;
*) echo 'Commands: install, start, stop, restart, status, uninstall' >&2; exit 1;;
esac
