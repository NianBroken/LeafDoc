# Ubuntu Docker 源码部署与维护

本文说明仓库中 `other/scripts/deploy.sh` 管理的完整 Compose 服务栈。它包含 MySQL、LeafDoc 应用和 Caddy HTTPS 网关。离线安装包使用另一套配置路径，只包含应用和网关，操作见 [打包与离线安装](docker-packaging.md)。不要在同一台主机上同时运行两套默认实例，两者使用相同 Compose 项目名和服务端口。

## 1. 环境与目录

在真实 Ubuntu 主机上运行 Linux Docker Engine 和 Docker Compose 插件。主机需要网络访问构建镜像与依赖源，需要为应用数据和数据库留出磁盘空间。发行入口固定构建 `linux/amd64`，不能将该归档直接当作 ARM 原生发行包。

源码机器需要 Git 与 Node.js，Python 3、OpenSSL 用于部署配置与验证。Go、npm 依赖和前端编译由容器负责，不需要在宿主机手工安装应用依赖。Node.js 版本以 `other/.nvmrc` 为准，不使用 Ubuntu 仓库中未经版本核对的旧版本替代。

项目提供的 Docker 安装脚本只接受 Ubuntu 18.04、20.04、22.04，使用这些系统时执行：

```bash
sudo bash other/scripts/install-docker.sh
```

