# Ubuntu 部署与服务维护

## 选择运行路径

| 路径 | 数据库 | 配置目录 | 管理入口 |
| --- | --- | --- | --- |
| 离线包 | 管理员提供外部 MySQL | /etc/leafdoc | /opt/leafdoc/manage.sh |
| 源码 Compose | Compose database 服务 | /etc/leafdoc-compose | other/scripts/deploy.sh |
| 原生 systemd | 独立原生 MySQL | install 参数指定 | other/scripts/server.sh |

同一机器不要让两套服务抢占相同固定端口。普通使用优先离线包，完整步骤见 [Docker 部署](docker-packaging.md)。以下命令均在 Ubuntu 宿主执行。

## 源码 Compose

准备系统级 Git、Node.js、Docker Engine、Compose、Python 3 和 OpenSSL。进入完整仓库：

```sh
docker info
docker compose version
node other/scripts/package.mjs
sudo bash other/scripts/deploy.sh init --hosts localhost,127.0.0.1
sudo bash other/scripts/deploy.sh config
sudo bash other/scripts/deploy.sh start
sudo bash other/scripts/deploy.sh status
```

init 生成私有配置和证书，重复执行保留已有凭据。hosts 使用逗号分隔，局域网访问应包含实际服务器地址。public 模式通过 domain 参数选择，但公网可信证书需自行提供，不能把本地 CA 当作公众受信任证书。

部署脚本按配置启动 database、app 和 gateway，再验证 HTTPS、前端静态资源、配置接口及管理员登录。默认管理员标识由认证配置决定，自动生成密码在私有 environment 中，不能假定与离线向导设置相同。请在宿主受控终端查看，不复制进项目文件。

```sh
sudo bash other/scripts/deploy.sh logs
sudo bash other/scripts/deploy.sh stop
sudo bash other/scripts/deploy.sh start
sudo bash other/scripts/deploy.sh restart
sudo systemctl enable --now docker
```

Compose 命名卷保存数据库和文档资源，不执行 down -v 更新项目。构建新包会更新 leafdoc:local 标签，再通过 start 让 Compose 应用新镜像。配置、证书与业务卷分别保留。

## 原生 systemd 安装

server.sh 明确只接受 Ubuntu 18.04、20.04 和 22.04，不将其他 Ubuntu 版本写成原生安装器已支持。需要 systemd、root 权限、两个 Linux amd64 程序和 MySQL 通用二进制压缩包。安装器安装所列系统库，不代替下载和准备 MySQL 包。

先通过项目 Dockerfile 构建镜像，提取其中已编译程序。以下在仓库根目录执行：

```sh
node other/scripts/package.mjs
mkdir -p /tmp/leafdoc-native
CID=$(docker create leafdoc:local)
docker cp "$CID:/app/leafdoc" /tmp/leafdoc-native/leafdoc
docker cp "$CID:/app/leafdoc-gateway" /tmp/leafdoc-native/leafdoc-gateway
docker rm "$CID"
chmod +x /tmp/leafdoc-native/leafdoc /tmp/leafdoc-native/leafdoc-gateway
sudo python3 other/scripts/configure-deployment.py --mode native --directory /etc/leafdoc --hosts localhost,127.0.0.1
read -r -p 'MySQL 通用 Linux amd64 二进制压缩包路径: ' MYSQL_ARCHIVE
sudo bash other/scripts/server.sh install /tmp/leafdoc-native/leafdoc /tmp/leafdoc-native/leafdoc-gateway "$MYSQL_ARCHIVE" /etc/leafdoc
```

MySQL 包需为可在目标 Ubuntu 运行的版本，归档顶层解开后包含 bin/mysqld，安装器使用 strip-components=1 解压。执行 ldd 验证库完整，缺库会终止，不伪装安装成功。选择官方发布包并验证其官方校验信息。

程序安装至 /usr/local/lib/leafdoc，MySQL 至 /usr/local/mysql，业务目录 /var/lib/leafdoc，数据库目录 /var/lib/leafdoc-mysql。安装器建立专用系统用户、初始化数据库，生成并启用 leafdoc-mysql、leafdoc 和 leafdoc-gateway 三个 unit，再验证 HTTPS 和登录。

```sh
sudo bash other/scripts/server.sh status
sudo journalctl -u leafdoc -u leafdoc-gateway -u leafdoc-mysql -n 100 --no-pager
sudo bash other/scripts/server.sh stop
sudo bash other/scripts/server.sh start
sudo bash other/scripts/server.sh restart
```

systemd unit 的 Restart 与启动依赖负责恢复，enable 负责开机启动。更改 unit 后 daemon-reload，更改 EnvironmentFile 后重启对应服务。uninstall 停止并禁用 unit，保留程序、配置和数据。

## 访问、登录和维护

前端 24691、后端 24692，由共享端口表控制，同端口支持 HTTP 与 HTTPS。路径前缀加入 URL，证书 SAN 必须覆盖实际访问地址。信任 local-ca.crt 后检查健康、首页静态资源和管理员实际登录，再执行页面读写验证。

数据库、实例目录和配置分别维护。错误按日志定位，优先检查服务状态、数据库、证书权限及端口。任何重建或更新都不以删除数据库解决配置问题。
