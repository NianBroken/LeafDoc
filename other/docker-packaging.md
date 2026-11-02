# Docker 离线包制作与 Ubuntu 部署

## 环境和产物

本文对应 `other/scripts/package.mjs` 与包内 `install.sh`。部署命令在 Ubuntu 宿主机执行。构建机需要 Git、Node.js 和运行 Linux 容器的 Docker，目标固定 linux/amd64。首次构建需获取基础镜像、npm 和 Go 依赖，离线指目标机不重新编译应用，不代表包内含数据库。

外层 tar.gz 是完整发行包，内部 images.tar 才是 Docker 镜像归档，不能将外层直接传给 docker load。

## 制作发行包

在仓库根目录执行：

```sh
git pull --ff-only
git status --short
docker info
docker compose version
node other/scripts/package.mjs
```

脚本读取 project.json 与共享端口表，收集已跟踪和未忽略新增文件，过滤运行数据、环境和密钥。无需先提交，也无需在宿主安装项目 npm 依赖。先复制源码快照与计算摘要，再从快照构建、导出和归档。构建或归档期间发现源码变化会终止，应停止编辑后重做。

终端输出 installers 下具体路径、归档 SHA256 和源码摘要。版本号相同仍能对应不同源码，以 manifest 确认来源。

| 包内文件 | 用途 |
| --- | --- |
| images.tar | docker save 导出镜像，同时提供 app 与 gateway |
| install.sh | 安装、升级和生命周期管理 |
| release-host-ports.sh | 启动前处理端口占用，执行前确认不影响同机其他业务 |
| release.env | 镜像标签与发行端口 |
| compose.yaml | 应用和网关，不含数据库 |
| manifest.json | 平台、版本、时间、源码摘要与逐文件摘要 |
| SHA256SUMS | 包内完整性校验 |
| source | 与镜像对应的源码快照 |
| fnos | 可视化模板与说明 |
| README.md | 本说明副本 |

## 校验与上传

构建机执行：

```sh
read -r -p '压缩包完整路径: ' PACKAGE
sha256sum "$PACKAGE"
tar -tzf "$PACKAGE" | head -n 20
read -r -p 'SSH 用户与主机，如 user@server: ' TARGET
ssh "$TARGET" 'mkdir -p "$HOME/leafdoc-upload"'
scp "$PACKAGE" "$TARGET:leafdoc-upload/"
```

目标 Ubuntu 执行，填写已上传文件的具体路径：

```sh
ls -lh "$HOME/leafdoc-upload"
read -r -p '已上传压缩包完整路径: ' PACKAGE
sha256sum "$PACKAGE"
mkdir -p "$HOME/leafdoc-release"
tar -xzf "$PACKAGE" -C "$HOME/leafdoc-release"
cd "$HOME/leafdoc-release"
sha256sum -c SHA256SUMS
cat manifest.json
```

整体摘要须与构建机一致，包内全部校验通过。失败就重新传输，不继续安装。SHA256 检查损坏，不代替来源信任。每个版本使用独立解压目录，避免混入旧文件。用 df -h 与 docker system df 检查容纳归档、解压、镜像和数据的空间。

## 数据库与宿主准备

按 Docker 官方 Ubuntu 安装流程准备系统 Engine 和 Compose 插件，执行：

```sh
sudo systemctl enable --now docker
docker info
docker compose version
```

数据库由管理员单独提供，准备容器可达地址、端口、凭据和建库或已有库权限。容器 localhost 不是宿主。向导会检查结构，清空不兼容库须输入库名确认，不使用其他业务库测试清空。

## 安装、访问和登录

目标 Ubuntu 解压目录执行：

```sh
sha256sum -c SHA256SUMS
sudo sh install.sh install
sudo /opt/leafdoc/manage.sh status
sudo /opt/leafdoc/manage.sh logs
```

安装校验后执行 docker load -i images.tar，保存部署资料，准备证书和令牌，compose up -d 后台启动。首次需要前缀时使用 `sudo env LEAFDOC_BASE_PATH=/docs sh install.sh install`，访问地址带 /docs。

