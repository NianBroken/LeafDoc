# LeafDoc 项目与使用说明

LeafDoc 是自行部署的在线 Markdown 文档系统。用户在浏览器建立章节和页面、编辑正文、上传附件、搜索内容，并通过账户角色控制编辑和管理权限。它适合项目手册、团队知识库和操作指南，将分散文本组织成可展开的目录，读者无需操作 Git 才能阅读和修改。

系统由 React 界面、Go 业务服务、Go 网关和 MySQL 组成。数据库保存页面、正文、账户和相关状态，实例目录保存安装资料、附件、头像、品牌资源及快照。页面不是扫描 Markdown 目录生成，迁移必须同时保留数据库和实例文件。

## 实际价值与功能

| 使用困难 | 项目处理方式 | 操作与结果 |
| --- | --- | --- |
| 内容分散难找 | 章节树、搜索、标签和属性 | 展开侧栏或检索进入页面 |
| 格式不易掌握 | 编辑器、预览和十篇默认教程 | 输入语法检查显示，保存后重新打开 |
| 多人共享权限不同 | 管理员、编辑者、阅读者和公开访问设置 | 管理员分配角色，编辑者维护正文 |
| 修改后需要追溯 | 可启用历史版本、比较和恢复 | 查看旧版本并恢复 |
| 大批文档需要迁入 | ZIP 导入计划和执行 | 先检查冲突，再执行导入 |
| 页面移动影响引用 | 稳定引用与可配置链接重构 | 检查重构预览和引用结果 |

| 功能 | 使用入口与实际行为 |
| --- | --- |
| 页面和章节 | 在侧栏创建。章节承载正文和子页面，普通页面不能有子节点 |
| 编辑 | 工具栏进入编辑，保存时检查页面版本，避免无条件覆盖并发修改 |
| 页面管理 | 重命名、移动、排序、复制、类型转换及删除，章节删除涉及子树 |
| 导航 | 页面树、面包屑、置顶与个人收藏 |
| 附件 | 上传、列出、重命名、删除和正文引用，上传大小受配置限制 |
| Markdown | 标题、列表、任务、表格、代码高亮、图片、数学块和 Mermaid。HTML 清理，单美元行内数学关闭 |
| 检索 | 标题和正文搜索，标签与属性查询 |
| 导入 | 上传 ZIP 建立计划、检查计划、执行或取消，上传不等于完成 |
| 历史 | 开启后查询版本、比较、恢复正文与历史附件 |
| 快照 | 管理员管理快照和恢复任务，恢复期间有写入限制与进度查询 |
| 账户 | 登录、注销、会话刷新、密码修改、用户管理、邀请和找回密码。邮件流程要求 SMTP |
| API 密钥 | 管理员通过 Cookie 会话创建和撤销，只调用允许该身份的接口 |
| 设置 | 品牌、公开访问、目录显示和实例选项 |
| 运维 | 健康检查与可配置指标，健康成功不能替代登录验证 |

功能受角色、实例设置和启动开关控制，未启用的能力不能视为已经可用。

## Ubuntu Docker 起步

以下命令在 Ubuntu 宿主机执行，准备 Git、Node.js、Docker Engine 和 Compose 插件。Node 调用发行脚本，前后端依赖安装与编译在 Docker 内完成。

```sh
git clone https://github.com/NianBroken/LeafDoc.git
cd LeafDoc
git pull --ff-only
docker info
docker compose version
node other/scripts/package.mjs
ls -lh installers/*.tar.gz
```

终端输出归档路径、压缩包 SHA256 与源码摘要。选择具体产物，不同时解压多个版本。

```sh
read -r -p '发行包完整路径: ' PACKAGE
mkdir -p "$HOME/leafdoc-release"
tar -xzf "$PACKAGE" -C "$HOME/leafdoc-release"
cd "$HOME/leafdoc-release"
sha256sum -c SHA256SUMS
sudo sh install.sh install
```

离线包包含应用镜像，不包含 MySQL。先准备应用容器可访问的数据库地址、端口、凭据与所需权限。容器 localhost 指向容器自身，不能代表宿主数据库。

安装保存 `/opt/leafdoc`、`/etc/leafdoc` 和 `/var/lib/leafdoc`，后台启动 app 与 gateway。首次令牌由脚本显示，也保存在 `/var/lib/leafdoc/setup-token`。打开 `http://服务器地址:24691` 或同端口 HTTPS，按向导验证令牌、配置和检查数据库、设置管理员并完成安装。随后使用自己设置的账户登录，没有统一默认密码。

登录后阅读教程，保存练习页面并重新打开。共享端口表位于 `backend/internal/core/serviceports/ports.json`，当前前端 24691、后端 24692，程序校验固定端口，不接受随意改号。带路径前缀的部署必须在访问地址加入前缀。

