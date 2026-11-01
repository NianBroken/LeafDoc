#!/bin/sh
# 容器内实例准备入口，只处理显式挂载的配置、公开证书与应用数据。
set -eu
config_dir=${LEAFDOC_CONFIG_DIR:-/config}
data_dir=${LEAFDOC_INSTANCE_DATA:-/instance-data}
public_dir=${LEAFDOC_PUBLIC_DIR:-$config_dir/public}
frontend_port=${LEAFDOC_FRONTEND_PORT:?}
backend_port=${LEAFDOC_BACKEND_PORT:?}
hosts=${LEAFDOC_HOSTS:-}
domain=${LEAFDOC_DOMAIN:-}
base_path=${LEAFDOC_BASE_PATH:-}
bootstrap_port=${LEAFDOC_CA_PORT:-}
managed_gateway=${LEAFDOC_MANAGED_GATEWAY:-false}
ca_days=3650
certificate_days=825
retry_duration=10s
umask 077
fail() { printf '%s\n' "$*" >&2; exit 1; }
case "$base_path" in ''|/*) ;; *) fail '路径前缀必须以 / 开头';; esac
case "$base_path" in *[!A-Za-z0-9/_.~%-]*) fail '路径前缀包含无效字符';; esac
for port in "$frontend_port" "$backend_port" ${bootstrap_port:+"$bootstrap_port"}; do
  case "$port" in ''|*[!0-9]*) fail '端口必须为数字';; esac
  [ "$port" -ge 1 ] && [ "$port" -le 65535 ] || fail '端口超出范围'
done
case "$domain" in *[!A-Za-z0-9.-]*) fail '公网域名格式无效';; esac
mkdir -p "$config_dir/tls" "$public_dir" "$data_dir"
chmod 700 "$config_dir/tls" "$data_dir"
chmod 755 "$config_dir" "$public_dir"
uid=$(id -u leafdoc)
gid=$(id -g leafdoc)
if [ ! -e "$data_dir/setup-token" ] && [ ! -e "$data_dir/installation.json" ]; then
  openssl rand -hex 32 > "$data_dir/setup-token.next"
  mv "$data_dir/setup-token.next" "$data_dir/setup-token"
fi
chown -R "$uid:$gid" "$data_dir"
settings_hash=$(printf '%s\n' "$hosts" "$domain" "$base_path" "$frontend_port" "$backend_port" "$bootstrap_port" | sha256sum | cut -d ' ' -f 1)
refresh=false
if [ "$managed_gateway" = true ]; then
  [ -f "$config_dir/prepared-settings" ] && [ "$(cat "$config_dir/prepared-settings")" = "$settings_hash" ] || refresh=true
  if [ -z "$domain" ] && [ -f "$config_dir/tls/server.crt" ]; then
    openssl x509 -checkend 2592000 -noout -in "$config_dir/tls/server.crt" >/dev/null || refresh=true
  fi
fi
if [ ! -e "$config_dir/Caddyfile" ] || [ "$refresh" = true ]; then
  if [ -z "$domain" ]; then
    [ -n "$hosts" ] || fail '请在 Compose 顶部的 LEAFDOC_HOSTS 填写 NAS 实际 IP 或主机名'
    sans='DNS:localhost,IP:127.0.0.1,IP:::1'
    for address in $hosts; do
      case "$address" in *[!A-Za-z0-9.:-]*|'') fail '访问地址无效';; esac
      case "$address" in *:*|[0-9]*.[0-9]*.[0-9]*.[0-9]*) sans="$sans,IP:$address";; *) sans="$sans,DNS:$address";; esac
    done
    # CA 私钥持久保存，签发中断时使用原私钥继续生成证书，不更换信任身份。
    if [ ! -e "$config_dir/tls/ca.key" ] && [ ! -e "$config_dir/tls/local-ca.crt" ]; then
      openssl genrsa -out "$config_dir/tls/ca.key.next" 3072
      mv "$config_dir/tls/ca.key.next" "$config_dir/tls/ca.key"
    fi
    if [ -f "$config_dir/tls/ca.key" ] && [ ! -e "$config_dir/tls/local-ca.crt" ]; then
      openssl req -new -x509 -sha256 -key "$config_dir/tls/ca.key" -out "$config_dir/tls/local-ca.crt.next" -days "$ca_days" -subj '/CN=LeafDoc Local CA' -addext 'basicConstraints=critical,CA:TRUE' -addext 'keyUsage=critical,keyCertSign,cRLSign'
      mv "$config_dir/tls/local-ca.crt.next" "$config_dir/tls/local-ca.crt"
    fi
    [ -f "$config_dir/tls/ca.key" ] && [ -f "$config_dir/tls/local-ca.crt" ] || fail 'CA 文件不完整，请恢复原有证书备份'
    openssl genrsa -out "$config_dir/tls/server.key.next" 2048
    openssl req -new -key "$config_dir/tls/server.key.next" -out "$config_dir/tls/server.csr" -subj '/CN=localhost'
    printf 'subjectAltName=%s\nbasicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature,keyEncipherment\nextendedKeyUsage=serverAuth\n' "$sans" > "$config_dir/tls/extensions.cnf"
    openssl x509 -req -in "$config_dir/tls/server.csr" -CA "$config_dir/tls/local-ca.crt" -CAkey "$config_dir/tls/ca.key" -CAcreateserial -out "$config_dir/tls/server.crt.next" -days "$certificate_days" -sha256 -extfile "$config_dir/tls/extensions.cnf"
    mv "$config_dir/tls/server.key.next" "$config_dir/tls/server.key"
    mv "$config_dir/tls/server.crt.next" "$config_dir/tls/server.crt"
    rm "$config_dir/tls/server.csr" "$config_dir/tls/extensions.cnf"
    printf '{\n auto_https off\n skip_install_trust\n' > "$config_dir/Caddyfile.next"
    if [ -z "$bootstrap_port" ]; then
      printf ' servers :80 {\n protocols h1\n }\n' >> "$config_dir/Caddyfile.next"
    else
      printf ' servers :%s {\n protocols h1\n }\n' "$bootstrap_port" >> "$config_dir/Caddyfile.next"
    fi
    printf '}\nhttps://:%s {\n tls /etc/caddy/tls/server.crt /etc/caddy/tls/server.key\n reverse_proxy app:%s {\n lb_try_duration %s\n }\n}\n' "$frontend_port" "$backend_port" "$retry_duration" >> "$config_dir/Caddyfile.next"
    if [ -z "$bootstrap_port" ]; then
      printf 'http:// {\n redir https://{host}:%s{uri} permanent\n}\n' "$frontend_port" >> "$config_dir/Caddyfile.next"
    fi
  else
    printf '{\n skip_install_trust\n}\n%s {\n reverse_proxy app:%s {\n lb_try_duration %s\n }\n}\n' "$domain" "$backend_port" "$retry_duration" > "$config_dir/Caddyfile.next"
  fi
  if [ -n "$bootstrap_port" ] && [ -z "$domain" ]; then
    # 明文入口只有公开 CA 路由，其他路径统一拒绝，不转发数据库或登录请求。
    printf 'http://:%s {\n @ca {\n method GET\n path %s/api/setup/ca\n }\n handle @ca {\n reverse_proxy app:%s\n }\n handle {\n respond 404\n }\n}\n' "$bootstrap_port" "$base_path" "$backend_port" >> "$config_dir/Caddyfile.next"
  fi
  mv "$config_dir/Caddyfile.next" "$config_dir/Caddyfile"
  printf '%s\n' "$settings_hash" > "$config_dir/prepared-settings.next"
  mv "$config_dir/prepared-settings.next" "$config_dir/prepared-settings"
fi
if [ -f "$config_dir/tls/local-ca.crt" ]; then
  cp "$config_dir/tls/local-ca.crt" "$public_dir/local-ca.crt"
  chmod 644 "$public_dir/local-ca.crt"
  printf '公开 CA SHA256 指纹: '
  openssl x509 -in "$public_dir/local-ca.crt" -noout -fingerprint -sha256
fi
if [ ! -f "$data_dir/installation.json" ]; then
  printf '首次安装令牌: '
  cat "$data_dir/setup-token"
  printf '\n'
fi
printf '实例准备完成，已有配置与初始化记录已保留。\n'