| 持久位置 | 内容 |
| --- | --- |
| /opt/leafdoc | Compose、管理脚本、manifest |
| /etc/leafdoc/environment | 镜像、端口及部署变量 |
| /etc/leafdoc/application.env | 可选应用参数，已有文件保留 |
| /etc/leafdoc/tls | CA、服务证书和私钥 |
| /var/lib/leafdoc | 挂载至 /app/data 的实例目录 |
| 外部 MySQL | 页面、账户及数据库状态，不在包内 |

打开 http://服务器地址:24691 或同端口 HTTPS。输入脚本显示的令牌，按向导检查网络、验证 MySQL、处理数据库结构、设置管理员并完成安装，再使用管理员登录。进入后打开教程，保存练习页并重新读取。

```sh
sudo cat /var/lib/leafdoc/setup-token
sudo /opt/leafdoc/manage.sh ca
curl --fail http://127.0.0.1:24691/api/health
```

客户端应信任 CA 并使用证书匹配地址。已安装实例不重新走首次向导。健康成功不能替代登录成功。

## 持续运行

| 命令 | 用途 |
| --- | --- |
| sudo /opt/leafdoc/manage.sh status | 容器及健康状态 |
| sudo /opt/leafdoc/manage.sh logs | 最近 100 行日志 |
| sudo /opt/leafdoc/manage.sh stop | 主动停止，保留数据 |
| sudo /opt/leafdoc/manage.sh start | 恢复后台服务 |
| sudo /opt/leafdoc/manage.sh restart | 重启现有容器，不重新加载环境文件 |
| sudo /opt/leafdoc/manage.sh uninstall | 移除容器，保留宿主实例目录 |

purge --confirm-delete-instance 删除实例目录与配置，外部数据库不会删除，不能用于普通更新或排错。应用环境变更后执行下面命令重新创建受影响容器：

```sh
sudo docker compose --project-name leafdoc --env-file /etc/leafdoc/environment -f /opt/leafdoc/compose.yaml up -d
```

## 保活

app 和 gateway 使用 restart: unless-stopped，up -d 后与 SSH 会话分离。关闭终端仍运行，主进程退出由 Docker 重启。健康失败只标记 unhealthy，不自行重启。主动 stop 后使用 start 恢复。

```sh
sudo docker compose --project-name leafdoc --env-file /etc/leafdoc/environment -f /opt/leafdoc/compose.yaml ps
sudo docker compose --project-name leafdoc --env-file /etc/leafdoc/environment -f /opt/leafdoc/compose.yaml config --quiet
```

docker inspect 检查目标容器 HostConfig.RestartPolicy.Name，应为 unless-stopped。维护窗口先确认 app 的实际容器名，再执行 `docker exec --user 0 容器名 sh -c 'kill -TERM 1'`，从容器内部模拟主进程退出，观察 RestartCount 增加、健康恢复并重新登录。不要使用 docker kill 代替此测试，该命令属于 Docker 人工停止操作，需要手动 start 恢复。不操作其他项目容器。

## 开机自动启动

```sh
sudo systemctl enable --now docker
systemctl is-enabled docker
systemctl is-active docker
```

宿主重启后 Docker 启动，之前运行的容器恢复，人为停止的容器仍停止。无需额外循环启动脚本。维护窗口重启 Ubuntu，再检查 Docker、容器、健康和登录。保活核验进程退出，自启核验宿主重启，两者分别执行。

## 故障处理

| 现象 | 检查与处理 |
| --- | --- |
| 构建失败 | 读取首个失败阶段，检查依赖网络、基础镜像和空间 |
| 源码变化 | 停止编辑后重新发行，不使用未完成产物 |
| 镜像加载失败 | 使用内部 images.tar，确认校验及空间 |
| 容器退出 | 查看参数、证书权限、数据库与配置错误 |
| 健康超时 | 检查监听和数据库连接，处理真实根因 |
| 登录失败 | 检查账户、Cookie、前缀与安装状态 |
| HTTPS 失败 | 检查 CA 信任和 SAN，签发匹配地址证书 |

参考 [Docker Ubuntu 安装](https://docs.docker.com/engine/install/ubuntu/)、[镜像保存](https://docs.docker.com/reference/cli/docker/image/save/) 与 [自动启动策略](https://docs.docker.com/engine/containers/start-containers-automatically/)。