HTTPS 证书由准备流程生成。执行 `sudo /opt/leafdoc/manage.sh ca` 获取 CA 路径，在客户端信任 CA 后访问匹配证书的地址。HTTP 和 HTTPS 均支持认证，公网应使用可信证书。

## Ubuntu 其他入口

带数据库的源码 Compose 和离线包使用不同配置目录，不混用。

```sh
node other/scripts/container.mjs build
sudo bash other/scripts/deploy.sh init --hosts localhost,127.0.0.1
sudo bash other/scripts/deploy.sh start
sudo bash other/scripts/deploy.sh status
```

此路径包含 database、app 和 gateway，配置默认在 `/etc/leafdoc-compose/environment`，数据使用命名卷。局域网 HTTPS 应把真实服务器地址加入 hosts。重复初始化保留密码和密钥，启动脚本会检查管理员登录。

原生 systemd 入口为 `other/scripts/server.sh`，需先准备程序产物和 MySQL 包，详见 Ubuntu 部署文档。

## 架构与运行流程

1. 容器入口读取配置，启动业务程序或 gateway。
2. 服务检查安装状态，未安装实例进入向导，安装后连接 MySQL。
3. 浏览器访问网关，请求交给业务服务，同一监听端口识别 HTTP 与 HTTPS。
4. API 校验身份、角色和输入。页面写入通过工作单元和保存编排，更新相关链接、检索、标签与历史。
5. 前端读取树和正文，由预览组件转换 Markdown。
6. 空实例创建三篇一级章节与七篇二级教程，已初始化实例不会因更新自动清空。

| 代码位置 | 职责 |
| --- | --- |
| frontend/src/app | 应用装配和路由 |
| frontend/src/features | 编辑、阅读、树、用户和设置界面 |
| frontend/src/shared | API、公共组件、语言和规则 |
| backend/cmd | 参数、启动和网关 |
| backend/internal/wiki | 路由、用例与保存装配 |
| backend/internal/core | 树、数据库、认证、搜索、附件、历史和快照 |
| other/scripts | 构建、安装、配置、启动与检查 |

## 框架、技术与依赖

| 技术 | 用途 | 获取方式 |
| --- | --- | --- |
| Go、Gin、urfave/cli | HTTP、命令行和服务启动 | Docker 按 go.mod 和 go.sum 下载 |
| MySQL、go-sql-driver/mysql | 页面、账户和状态存储 | 外部数据库，Go 驱动 |
| cmux | 同端口双协议 | Go 模块 |
| JWT、x/crypto | 会话签名与密码处理 | Go 模块 |
| goldmark、bluemonday | Markdown 与 HTML 清理 | Go 模块 |
| React、TypeScript、Vite | 界面、类型检查和构建 | npm ci 按锁文件 |
| CodeMirror | Markdown 编辑器 | 前端依赖 |
| React Router、Zustand | 路由与状态 | 前端依赖 |
| Radix、Tailwind、Lucide | 组件、样式与图标 | 前端依赖 |
| react-markdown、remark、rehype | 解析、GFM、清理与高亮 | 前端依赖 |
| KaTeX、Mermaid | 数学块和图表 | 前端依赖 |
| i18next、Prometheus | 界面语言和后端指标 | 前端与 Go 依赖 |
| Docker、Compose、systemd | 构建及后台运行 | Ubuntu 系统工具 |

精确版本以锁文件和 Dockerfile 为准。正式运行无需启动 Vite，也不需要容器内保留编译器。

## 接口、配置与维护

接口位于 `/api`，前面加入实例路径前缀。GET /api/health 检查响应，GET /api/config 提供前端配置，POST /api/auth/login 使用 identifier 和 password。Cookie 写请求需要 CSRF，API 密钥不能替代全部管理权限。

配置分属 Compose、环境文件、安装记录与管理员设置。密码和私钥不写入项目文档。

```sh
sudo /opt/leafdoc/manage.sh status
sudo /opt/leafdoc/manage.sh logs
sudo /opt/leafdoc/manage.sh stop
sudo /opt/leafdoc/manage.sh start
sudo /opt/leafdoc/manage.sh restart
sudo systemctl enable --now docker
```

主动 stop 后 unless-stopped 不强行恢复，start 恢复运行。更新使用新包 upgrade，只 restart 不加载新镜像或环境。purge 用于删除实例，不能用于更新。

备份覆盖数据库和实例文件，并协调写入时间，先验证恢复。HTTPS 问题检查 CA 与证书地址，数据库问题检查容器可达地址和权限，登录问题检查账户、Cookie、前缀和日志。健康成功不代表业务验证完成。

## 详细文档

- [Docker 打包部署](other/docker-packaging.md)
- [Docker 更新](other/docker-update.md)
- [Ubuntu 部署](other/ubuntu-deployment.md)
- [完整配置表](other/configuration.md)
- [接口调用](other/api.md)
- [开发维护](other/development.md)
- [fnOS 部署](other/fnos-deployment.md)
