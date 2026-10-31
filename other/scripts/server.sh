#!/usr/bin/env bash
# 本脚本在列出的 Ubuntu 版本上安装或管理后端与 Caddy 的 systemd 服务。
# 程序、配置和数据使用各自系统目录，卸载仅移除程序与服务单元，保留配置和业务数据。
set -euo pipefail
# 未传动作时查看状态，系统路径集中声明供安装与卸载分支共用。
command_name="${1:-status}"
install_dir=/usr/local/lib/leafdoc
config_dir=/etc/leafdoc
data_dir=/var/lib/leafdoc
unit_dir=/etc/systemd/system
# 安装、服务状态变更和卸载要求管理员权限，status 分支不调用此检查。
require_root() {
    if [ "$(id -u)" -ne 0 ]; then echo 'Run as root.' >&2; exit 1; fi
}
case "$command_name" in
install)
    require_root
    if [ "$#" -ne 7 ]; then echo 'Usage: server.sh install BINARY CADDY ENV_FILE CERT KEY HTTPS_HOST' >&2; exit 1; fi
    # 输入文件先解析为实际路径，主机文本只接受列出的地址字符，随后进入配置模板。
    binary="$(realpath "$2")"
    gateway="$(realpath "$3")"
    environment="$(realpath "$4")"
    certificate="$(realpath "$5")"
    private_key="$(realpath "$6")"
    hostname="$7"
    case "$hostname" in *[!a-zA-Z0-9.:-]*|'') echo 'Invalid HTTPS host.' >&2; exit 1;; esac
    . /etc/os-release
    case "$ID:$VERSION_ID" in ubuntu:18.04|ubuntu:20.04|ubuntu:22.04) ;; *) echo 'Supported: Ubuntu 18.04, 20.04, 22.04.' >&2; exit 1;; esac
    # 缺少端口检查、健康检查工具或系统证书包时，通过系统包管理器安装。
    if ! command -v lsof >/dev/null || ! command -v curl >/dev/null || [ ! -s /etc/ssl/certs/ca-certificates.crt ]; then
        apt-get update
        DEBIAN_FRONTEND=noninteractive apt-get install -y lsof ca-certificates curl
    fi
    # 服务账号使用禁止交互登录的系统用户，已有同名账号直接复用。
    if ! id leafdoc >/dev/null 2>&1; then useradd --system --home-dir "$data_dir" --shell /usr/sbin/nologin leafdoc; fi
    install -d -m 755 "$install_dir"
    install -d -m 750 -o root -g leafdoc "$config_dir"
    install -d -m 750 -o leafdoc -g leafdoc "$data_dir"
    # 为 QUIC 写入接收与发送缓冲区上限，并立即加载该系统配置。
    printf 'net.core.rmem_max=7500000\nnet.core.wmem_max=7500000\n' > /etc/sysctl.d/60-leafdoc-quic.conf
    sysctl -p /etc/sysctl.d/60-leafdoc-quic.conf
    # 二进制可执行，配置与私钥只向 root 和服务组开放；证书可公开读取。
    install -m 755 "$binary" "$install_dir/leafdoc"
    install -m 755 "$gateway" "$install_dir/caddy"
    install -m 640 -o root -g leafdoc "$environment" "$config_dir/environment"
    install -m 644 "$certificate" "$config_dir/server.crt"
    install -m 640 -o root -g leafdoc "$private_key" "$config_dir/server.key"
    # 固定服务端口从所安装应用的共享端口入口取得，不在部署脚本重复维护数值。
    frontend="$("$install_dir/leafdoc" service-port frontend)"
    backend="$("$install_dir/leafdoc" service-port backend)"
    admin="$("$install_dir/leafdoc" service-port gatewayAdmin)"
    # 配置使用提供的证书，Caddy 暴露 HTTPS 并反向代理本机后端。
    # 以下 heredoc 是生成配置的原始内容，输出字节参与部署行为。
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
    # 后端单元使用专用账号运行，启动前释放后端与指标监听，失败时由 systemd 重启。
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
    # 网关单元依赖网络和后端，同样由 systemd 管理，模板文本保持为服务配置内容。
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
    # 安装后重新加载单元并启用开机运行，实际重启两个服务后轮询后端健康接口。
    systemctl daemon-reload
    systemctl enable leafdoc.service leafdoc-gateway.service
    systemctl restart leafdoc.service leafdoc-gateway.service
    ready=false
    for attempt in {1..120}; do
        if curl --fail --silent "http://127.0.0.1:$backend/api/health" >/dev/null; then ready=true; break; fi
        sleep 1
    done
    # 最多进行一百二十轮后端检查，失败输出后端日志并退出。
    # 这里不访问 HTTPS 页面、不检查网关证书，也不执行管理员登录。
    if [ "$ready" != true ]; then journalctl -u leafdoc.service -n 60 --no-pager; exit 1; fi
    echo "HTTPS frontend: https://$hostname:$frontend"
    ;;
# 服务控制同时作用于后端和网关，实际执行结果由 systemctl 返回。
start|stop|restart)
    require_root
    systemctl "$command_name" leafdoc.service leafdoc-gateway.service
    ;;
status)
    systemctl status leafdoc.service leafdoc-gateway.service --no-pager
    ;;
# 卸载停止并禁用服务，再移除单元和程序，数据与私有配置目录保持存在。
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
