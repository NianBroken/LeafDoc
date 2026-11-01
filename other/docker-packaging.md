# 打包、上传与 Ubuntu Docker 离线安装

本文同时作为源码仓库的打包说明和离线安装包中的 `README.md`。源码命令在仓库根目录执行，安装命令在解压后的安装包目录执行。两者不是同一个目录。

## 1. 安装包实际包含什么

发行入口读取 `other/project.json` 的名称与版本，创建源码快照，重新构建 `linux/amd64` 应用镜像，拉取 `caddy:2-alpine`，将两个镜像导出，再压缩为安装包。目标服务器不需要 Go、Node.js 或 npm，但必须已经安装可工作的 Linux Docker Engine 与 Compose 插件。

安装包不包含 MySQL 镜像，也不安装数据库。部署前准备可从应用容器访问的 MySQL 数据库，创建专用数据库与具有读写和建表权限的账号。数据库地址不能把容器内的 `127.0.0.1` 当作宿主机。数据库在另一台主机时，使用其实际地址，并限制防火墙的允许来源。

归档由以下文件组成：

| 文件 | 作用 |
| --- | --- |
| `images.tar` | 应用和网关镜像，安装时由 `docker load` 导入 |
| `compose.yaml` | 应用与网关服务定义，不含数据库 |
| `install.sh` | 系统级安装、升级和日常管理入口 |
| `release.env` | 本次镜像标识与端口 |
| `manifest.json` | 版本、构建时间、平台、源码文件摘要及镜像标识 |
| `SHA256SUMS` | 包内文件的 SHA-256 校验清单 |
| `README.md` | 本说明 |

应用运行时数据挂载到 `/var/lib/leafdoc`，配置放在 `/etc/leafdoc`，管理程序安装到 `/opt/leafdoc`。不要将这些路径当作可以清理的解压缓存。

## 2. 在源码机器打包

准备 Git、Node.js 和可访问的 Linux Docker。版本声明来自 `other/.nvmrc`、后端模块和 Dockerfile，构建过程在容器中执行应用依赖安装与编译。打包需要联网获取构建基础镜像、Go 模块、npm 依赖和 Caddy 镜像。离线运行不表示首次构建也不需要网络。

```bash
node --version
docker info --format '{{.OSType}}'
docker compose version
node other/scripts/package.mjs
```

Docker 类型应输出 `linux`。也可以执行 `node other/scripts/container.mjs build` 或 `release`，这两个动作均转交同一个打包脚本，不是独立的快速镜像构建命令。

产物位于根目录 `installers/`，名称由项目名、版本与时间生成。终端打印归档路径、归档 SHA-256 和源码摘要。脚本会先清理该输出目录中的已有内容，因此需要长期保存的旧版本必须提前复制到目录之外。

打包会读取受 Git 管理和未被忽略的文件，排除依赖、数据、密钥配置和指定构建产物。不要求先提交修改，但构建期间不要继续修改源码。脚本会对比构建前后的文件清单和摘要，发现变化后拒绝将该次构建作为完成产物。

应用镜像使用带版本与时间的标签。完整发行成功后，打包入口将同一镜像标记为 `leafdoc:local`，供本机启动脚本和源码 Compose 默认配置使用。发行未成功时不更新该标签。打包入口不会推送镜像到注册表。需要明确固定版本时，在源码 Compose 配置中设置本次发行镜像标识，参见仓库中的 Ubuntu 部署文档。

## 3. 上传和解压

将终端生成的具体归档名称填入变量，并填写真实目标 SSH 地址。以下变量需要由操作者输入，不能原样当成服务器地址执行。

```bash
read -r -p '安装包路径: ' ARCHIVE
read -r -p '目标 SSH 地址，例如 user@host: ' TARGET
sha256sum "$ARCHIVE"
scp "$ARCHIVE" "$TARGET:/tmp/"
```

记录本机摘要，在目标 Ubuntu 上执行 `sha256sum`，必须得到相同值。再解压到专用空目录。不要把多个版本混在同一个目录，也不要把解压目录设为 `/etc/leafdoc` 或 `/var/lib/leafdoc`。

```bash
read -r -p '服务器上的归档路径: ' ARCHIVE
mkdir -p "$HOME/leafdoc-install"
tar -xzf "$ARCHIVE" -C "$HOME/leafdoc-install"
cd "$HOME/leafdoc-install"
sha256sum -c SHA256SUMS
sudo docker info --format '{{.OSType}}'
sudo docker compose version
```

所有包内文件应显示校验成功。摘要只能发现传输损坏和内容变化，下载来源的可信性仍需由管理员确认。

## 4. 首次安装与 HTTPS

本机或局域网模式直接执行：

```bash
sudo sh ./install.sh install
```

脚本导入镜像，创建系统目录、数据目录、安装令牌和 TLS 证书，然后以后台方式启动应用与网关。默认从主机地址列表生成证书；需要明确访问主机名时，在首次安装时传入空格分隔的地址：

```bash
read -r -p '访问地址列表: ' HOSTS
sudo env LEAFDOC_HOSTS="$HOSTS" sh ./install.sh install
```

