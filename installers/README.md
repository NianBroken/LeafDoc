# LeafDoc Docker 安装包

本目录提供 LeafDoc 的多阶段镜像构建文件、Docker Compose 服务定义和运行配置示例。构建会从仓库源码编译前端和后端，生成可独立运行的 Linux 镜像，不依赖宿主机的 Node.js 或 Go。

## 构建

Docker Desktop 切换到 Linux 容器模式后，在仓库根目录执行：

```powershell
docker build --pull -f installers/Dockerfile -t leafdoc:local .
```

构建上下文必须是仓库根目录，因为 Dockerfile 会读取 `backend/`、`frontend/` 和 `other/` 的源码与锁文件。

## 使用 Compose 启动

在本目录创建仅供本机运行的 `.env` 文件，设置根密码和应用数据库密码，再执行：

```powershell
docker compose --env-file .env -f installers/compose.yaml up --build -d
docker compose --env-file .env -f installers/compose.yaml ps
docker compose --env-file .env -f installers/compose.yaml logs app
```

应用默认只绑定回环地址 `127.0.0.1:24692`。需要从其他设备访问时，将 `LEAFDOC_BIND_ADDRESS` 改为主机实际接口地址，并按部署环境配置 HTTPS 网关与防火墙。

## 首次初始化

打开 `http://127.0.0.1:24692`，按页面逐步验证安装令牌、数据库连接和管理员资料。安装令牌首次启动时由实例生成，可从应用数据卷读取：

```powershell
docker compose --env-file .env -f installers/compose.yaml exec app cat /app/data/setup-token
```

数据库服务由 Compose 运行，安装向导负责创建目标数据库并初始化 LeafDoc 所需结构。应用数据和 MySQL 数据分别保存在 `application-data` 与 `database` 命名卷中，容器重建和升级时保留。

运行环境变量只在容器启动时注入。不要将 `.env`、数据库密码、安装令牌或数据卷内容加入镜像或提交到版本库。停止服务使用 `docker compose ... down`，不要附加 `-v`，以保留持久数据。
