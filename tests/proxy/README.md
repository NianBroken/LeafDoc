# 反向代理认证集成测试

在项目根目录执行 `node tests/proxy/run.mjs`，Windows、Linux 和 macOS 使用同一入口。`run.sh` 仅转交此入口。需要 Docker Compose、Node.js 和 Go。

端口统一读取 `internal/core/serviceports/ports.json`。普通代理使用 `24695`，自动创建用户代理使用 `24696`。两个后端容器均在各自容器内使用 `24692`，不发布到宿主机。私有 MySQL 使用 `24697`，也不发布到宿主机。测试不会占用正式前端 `24691`。

运行器为每轮创建独立 Compose 项目、随机测试库名称及随机凭据。两个后端仅连接该项目中的 `mysql` 服务，各自使用独立数据库。不会读取正式 `.env`，不会连接远端正式数据库。MySQL 数据存放于容器内存挂载中，退出流程执行 `docker compose down --volumes --remove-orphans`。测试失败也执行清理，清理失败明确报告。

仅核验配置时执行 `node tests/proxy/run.mjs --check`。该命令执行 Compose 配置解析，并核对数据库隔离和端口发布，不创建容器，不监听端口，不连接数据库。直接执行 `go test` 会被隔离标记检查拒绝，避免错误测试正式运行服务。

固定测试端口被占用时必须处理占用服务，再执行测试，不得更换测试端口。不要直接运行缺少运行器环境的 Compose 命令。
