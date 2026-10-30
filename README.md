# LeafDoc

项目地址：[LeafDoc](https://github.com/NianBroken/LeafDoc)

LeafDoc 使用 Go 提供接口，使用 React 提供页面。MySQL 是全部业务数据的唯一持久化来源，保存账户、会话、API 密钥、正文、目录、历史版本、收藏、标签、属性、链接、搜索和站点设置。附件、头像、品牌图片及完整备份包保存在文件目录中。

## 构建与运行

Go 版本以 `go.mod` 的工具链声明为准，Node.js 主版本以 `.nvmrc` 为准。前端依赖使用 `ui/package-lock.json` 锁定。全部依赖安装、编译和测试在 Linux 容器中执行，Windows 仅调用 Docker、Node.js 调度脚本和已安装的 Caddy，不生成 Windows 可执行产物。

Windows 使用 `scripts/start.ps1` 构建内嵌前端的 Linux 镜像并启动独立 HTTPS 服务。脚本从进程环境和系统环境解析 Docker、Node.js 和 Caddy，不依赖固定安装目录。Docker 必须使用 Linux 容器模式，系统需要可连接且预先创建的 MySQL 业务数据库。

项目根目录的 `.env` 保存本机运行配置，该文件受 Git 忽略规则保护。已有进程环境变量优先于 `.env`。数据库参数如下，密码应填写实际值，不得提交到源码仓库。

```dotenv
LEAFDOC_MYSQL_HOST=your-mysql-host
LEAFDOC_MYSQL_PORT=3306
LEAFDOC_MYSQL_DATABASE=leafdoc
LEAFDOC_MYSQL_USER=your-mysql-user
LEAFDOC_MYSQL_PASSWORD=your-mysql-password
LEAFDOC_ADMIN_USERNAME=Nian
LEAFDOC_ADMIN_EMAIL=Nian@LeafDoc.doc
LEAFDOC_ADMIN_PASSWORD=Nian123456
```

MySQL 主机填写主机名或 IP 地址，不使用 `http://` 前缀。目标数据库需要预先创建，并授予配置账户建表、读写和结构升级权限。连接使用 `utf8mb4` 和 UTC 时间。应用不会在数据库连接失败时使用其他存储方式。

```powershell
./scripts/start.ps1 -Build
```

HTTPS 证书默认从系统公共数据目录下的 `LeafDoc/tls/server.crt` 和 `LeafDoc/tls/server.key` 读取。通过 `LEAFDOC_TLS_CERT` 和 `LEAFDOC_TLS_KEY` 指定其他受信任证书和私钥。管理员参数仅在空数据库初始化时生效，重启不会覆盖已有账户密码。JWT 密钥由脚本生成并保存在运行目录中，不写入备份或源码。

端口分配由 `internal/core/serviceports/ports.json` 统一维护。

| 服务 | 固定端口 | 访问方式 |
| --- | --- | --- |
| HTTPS 前端入口 | 24691 | `https://localhost:24691` |
| 后端接口 | 24692 | `http://127.0.0.1:24692` |
| Caddy 管理接口 | 24693 | 仅监听本机 |
| 独立指标服务 | 24694 | 启用指标时监听 |

启动脚本会强制结束直接占用目标 TCP 或 UDP 端口的进程，确认端口释放后再启动，不自动选择其他端口。已记录的旧实例也会先停止。`-Port` 仅接受统一分配中的前端端口。

`-DataDirectory` 指定实例文件目录，`-BasePath` 指定部署前缀。服务使用隐藏后台进程，终端退出不停止服务。使用以下命令停止当前实例：

```powershell
./scripts/start.ps1 -Stop
```

运行记录位于数据目录中的 `.runtime/processes.json`，保存容器 ID 和网关进程身份。后端日志通过 `docker logs` 读取，HTTPS 网关日志保存在同目录。停止命令核对容器所属项目及进程路径与创建时间，只停止匹配的实例。后端容器使用 `unless-stopped`，终端退出后继续运行，Docker 恢复后自动启动。证书私钥、数据库密码、JWT 密钥、运行日志和构建产物不得进入 Git。

## 开发与容器

`node scripts/container.mjs build` 构建 Linux 应用镜像，`node scripts/container.mjs release` 使用相同流程提供发布镜像。`make build` 和 `make release` 调用同一入口，不导出 Windows 程序。`make run` 和 `make dev` 调用 Windows 持久启动脚本。

`Dockerfile.tools` 提供独立 Linux 检查环境，前端依赖保存在镜像中，不复用 Windows 的 `node_modules`。前端包声明仅支持 Linux，依赖安装由容器执行。业务数据库、`.env`、实例目录和证书私钥均不进入构建上下文。前端与后端通过同一个 HTTPS 入口访问。

开发工具、预览服务与正式服务执行相同的身份验证和权限规则，不提供免登录编辑模式。公开访问仅允许匿名读取，写入始终要求有效身份与相应角色。可信反向代理认证和 API 密钥认证继续受权限校验约束。

Vite 开发与预览入口在返回 HTML 前读取后端品牌信息，后端必须同时运行。后端不可用时入口返回服务不可用页面，不返回未经解析的模板。编辑页面始终自动保存有效修改，不提供关闭自动保存的偏好选项。版本冲突和无效输入仍会阻止自动覆盖。

容器提供内嵌前端的后端服务，容器端口为 `24692`。通过容器环境传入 MySQL 和认证配置，在外部 HTTPS 网关将前端入口 `24691` 转发到后端 `24692`。数据库服务独立部署，不随应用容器创建本地数据库。

## 默认内容与资源

站点默认名称为 LeafDoc。默认 Logo、站点图标和账户头像来自同一张原始图片，源文件位于 `resources`，浏览器资源由 `internal/core/brandassets` 内嵌提供。资源地址支持部署前缀，未上传头像的合法账户直接获得默认图片。用户仍可上传与删除自定义头像，管理员仍可管理自定义品牌资源。

空实例初始化五篇简体中文 Markdown 教程，内容源文件位于 `internal/core/defaultdocs/content`。页面和初始化状态写入 MySQL，完成后不会因重启重复创建页面，也不会重建用户主动删除的教程。教程覆盖写作流程、文本与代码、列表与引用、链接图片表格、扩展语法与排错。初始化状态随完整备份保存和恢复。

项目自身技术标识使用 `leafdoc`，环境变量使用 `LEAFDOC_`，Markdown 托管元信息使用 `leafdoc_*`。当前格式不读取旧项目名称对应的环境变量与数据字段，已有实例切换前应明确安排数据重建或独立转换流程。

## 完整备份与恢复

启动脚本启用完整备份和历史版本。API 密钥管理沿用项目默认关闭状态，不显示设置入口。管理员可以在完整备份页面设置自动计划、创建备份、下载、恢复和删除备份。

完整备份采用带格式版本的 ZIP 包，包含数据库业务数据导出、备份元信息、资源文件和校验清单。备份不包含数据库连接密码、TLS 私钥和 JWT 密钥。恢复前检查格式与文件校验，恢复数据库及文件资源后刷新业务缓存，并使旧会话失效。

下载的 ZIP 不应直接解压覆盖运行目录。使用页面恢复功能，或在服务停止后使用相同的离线恢复入口。

```powershell
# 先停止服务，再将实际数据目录挂载到 Linux 容器。
./scripts/start.ps1 -Stop
docker run --rm --env-file .env --mount "type=bind,source=$((Resolve-Path ./data).Path),target=/app/data" leafdoc:local --data-dir /app/data restore-snapshot /app/data/snapshots/backup.zip
./scripts/start.ps1
```

离线恢复同样需要通过环境变量或 `.env` 提供 MySQL 配置。包含旧本地数据库文件的备份不受支持，恢复入口会明确拒绝不兼容格式。

## 验证

自动检查不创建测试数据库，不执行测试数据库迁移，不清空业务数据库。依赖隔离数据库的自动测试及其启动工具不属于项目提供的测试能力。保留的独立测试与前端检查统一在 Linux 容器运行。

```powershell
node scripts/container.mjs test
node scripts/container.mjs vet
node scripts/container.mjs ui-check
node scripts/container.mjs verify
node scripts/container.mjs schema
```

实际部署还应通过浏览器验证登录、用户操作、站点名称、默认图片、五篇教程、链接、搜索、页面创建和编辑、历史版本及完整备份恢复。在不同窗口宽度下截图核对表头与内容对齐，执行数据库重启持久化检查，完成后保留后台服务运行。仅构建成功不能代替浏览器访问验证。

## 初始化默认值与数据库结构

初始管理员为 `Nian`，邮箱为 `Nian@LeafDoc.doc`，默认密码为 `Nian123456`。空库初始化创建该账号，后续启动保留已有账号及密码。站点使用中文、年月日日期、24 小时制、公开只读访问和默认隐藏目录。自动备份使用 `Asia/Shanghai` 时区，每天 `00:00` 执行，启动时备份及保留数量沿用系统默认设置。

`internal/core/database/default.sql` 是唯一的 SQL 默认文件，包含全部表结构、字段说明、索引、关联说明及初始化值。Go 建表定义和服务默认值是生成来源。手动 SQL 初始化使用已选定的空 MySQL 数据库，完成后正常启动应用，由应用生成教程和运行期随机标识。

```powershell
node scripts/container.mjs schema
```

执行 `node scripts/container.mjs schema-write` 同时生成 SQL 和前端默认值，生成器不连接数据库。前端初始默认值与 SQL 使用同一 Go 生成器，不能单独修改生成文件。容器检查入口核对生成内容是否与定义一致。完整配置来源与空值含义见 `docs/configuration-defaults.md`。文件路径按运行目录自动解析，邮件及代理等可选功能在未配置时禁用，数据库连接与证书使用实际部署配置。