首次安装需要路径前缀时，同时传入 `LEAFDOC_BASE_PATH`，例如 `/docs`。前缀只允许脚本接受的 URL 路径字符，中文先进行 URL 编码。访问入口也必须包含该前缀。

公网域名模式在首次安装时传入 `LEAFDOC_DOMAIN`。域名必须指向本机，允许外部访问 80 和 443 端口，并具备自动证书申请所需网络条件。该模式不是完全离线的证书部署方案。

```bash
read -r -p '公网域名: ' DOMAIN
sudo env LEAFDOC_DOMAIN="$DOMAIN" sh ./install.sh install
```

本地模式入口为 `https://主机地址:24691`，公网域名模式使用 `https://域名`。安装包同时映射 80 和 443，80 用于 HTTPS 跳转。安装前确认这些端口未被其他服务占用。端口来源是源码共享端口表和包内 `release.env`，不要随意只修改其中一处。

本地证书需要在实际使用的浏览器所在设备安装 CA 信任：

```bash
sudo /opt/leafdoc/manage.sh ca
sudo install -m 644 /etc/leafdoc/local-ca.crt /usr/local/share/ca-certificates/leafdoc.crt
sudo update-ca-certificates
```

其他客户端需要安全地取得同一 CA 公钥证书并导入其信任库。不要复制 CA 私钥或服务器私钥。公网模式未创建本地 CA 时，不执行 `ca` 操作。

## 5. 网页初始化与首次登录

安装输出包含首次安装令牌，令牌也保存在 `/var/lib/leafdoc/setup-token`。这是允许初始化的凭据，不是管理员登录密码。只向负责初始化的管理员提供。

打开 HTTPS 入口，按照安装页填写令牌、数据库主机、端口、库名、账号、数据库密码，以及管理员用户名、邮箱和密码。启用数据库 TLS 时，证书必须通过服务器名称验证，自建 CA 的内容需要填写到安装配置。

程序实际检查数据库读写权限，构造正式服务并等待搜索初始化后才保存完成记录。连接已有管理员的数据库时，需要提供该管理员的真实凭据，不能把首次安装当作重置已有账号密码的方法。

成功后进入正式系统，使用刚确认的管理员资料登录。验证页面树能够加载，再创建测试页面、保存正文并重新打开。健康接口在引导阶段也会返回成功响应，因此只访问 `/api/health` 不能证明数据库已配置或已经可以登录。

完成记录保存为数据目录中的 `installation.json`。其中含数据库凭据与实例密钥，必须随实例数据一起保护。不要手工删除该文件来尝试修复登录问题。

## 6. 日常运行、保活与自启

```bash
sudo /opt/leafdoc/manage.sh status
sudo /opt/leafdoc/manage.sh logs
sudo /opt/leafdoc/manage.sh stop
sudo /opt/leafdoc/manage.sh start
sudo /opt/leafdoc/manage.sh restart
```

`start` 使用 `docker compose up -d`，容器脱离当前终端运行。关闭 SSH、退出 shell 或关闭说明页面不会停止系统。`status` 显示容器状态，`logs` 显示最近 100 行日志。

应用和网关采用 `restart: unless-stopped`。异常退出后由 Docker 重启，手动停止后不会因为 Docker 再次启动而自动恢复。要恢复运行，执行 `start`。

在使用 systemd 的 Ubuntu 主机上同时启用 Docker 服务：

```bash
sudo systemctl enable --now docker
sudo systemctl is-enabled docker
```

