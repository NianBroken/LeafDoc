# LeafDoc

LeafDoc 使用 Go 提供接口，使用 React 提供页面，将文档保存为 UTF-8 Markdown 文件，将账户、会话和检索数据保存在本机 SQLite 数据库中。

## 构建与运行

Go 版本以 `go.mod` 的工具链声明为准，Node.js 主版本以 `.nvmrc` 为准。前端依赖使用 `ui/package-lock.json` 锁定。通过既有包管理方式恢复依赖后，在 `ui` 目录执行 `npm run build`。

Windows 使用 `scripts/start.ps1` 构建内嵌前端并启动独立 HTTPS 服务。脚本从系统环境中解析 Go、Node.js 和 Caddy，不依赖固定安装目录。

```powershell
./scripts/start.ps1 -Build
```

启动前设置 `LEAFDOC_TLS_CERT` 和 `LEAFDOC_TLS_KEY`，指向本机受信任的证书和私钥。首次启动通过 `LEAFDOC_ADMIN_USERNAME`、`LEAFDOC_ADMIN_EMAIL`、`LEAFDOC_ADMIN_PASSWORD` 提供管理员身份。密码仅通过当前进程环境传入，不保存在源码中。

脚本输出实际访问地址，默认自动选择可用端口。`-Port` 指定入口端口，`-DataDirectory` 指定实例数据目录，`-BasePath` 指定部署前缀。服务使用独立后台进程，终端退出不停止服务。再次启动前可用以下命令停止当前实例：

```powershell
./scripts/start.ps1 -Stop
```

进程记录位于数据目录中的 `.runtime/processes.json`，同目录保存后端和 HTTPS 网关日志。脚本只停止路径与启动时间均匹配的实例进程。证书私钥、JWT 密钥、数据库、运行日志与构建产物不得进入 Git。

## 默认内容与资源

站点默认名称为 LeafDoc。默认 Logo、站点图标和账户头像来自同一张原始图片，源文件位于 `resources`，浏览器资源由 `internal/core/brandassets` 内嵌提供。资源地址支持部署前缀，未上传头像的合法账户直接获得默认图片。用户仍可上传与删除自定义头像，管理员仍可管理自定义品牌资源。

空实例初始化五篇中文 Markdown 教程，正文唯一来源位于 `internal/core/defaultdocs/content`。初始化状态写入 `default-documents.json`，完成后不会因重启重复创建页面，也不会重建用户主动删除的教程。初始化状态随完整快照保存和恢复。

项目自身技术标识使用 `leafdoc`，环境变量使用 `LEAFDOC_`，Markdown 托管元信息使用 `leafdoc_*`。当前格式不读取旧项目名称对应的环境变量与数据字段，已有实例切换前应明确安排数据重建或独立转换流程。

## 验证

```powershell
go test ./...
Push-Location ui
npm.cmd test
npm.cmd run lint
npm.cmd run format:check
npm.cmd run check:architecture
npm.cmd run build
Pop-Location
```

实际部署还应验证登录、站点名称、默认图片、五篇教程、链接、搜索以及页面创建和编辑。仅构建成功不能代替浏览器访问验证。
