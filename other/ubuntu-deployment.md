# Ubuntu 部署与运行

## 部署对象

主要运行环境为 Ubuntu 18.04、20.04、22.04，提供原生 systemd 和 Docker Compose 两种部署方式。Windows 仅作为开发电脑和本地虚拟机宿主。

Ubuntu 18.04 和 20.04 已结束标准维护，Docker 当前官方支持列表也不包含这两个版本。旧版本从各自的官方签名仓库安装历史兼容包，不承诺上游仍提供安全更新。新服务器使用 Ubuntu 22.04 或更新的受支持版本。版本边界见 [Docker Ubuntu 安装要求](https://docs.docker.com/engine/install/ubuntu/)。

## 构建

在源码根目录执行 `node other/scripts/container.mjs build` 构建 Linux 应用镜像。构建依赖位于容器内，按现有锁文件安装。原生程序包含前端页面，部署后的服务器不需要 Node、Go 或 Docker。

原生二进制可以从应用镜像的 `/app/leafdoc` 导出。Go 构建使用 `CGO_ENABLED=0`，不依赖宿主 glibc。Caddy 使用官方 Linux 二进制，MySQL 使用官方 MySQL 8.0 的 glibc 2.17 通用最小安装包，安装方式见 [MySQL 8.0 通用二进制安装说明](https://dev.mysql.com/doc/refman/8.0/en/binary-installation.html)。下载后核验官方校验信息。

## Docker Compose

在源码根目录执行以下命令。`init` 需要系统 Python 3 和 OpenSSL，初始化保留已有密码与密钥。

```sh
sudo bash other/scripts/install-docker.sh
sudo bash other/scripts/deploy.sh init --hosts localhost,127.0.0.1,服务器地址
sudo bash other/scripts/deploy.sh start
sudo bash other/scripts/deploy.sh status
sudo bash other/scripts/deploy.sh logs
sudo bash other/scripts/deploy.sh stop
```

将示例中的服务器地址换成实际地址。私有配置默认为 `/etc/leafdoc-compose/environment`，可通过 `LEAFDOC_CONFIG_DIR` 指定其他系统目录。配置中保留现有 `LEAFDOC_*` 参数，密码与密钥随机生成，不写入仓库。

数据库、正文和网关状态使用独立命名卷。停止服务保留数据，不执行 `down -v`。数据库没有宿主端口映射，应用仅供网关访问。镜像中的应用使用专用普通用户运行，绑定宿主目录时必须由管理员按镜像用户的实际 UID 与 GID 配置目录属主，不使用全员可写权限。

首次运行前确保 `LEAFDOC_IMAGE` 指向实际构建或加载的镜像。生产环境可将数据库、应用和 Caddy 镜像改为已验证的摘要引用。初始化生成的镜像标签用于首次解析，部署记录应保存实际镜像摘要及系统组件版本。数据库健康检查通过后才会启动应用，详见 [Compose 启动顺序](https://docs.docker.com/compose/how-tos/startup-order/)。

## 原生 systemd

准备应用程序、Caddy Linux 程序和已校验的 MySQL 安装包，然后执行：

```sh
sudo python3 other/scripts/configure-deployment.py --mode native --directory /etc/leafdoc --hosts localhost,127.0.0.1,服务器地址
sudo bash other/scripts/server.sh install /绝对路径/leafdoc /绝对路径/caddy /绝对路径/mysql.tar.xz /etc/leafdoc
sudo bash other/scripts/server.sh status
```

程序位于 `/usr/local/lib/leafdoc`，数据库程序位于 `/usr/local/mysql`，正文位于 `/var/lib/leafdoc`，数据库位于 `/var/lib/leafdoc-mysql`。三项服务分别为 `leafdoc-mysql`、`leafdoc`、`leafdoc-gateway`。

重复安装保留数据库和凭据。安装器不会结束其他程序来抢占端口。端口冲突必须查看占用来源并修正配置。卸载仅注销服务，保留数据、配置与程序供恢复使用。

通过 `journalctl -u leafdoc -u leafdoc-gateway -u leafdoc-mysql` 查看原生日志。数据库延迟启动时后端由 systemd 重试，正常停止给予数据库完整退出时间。

## HTTPS、局域网与公网

本地与局域网默认使用部署目录下 `tls/local-ca.crt` 签发的证书。将该 CA 导入客户端受信任根证书库。服务器证书必须包含实际访问的 DNS 名称或 IP，新增访问地址时重新签发证书，不能关闭浏览器证书验证。

业务前端端口从 `backend/internal/core/serviceports/ports.json` 统一取得。局域网防火墙仅开放 HTTPS 入口，禁止开放数据库、SSH 转发和管理端口。

公网启用步骤：

1. 准备真实域名，将 A 或 AAAA 记录指向可从公网访问的服务器地址。
2. 路由器转发或云防火墙放行 TCP 80、TCP 443，以及需要 HTTP/3 时的 UDP 443。运营商级 NAT 下需先取得公网入口。
3. 对相应部署重新执行配置生成命令并传入 `--domain 实际域名`，沿用原配置目录保留凭据。Docker 使用 `deploy.sh init --domain 实际域名`，原生使用 `configure-deployment.py --mode native --directory /etc/leafdoc --domain 实际域名`。
4. 重启网关或 Compose 服务。Compose 自动叠加公网端口配置，Caddy 为域名申请和续期正式证书。
5. 从外部网络验证 DNS、HTTPS、登录、上传和下载，确认代理转发信息与实际外部地址一致。

没有域名和公网入口时只能验证配置生成与本地服务，不能认定公网访问已经完成。

## 可用性验证

`verify-deployment.py` 校验真实 HTTPS、前端资源、配置接口和管理员登录。管理员名称默认沿用应用的 `Nian`，密码保存在部署环境文件中。

```sh
sudo python3 other/scripts/verify-deployment.py --url https://localhost:实际前端端口 --ca /etc/leafdoc/tls/local-ca.crt --environment /etc/leafdoc/environment
```

Docker Compose 异常退出恢复测试会强制终止唯一匹配的 LeafDoc 应用进程，并等待 `unless-stopped` 策略将它恢复到健康状态。测试期间应用会短暂中断。

```sh
sudo bash other/scripts/test-container-recovery.sh
```

功能验收还需覆盖中文文档、目录移动、搜索、附件、历史版本、导入导出和快照恢复，并验证服务重启与系统重启后的数据持久保存。Docker 健康检查用于报告状态，进程退出由重启策略处理，不能把健康状态报告等同于业务验收。
