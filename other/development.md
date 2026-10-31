# 开发与维护

本文面向需要修改 LeafDoc 代码和构建发行包的维护者。生产安装和日常运行见 [Ubuntu 部署](ubuntu-deployment.md) 与 [离线打包](docker-packaging.md)，功能与使用入口见 [项目说明](../README.md)。

## 开发环境与目录职责

项目的主机工具入口是 Node.js，构建和检查在 Linux 容器中执行。`frontend/package.json` 声明 Linux 操作系统要求，不能把在 Windows 宿主执行 `npm install` 当作项目默认依赖准备方式。使用 Docker Linux 容器，避免宿主系统和依赖平台不一致。

版本来源集中在 `other/.nvmrc`、`backend/go.mod`、`other/.golangci-lint-version`、npm 锁文件和 Dockerfile。工具镜像同时包含 Node 与 Go，应用镜像以多阶段构建分别产出前端和后端，运行阶段不包含编译器。

| 目录 | 职责 |
| --- | --- |
| `backend/cmd` | 应用命令入口、服务端口查询和默认配置生成工具 |
| `backend/internal/http` | HTTP 接入、静态页面及路由相关公共能力 |
| `backend/internal/wiki` | 功能路由、用例、服务装配及保存流程 |
| `backend/internal/core` | 页面、账号、存储、索引、导入、快照等业务实现 |
| `frontend/src/app` | 应用启动、路由和界面装配 |
| `frontend/src/features` | 按功能组织的页面与交互 |
| `frontend/src/shared` | 公共控件、接口约定、状态、工具和语言资源 |
| `other/scripts` | 构建、部署、版本核对、结构检查与运行入口 |
| `installers` | 生成的离线发行包，不参与版本管理 |

依赖方向应服从现有架构检查。公共规则集中维护，不在前端和后端分别手写同一份端口、默认配置或错误编号映射。

## 修改代码前的检查

```bash
git status --short
git pull --ff-only
docker info --format '{{.OSType}}'
node --version
```

确认工作区修改归属，再同步远端。不要使用强制重置覆盖其他修改。Docker 类型必须为 `linux`。修改前阅读对应功能的入口、调用链与测试，不只阅读注释或文档。

`AGENTS.md` 与现有 Git 策略钩子属于项目保护范围，不通过修改它们绕过约束。私有环境变量、数据库连接信息、证书私钥、日志和发行包不能提交。

## 统一检查命令

在仓库根目录执行，工具入口会定位项目并创建 Linux 检查镜像：

```bash
node other/scripts/container.mjs verify
node other/scripts/container.mjs test
node other/scripts/container.mjs vet
node other/scripts/container.mjs lint
node other/scripts/container.mjs ui-check
node other/scripts/container.mjs schema
```

| 动作 | 实际执行内容 |
| --- | --- |
| `verify` | 项目结构、私有引用、工具链及数据库文件检查 |
| `test` | 后端 `go test ./...` |
| `vet` | 后端 `go vet ./...` |
| `lint` | 按项目配置执行 golangci-lint |
| `ui-check` | 前端测试、零告警 lint、格式检查、架构检查和构建 |
| `ui` | 仅执行前端构建 |
| `schema` | 检查数据库默认值生成结果 |
| `schema-write` | 生成数据库默认 SQL 与前端默认配置并复制回源码 |
| `bench` | 执行链接与历史相关基准测试 |

`schema-write` 会修改源码，不能拿它替代只读检查。前端构建会同步错误翻译目录，修改错误定义时应同时检查相应输出。检查入口通过源码快照构建工具镜像，修改文件后需要重新调用入口，不能假设已有镜像已包含改动。

也可以运行 `make -f other/Makefile ci`，它汇集 verify、lint、test、vet、前端和 schema 检查。需要调整并行度时由调用者显式指定，避免多个高资源构建把机器内存耗尽。

## 功能开发的核对方法

新增页面行为时同时检查前端操作、接口请求、鉴权、业务服务和存储结果。页面保存的关联处理统一经过现有编排器，涉及搜索、链接、历史、标签及属性时不得只修改正文保存函数。

新增设置时核对初始默认值与已持久化设置的关系。环境变量是启动配置来源，不是覆盖所有运行期设置的通用入口。修改身份认证时覆盖匿名、阅读者、编辑者、管理员、令牌过期及无效会话，不能只验证管理员成功路径。

导入测试资源位于 Core 导入模块的 fixtures。README 文件名、相对链接、目录层次、附件及同名笔记是实际测试输入，改写文字后仍需保持这些场景。内置帮助通过后端资源嵌入进入程序，修改后必须重新构建应用。

测试使用隔离数据和临时目录。涉及在线恢复、数据库清理、端口释放及卸载时先核对目标归属，不针对正在交付的实例执行破坏性验证。

## 构建发行包

```bash
node other/scripts/container.mjs build
```

`build` 和 `release` 都进入 `package.mjs`，会创建源码清单、重新构建 Linux amd64 应用镜像、导出应用和 Caddy 镜像并生成压缩安装包。它不是单独编译后端的快捷方式。

打包入口会清空 `installers` 中已有内容。发行前把需要保留的包复制到仓库外。构建期间不要继续编辑源码，入口会比较构建前后的清单，发现变化即拒绝交付该包。发布元信息来自 `other/project.json`，不在多处独立填写版本号。

Dockerfile 构建前端后将 `dist` 复制到后端嵌入目录，再编译正式程序。不要提交生成的嵌入页面，也不要将只包含后端代码的构建结果当作完整 Web 应用交付。

## 启动、排错与提交

Ubuntu 使用 `other/scripts/deploy.sh` 管理源码部署。`dev.sh`、`server.sh` 和 Makefile 的 run/dev 目标应按它们的实际委派关系理解，不能仅凭名字认定是前后端热更新开发服务器。前端 Vite 开发脚本存在，但不代替数据库、认证和 HTTPS 的完整运行条件。

Windows 使用 PowerShell 启动入口和 Docker Linux 容器。它负责后台启动及保存进程归属信息，重启前会核对已有实例，所需端口仍来自共享端口表。

提交前检查 `git diff --check`、工作区清单和忽略规则。仅提交源码、必要测试及有效文档，不提交 `other/.env`、`other/data`、`.validation`、`installers` 或第三方依赖目录。检查通过后再次同步远端，遵守现有提交策略，不改写已存在的远端记录。