容器重启策略不负责启动宿主机 Docker 服务，也不修复数据库停机、磁盘满或证书到期。数据库由管理员单独配置保活与备份。详见 [Docker 自动启动说明](https://docs.docker.com/engine/containers/start-containers-automatically/)。

## 7. 备份、恢复与升级

管理员在系统备份功能中创建完整快照，等待任务完成，再下载 ZIP。快照包含 MySQL 记录与文件资源，不能用只复制附件目录代替完整备份。恢复前在隔离实例验证备份能够打开和恢复，正式恢复会改变当前实例内容。

系统级灾难恢复还需要安全备份 `/etc/leafdoc`、`/var/lib/leafdoc` 和外部 MySQL。停写期间执行数据库备份及目录归档，恢复时保持数据库与文件资源来自同一时间点。配置备份包含密钥，不得放进公开下载目录或 Git。

升级前保留旧安装包并完成备份，将新版本解压到另一个空目录，校验后执行：

```bash
sudo sh ./install.sh upgrade
sudo /opt/leafdoc/manage.sh status
sudo /opt/leafdoc/manage.sh logs
```

升级替换镜像资料和管理入口，保留环境配置、Caddyfile、实例数据与安装完成记录。已有 Caddyfile 不会被安装脚本重新生成，不能通过再次传入域名或地址变量改变既有网关配置。应用环境配置位于 `/etc/leafdoc/application.env`，修改后需要重新创建应用容器，单纯 `restart` 不会重新读取 Compose 环境文件。

需要重新创建时使用已安装文件及其环境：

```bash
sudo docker compose --project-name leafdoc --env-file /etc/leafdoc/environment -f /opt/leafdoc/compose.yaml up -d --force-recreate
```

升级后重新验证 HTTPS、登录、读取和保存。不要将旧镜像直接连接升级后的数据库当作完整回滚。回退需要旧安装包及升级前一致的数据库、配置和文件资源备份。

## 8. 卸载与问题定位

```bash
sudo /opt/leafdoc/manage.sh uninstall
```

此操作移除容器与网络，保留系统目录和绑定数据，可用于后续恢复。`purge --confirm-delete-instance` 会删除实例目录与相关卷，且不会删除外部 MySQL。该命令具有破坏性，只在确认备份与目标实例后使用，不属于正常升级步骤。

无法访问时检查 `status` 与 `logs`，确认 Docker、端口映射、服务器防火墙和 DNS。证书错误应核对访问主机名、证书有效期和客户端信任库，不通过关闭校验处理。引导页无法保存时核对安装令牌、数据库地址、权限和 TLS。登录失败时使用真实已有账号，不反复修改环境文件中的初始密码。磁盘不足时检查镜像、日志和数据占用，不删除数据目录或数据库文件。

## fnOS 可视化部署适配

发行包 fnos 目录提供专用 docker-compose.yml 和界面操作说明，同一应用镜像同时用于 Ubuntu 与 fnOS。唯一打包入口在同一源码快照构建后生成两类部署资料，镜像标签、版本和来源摘要保持一致。SHA256SUMS 递归覆盖 fnos 文件，Ubuntu 安装仍可直接校验整个归档。没有增加第二套打包脚本，没有删减既有源码部署、系统安装、数据库配置、公网代理或业务能力。

fnOS Compose 的逐字段维护说明位于 fnos 目录的 README.md。源码仓库中的对应来源是 other/fnos-deployment.md，配置模板是 other/compose.fnos.yaml。打包脚本将模板中的应用镜像、网关镜像和端口占位符替换为本次实际值后生成 fnos/docker-compose.yml。维护人员应修改源码文档和模板，再重新打包，不能只修改已经解压到 NAS 的生成文件。文档逐项说明 x-* 扩展字段、YAML 锚点、三个服务、所有环境变量、端口映射、挂载选项、nocopy 和五个命名卷，并列出 Ubuntu 与 fnOS 其他 Compose 文件的选择边界。

fnOS 使用 prepare 一次性服务准备命名卷。该服务从应用镜像执行共享 prepare-instance.sh，负责权限、令牌、本地证书和网关配置。Ubuntu 安装程序也调用镜像内同一准备入口，宿主仍使用 /opt、/etc 和 /var/lib 系统目录。已有实例配置与证书不覆盖。fnOS 应用只挂载自身数据与公开证书卷，配置私钥保留在网关配置卷中，不向应用暴露。准备服务不需要 Docker 套接字或特权模式，只获得项目卷写权限。

fnOS 不需要环境文件，发行时会把实际镜像标签和集中端口写入 YAML。管理员只在可视化编辑器填写实际 NAS 地址与可选部署参数。准备失败会阻止业务容器依赖启动，重复启动继续使用原初始化记录。首次令牌从受管理界面保护的 prepare 日志取得，成功初始化后不再输出。公开 CA 下载只在应用初始化服务中开放，正式服务没有该下载处理。明文证书入口只代理 GET 证书路径，不接收初始化资料或任何正式业务请求。

本地 HTTPS 的证书验证仍由客户端执行，不能以浏览器忽略安全警告代替信任配置。管理员应在可信局域网下载公开 CA，并对照初始化日志输出的 SHA256 指纹。下载接口不会返回私钥；数据库凭据与管理员资料只能经 HTTPS 提交。公网域名模式沿用 Caddy 自动签发逻辑，需要明确映射标准挑战端口或配置前置网关，并不以默认占用 NAS 管理端口为代价。

fnOS 的完整可视化验收流程为文件管理上传、Docker 本地镜像从 NAS 文件添加、Compose 新增项目上传配置、启动准备服务、读取令牌、信任证书、网页数据库初始化及实际业务复验。升级必须更新原项目并保留名称和卷，不新建另一个同名数据来源不明的安装实例。停止或容器重建不会清空命名卷，明确删除实例卷才重新进入引导。实际测试证据应记录界面操作，不能用 SSH 预先准备目录或隐藏接口调用冒充可视化部署。
## fnOS 配置卷与重复准备约束

fnOS 的网关配置卷明确设置 nocopy，阻止 Docker 将镜像默认 Caddyfile 复制到空卷。否则初始化服务会把默认站点误认成已经准备的实例，造成容器运行而应用端口无法访问。fnOS 设置 LEAFDOC_MANAGED_GATEWAY，准备服务记录访问参数摘要，参数变化时重新生成网关文件并保留 CA 身份与业务数据。原 Ubuntu 安装不设置此开关，继续保留实例已有网关文件，避免覆盖运维人员的定制配置。证书下载服务只接受有效的单个自签名 CA 证书，私钥、拼接内容和无法解析的文件均返回服务不可用，不改变初始化门禁。