其他 Ubuntu 版本按 [Docker 官方 Ubuntu 安装步骤](https://docs.docker.com/engine/install/ubuntu/) 安装 Engine 与 Compose 插件。不要绕过脚本的发行版检查或混入其他 Ubuntu 版本的软件源。

```bash
sudo apt-get update
sudo apt-get install -y git python3 openssl ca-certificates curl
node --version
sudo docker info --format '{{.OSType}}'
sudo docker compose version
```

下文源码命令在仓库根目录执行。源码不是运行数据，更新源码不应清理配置目录或数据库卷。

| 内容 | 位置 |
| --- | --- |
| 私有部署配置 | `/etc/leafdoc-compose/environment` |
| 本地证书与私钥 | `/etc/leafdoc-compose/tls/` |
| Caddy 配置 | `/etc/leafdoc-compose/Caddyfile` |
| 应用容器数据 | `documents` Compose 命名卷，容器内 `/app/data` |
| MySQL 数据 | `database` Compose 命名卷，容器内 `/var/lib/mysql` |
| 网关持久数据 | `gateway_data` 与 `gateway_config` 命名卷 |

卷的实际名称由 Compose 项目名生成。使用 `docker volume inspect` 查询真实位置，不把 Docker 内部路径作为手工编辑入口。

## 2. 获取源码与构建

```bash
git clone https://github.com/NianBroken/LeafDoc.git
cd LeafDoc
git status --short
git pull --ff-only
node other/scripts/package.mjs
```

打包入口创建源码快照并重新编译应用，生成 `installers/` 下的压缩包，同时留下本机镜像。`container.mjs build`、`release` 和 Makefile 的构建入口均进入这一发行流程。完整发行成功后，同一应用镜像同时标记为 `leafdoc:local`，供源码 Compose 默认配置使用。发行未成功时不更新该标签。

构建完成后从终端输出、包内 `release.env` 或 `manifest.json` 取得本次应用镜像标识。使用 `docker image ls` 核对该镜像实际存在，不选择其他旧版本替代。设置部署配置时将此标识写入 `LEAFDOC_IMAGE`。

构建过程会清理 `installers/`，构建前把需要保留的旧包移到该目录之外。更新源码时先保护未提交修改，不执行强制重置。

## 3. 生成配置与选择镜像

```bash
sudo bash other/scripts/deploy.sh init
sudoedit /etc/leafdoc-compose/environment
```

在环境文件中将 `LEAFDOC_IMAGE` 设置为刚构建的版本镜像标签，可明确固定部署版本。保留默认 `leafdoc:local` 时，使用最近一次完整发行成功的应用镜像。其余随机密码和 JWT 密钥由初始化脚本生成并保存，重复初始化会保留已有值。不要把该文件加入仓库，不要将其完整内容复制到公开日志。

源码部署的数据库容器使用配置中的 MySQL 用户、数据库和密码初始化，应用通过 Compose 内部主机名 `database` 访问它。数据库没有对外发布端口。不要用修改环境文件的方式给已有 MySQL 数据卷重设密码，镜像的初始化变量仅用于新数据库。

默认管理员用户名来自程序初值 `Nian`，初始密码是环境文件中的 `LEAFDOC_ADMIN_PASSWORD`。该密码不是固定的演示密码。已有管理员账号不会在普通启动时被初始参数覆盖。

端口定义集中在 `backend/internal/core/serviceports/ports.json`，当前网关为 24691，应用为 24692，指标为 24694。应用端口不向宿主机发布，读者从网关访问。不要仅修改 Compose 的某个端口而不核对健康检查与程序端口约束。

## 4. HTTPS、局域网与路径前缀

默认配置覆盖 `localhost` 和 `127.0.0.1`。需要局域网地址时在首次初始化提供真实主机名或地址，逗号分隔：

```bash
read -r -p '证书访问地址，逗号分隔: ' HOSTS
sudo bash other/scripts/deploy.sh init --hosts "$HOSTS"
```

证书只在尚不存在时创建。更改地址列表并不会自动替换已有证书，应提前备份配置并准备覆盖新地址的有效证书。使用自备证书时 `--tls-directory` 指向同时包含 `server.crt`、`server.key` 和 `local-ca.crt` 的目录。

需要公网域名时，在初始化提供 `--domain`。部署脚本会载入公网 Compose 覆盖文件，额外发布 80、443 和 443 UDP。域名 DNS、外部连通和证书申请条件需要实际满足。默认的本地 HTTPS 入口仍保留。

需要 URL 前缀时在环境文件设置 `LEAFDOC_BASE_PATH`，例如 `/docs`。前端、接口和健康检查均通过该前缀访问。该配置不是域名，也不包括协议或查询参数。

信任本地 CA：

```bash
sudo install -m 644 /etc/leafdoc-compose/tls/local-ca.crt /usr/local/share/ca-certificates/leafdoc-compose.crt
sudo update-ca-certificates
```

该操作只更新当前 Ubuntu 的信任库。其他客户端和具有独立证书库的浏览器仍需导入同一 CA。证书信任问题应通过正确导入证书解决，不使用跳过校验的命令当作正式访问方法。

## 5. 启动、访问与确认登录

```bash
sudo bash other/scripts/deploy.sh config
sudo bash other/scripts/deploy.sh start
sudo bash other/scripts/deploy.sh status
```

启动顺序为 MySQL 健康检查通过、应用启动并通过健康检查、网关启动。`start` 随后执行真实 HTTPS 验证，读取前端资源和配置并尝试管理员登录。最终正常入口为 `https://localhost:24691`，设置前缀后在端口后加该路径。远程客户端将 `localhost` 换成证书覆盖的服务器地址。

在浏览器完成登录，确认进入系统。创建一页测试内容，保存后重新打开，确认正文和目录实际可用。只检查容器处于 Up 状态不足以证明登录可用。

已有管理员在系统中变更密码后，环境文件中的初始密码不再代表现行密码。部署验证脚本读取环境文件尝试登录，此时需要先确认实际账号和验证配置是否一致。不能为了让验证通过而重置已有账号。

## 6. 状态、日志、保活和自启

```bash
sudo bash other/scripts/deploy.sh logs
sudo bash other/scripts/deploy.sh stop
sudo bash other/scripts/deploy.sh start
sudo bash other/scripts/deploy.sh restart
sudo systemctl enable --now docker
sudo systemctl is-enabled docker
```

服务通过 `up -d` 在后台运行，退出终端不会关闭应用。三个服务均采用 `unless-stopped`，容器异常退出时 Docker 按策略重启，手动停止后需要再次启动。Docker 服务本身必须随宿主机启动，容器策略不负责启动 Docker。

源码 Compose 已限制各容器 JSON 日志单文件 10 MB、保留 3 份。应用和网关具有停止宽限时间，不使用强制结束代替正常停止。重启循环应查看真实错误，核对数据库健康、磁盘和配置，不能用反复重启掩盖故障。

## 7. 备份与恢复

管理员在系统备份功能创建完整 ZIP 快照，待完成后下载并在隔离环境恢复验证。自动备份的间隔和保留规则需要在系统设置中确认，不能因启动参数含 `--snapshot` 就认定当前已有可用备份。

灾难恢复至少保留应用数据、MySQL 一致备份、私有环境、TLS 和网关配置。应用快照不等于服务器配置备份。执行系统级备份前安排停写窗口，并保护包含密码和密钥的备份。

对于容器级完整冷备份，可在停止整套服务后归档数据库卷和文档卷。先通过下列命令查出真实卷名及挂载关系，再按 Docker 卷备份步骤保存，不直接猜测卷名：

```bash
sudo bash other/scripts/deploy.sh stop
sudo docker volume ls --filter label=com.docker.compose.project=leafdoc
sudo docker ps -a --filter label=com.docker.compose.project=leafdoc --format '{{.Names}}'
```

备份完成后执行 `start`，再次登录验证。恢复时先恢复匹配的数据库与文档资源，再恢复私有配置并启动，禁止将一个时间点的数据库与另一个时间点的附件混用。恢复测试使用隔离机器或隔离 Docker 环境，不能连接正式数据库。

## 8. 升级、配置变更与回退

升级前下载可恢复的快照并保存配置，记录当前镜像标识。拉取最新源码，重新执行打包，确认新镜像可用，再修改 `LEAFDOC_IMAGE` 并执行：

```bash
sudo bash other/scripts/deploy.sh config
sudo bash other/scripts/deploy.sh start
sudo bash other/scripts/deploy.sh status
sudo bash other/scripts/deploy.sh logs
```

`start` 的 `up -d` 会按 Compose 差异更新容器，同时保留命名卷。环境变更使用该流程，单纯 `docker compose restart` 不重新创建环境。修改公网域名或证书前核对 Caddyfile、证书文件与 Compose 覆盖模式。

回退必须使用升级前的匹配数据备份和镜像，不能只改镜像标签就认定数据库兼容。首次初始化脚本不是数据库密码轮换工具，已有数据库的密码变更需要数据库与应用同时更新。

## 9. 停用与排障

部署脚本提供 stop，不提供删除数据的动作。正常停用使用 `stop`，保留数据卷。不要执行带 `-v` 的 Compose 删除命令来排查启动问题，该选项会删除命名卷中的数据库与正文资源。

| 现象 | 具体检查 |
| --- | --- |
| 镜像不存在或被要求拉取 | 检查本次发行是否完整成功，以及环境文件指定的镜像是否存在 |
| 数据库健康检查失败 | 查看 database 日志，核对已有数据卷的密码与环境文件，检查磁盘空间 |
| HTTPS 证书不可信 | 核对 CA 信任、访问名称和证书有效期 |
| 页面能开但无法登录 | 核对真实已有管理员凭据，检查应用日志和数据库状态 |
| 前缀访问失败 | 检查 `LEAFDOC_BASE_PATH` 与实际地址是否一致 |
| 重启后服务未恢复 | 检查 Docker 自启，确认实例是否曾被手动停止 |

配置、密钥和数据库资料只在受限权限目录维护。排障时保留真实错误信息，分享日志前去掉凭据。Ubuntu 原生 systemd 安装脚本 `server.sh` 是独立路径，需要另行提供匹配的应用二进制、Caddy 和 MySQL 归档，不应与本节 Docker 命令混用。