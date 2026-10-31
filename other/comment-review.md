# LeafDoc 逐文件注释核验清单

## 范围与状态

本清单登记项目文件的阅读、注释处理与验证结果。待阅读表示尚未完成全文核实，不代表已满足注释要求。

源码和支持注释的配置在原文件内说明。严格 JSON、锁文件、二进制资源和测试样本在本清单逐项记录。AGENTS.md 与 Git 策略钩子保持原样。依赖目录、Git 内部文件、构建产物、缓存及运行数据不作为项目源码修改。

## 文件记录

| 文件 | 类型 | 全文阅读 | 处理与事实依据 | 核验 |
| --- | --- | --- | --- | --- |
| `.dockerignore` | .dockerignore | 已全文阅读 | Docker 构建上下文排除规则，已按实际路径规则重写说明，未改变匹配项。 | 待统一验收 |
| `.gitattributes` | .gitattributes | 已全文阅读 | 自动识别文本并统一 LF 行尾，说明文本识别与二进制边界。 | 待统一验收 |
| `.gitignore` | .gitignore | 已全文阅读 | 本地产物忽略规则，英文人工说明已改写，匹配规则保持一致。 | 待统一验收 |
| `AGENTS.md` | .md | 已全文阅读 | 项目执行约束文件，全文读取，按保护要求保持原样。 | 待统一验收 |
| `backend/cmd/leafdoc/cli.go` | .go | 已全文阅读 | 根命令、参数来源、环境变量优先级、维护子命令和用法错误处理。已有中文函数说明已逐段核实，用户可见 Usage 字符串保持原产品文案。 | 待统一验收 |
| `backend/cmd/leafdoc/flags_auth.go` | .go | 已全文阅读 | 认证与公开访问参数声明，字段绑定环境来源、默认值和校验。源码注释已核实，Usage 属于产品帮助文本保持原样。 | 待统一验收 |
| `backend/cmd/leafdoc/flags_backup.go` | .go | 已全文阅读 | 快照和恢复相关参数声明，说明开关、路径和大小约束。源码注释已核实，Usage 属于产品帮助文本保持原样。 | 待统一验收 |
| `backend/cmd/leafdoc/flags_email.go` | .go | 已全文阅读 | SMTP 邮件参数声明和安全模式校验入口。源码注释已核实，Usage 属于产品帮助文本保持原样。 | 待统一验收 |
| `backend/cmd/leafdoc/flags_frontend.go` | .go | 已全文阅读 | 前端语言、上传限制和资源配置参数声明。源码注释已核实，Usage 属于产品帮助文本保持原样。 | 待统一验收 |
| `backend/cmd/leafdoc/flags_metrics.go` | .go | 已全文阅读 | Prometheus 指标监听参数声明。源码注释已核实，Usage 属于产品帮助文本保持原样。 | 待统一验收 |
| `backend/cmd/leafdoc/flags_naming_test.go` | .go | 已全文阅读 | 核对命令行参数命名与结构字段的一致性，失败时报告具体字段。 | 待统一验收 |
| `backend/cmd/leafdoc/flags_proxy.go` | .go | 已全文阅读 | 反向代理认证、可信代理和登出地址参数声明。源码注释已核实，Usage 属于产品帮助文本保持原样。 | 待统一验收 |
| `backend/cmd/leafdoc/flags_server.go` | .go | 已全文阅读 | 服务监听、日志和数据目录参数声明。源码注释已核实，Usage 属于产品帮助文本保持原样。 | 待统一验收 |
| `backend/cmd/leafdoc/main.go` | .go | 已全文阅读 | 执行命令动作，构造服务依赖，启动业务与指标监听并处理优雅关停。已有中文函数说明覆盖参数校验、资源生命周期和错误路径。 | 待统一验收 |
| `backend/cmd/leafdoc/main_test.go` | .go | 已全文阅读 | 通过真实命令树验证帮助、参数优先级、校验、日志、快照命令、监听器与生命周期关闭。测试说明已逐例核实。 | 待统一验收 |
| `backend/cmd/leafdoc/ports_test.go` | .go | 已全文阅读 | 验证固定服务端口参数只接受共享分配值并拒绝临时端口和 Unix 套接字。 | 待统一验收 |
| `backend/cmd/leafdoc/signals_unix.go` | .go | 已全文阅读 | 非 Windows 构建登记 SIGUSR1 和 SIGHUP 到调用方通道，生命周期撤销由公共启动入口承担，构建约束原样保留。 | 待统一验收 |
| `backend/cmd/leafdoc/signals_windows.go` | .go | 已全文阅读 | Windows 平台同名信号入口为空操作，不登记不支持的 Unix 重载信号，构建约束保持原样。 | 待统一验收 |
| `backend/cmd/schema-defaults/main.go` | .go | 已全文阅读 | 从统一结构和默认设置生成数据库默认 SQL 与浏览器默认值，保持生成字节格式。 | 待统一验收 |
| `backend/cmd/service-ports/main.go` | .go | 已全文阅读 | 命令参数逐项作为服务名传入共享端口释放，失败写标准错误并退出一，成功无输出。 | 待统一验收 |
| `backend/go.mod` | .mod | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `backend/go.sum` | .sum | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `backend/internal/architecture_test.go` | .go | 已全文阅读 | 解析生产 Go 文件导入并逐项执行模块边界规则，测试允许和拒绝的依赖关系。 | 待统一验收 |
| `backend/internal/core/assets/assets_service.go` | .go | 已全文阅读 | 维护页面附件目录、命名校验、保存、列举、删除、重命名和复制操作，并应用页面忽略规则。 | 待统一验收 |
| `backend/internal/core/assets/assets_service_test.go` | .go | 已全文阅读 | 核对附件文件名、目录、保存列举、删除重命名复制和忽略规则的实际约束。 | 待统一验收 |
| `backend/internal/core/auth/apikey_service.go` | .go | 已全文阅读 | 密钥生成、摘要比较、所属账号读取和角色交集，核对前缀冲突最多两次及五分钟使用时间节流。 | 待统一验收 |
| `backend/internal/core/auth/apikey_service_test.go` | .go | 已全文阅读 | 测试密钥格式、摘要验证与角色访问组合，使用独立输入核对服务辅助逻辑，不通过数据库验证持久化。 | 待统一验收 |
| `backend/internal/core/auth/apikey_store.go` | .go | 已全文阅读 | 密钥表字段映射、可空秒级时间、停用状态与 SQL 操作，核对重复键统一映射及游标释放。 | 待统一验收 |
| `backend/internal/core/auth/apikey_store_test.go` | .go | 已全文阅读 | 测试密钥到期时间的边界判断，覆盖有效期缺省与指定时间，不建立真实数据库连接。 | 待统一验收 |
| `backend/internal/core/auth/auth_service.go` | .go | 已全文阅读 | 认证流程编排，逐段核实账号查询、尝试计数、占位哈希、签发与服务替换，说明签发失败不恢复计数。 | 待统一验收 |
| `backend/internal/core/auth/email_token_service.go` | .go | 已全文阅读 | 邮件令牌服务签发邀请与重设密码链接，关闭时等待邮件任务结束。令牌消费、密码保存和会话撤销分别执行，未以共同事务包裹。 | 待统一验收 |
| `backend/internal/core/auth/email_token_store.go` | .go | 已全文阅读 | 令牌以标识与秘密值组成，对数据库只保存秘密摘要。解析核对用途与有效期，消费通过条件更新保证单次使用，清理任务由关闭信号结束。 | 待统一验收 |
| `backend/internal/core/auth/errors.go` | .go | 已全文阅读 | 认证错误实例与稳定错误码，仅声明错误，不在定义处执行校验；人工说明与实际定义一致。 | 待统一验收 |
| `backend/internal/core/auth/login_attempt_tracker.go` | .go | 已全文阅读 | 进程内尝试次数与锁定，核实第五次仍校验密码、锁定期间不延长截止时间。 | 待统一验收 |
| `backend/internal/core/auth/maintenance/adminreset.go` | .go | 已全文阅读 | 管理员维护依次设置密码与撤销会话，两个阶段不在同一事务中。 | 待统一验收 |
| `backend/internal/core/auth/models.go` | .go | 已全文阅读 | 内部用户与公开用户表示，说明字段复制、密码排除和两个密码状态。 | 待统一验收 |
| `backend/internal/core/auth/session_manager.go` | .go | 已全文阅读 | JWT 签发、代际核对和刷新轮换，逐项核对声明、数据库会话与先插入后撤销的独立操作。 | 待统一验收 |
| `backend/internal/core/auth/session_store.go` | .go | 已全文阅读 | 刷新会话表与代际表的读写、每小时清理及关闭等待，说明有效期比较和清理边界不同。 | 待统一验收 |
| `backend/internal/core/auth/user_resolver.go` | .go | 已全文阅读 | 用户标签缓存，核实缺失缓存、并发写入优先级与整体重载的指针边界。 | 待统一验收 |
| `backend/internal/core/auth/user_service.go` | .go | 已全文阅读 | 用户创建、角色与密码操作由服务编排。账号查找错误与密码摘要不匹配分别映射，相关错误说明已按实际分支重写。 | 待统一验收 |
| `backend/internal/core/auth/user_store.go` | .go | 已全文阅读 | 用户 SQL 存取使用统一列集合，查询连接只在获取连接时持锁。管理员更新包含数量约束，密码更新清除强制改密标记。 | 待统一验收 |
| `backend/internal/core/avatar/avatar_service.go` | .go | 已全文阅读 | 校验用户路径、扩展名与字节上限后解码，居中裁图并缩放成 256 像素 PNG，原子写入固定用户路径，删除不存在文件也成功。 | 待统一验收 |
| `backend/internal/core/avatar/avatar_service_test.go` | .go | 已全文阅读 | 内存图像与临时目录核验多格式上传、真实编码超限、中心裁剪、删除幂等和路径拒绝，四色断言比较集合而非逐角绑定。 | 待统一验收 |
| `backend/internal/core/brandassets/assets.go` | .go | 已全文阅读 | 默认品牌名称与嵌入资源读取，通配符收集资源，Read 只拼接嵌入路径。 | 待统一验收 |
| `backend/internal/core/brandassets/files/leafdoc-avatar.webp` | .webp | 已全文阅读 | 二进制资源或占位文件，已读取元数据和哈希并核对引用位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/brandassets/files/leafdoc-favicon.ico` | .ico | 已全文阅读 | 二进制资源或占位文件，已读取元数据和哈希并核对引用位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/brandassets/files/leafdoc-icon-180.png` | .png | 已全文阅读 | 二进制资源或占位文件，已读取元数据和哈希并核对引用位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/brandassets/files/leafdoc-logo.webp` | .webp | 已全文阅读 | 二进制资源或占位文件，已读取元数据和哈希并核对引用位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/branding/branding_service.go` | .go | 已全文阅读 | 服务锁协调外观配置和资源文件，发现可用资源不自动保存配置；上传按写盘、清理其他资源、存配置顺序，后续失败不回滚文件。 | 待统一验收 |
| `backend/internal/core/branding/branding_service_test.go` | .go | 已全文阅读 | Windows 专属用例核对带卷标的相对文件名被拒绝，其他系统按明确平台条件跳过。 | 待统一验收 |
| `backend/internal/core/branding/branding_store.go` | .go | 已全文阅读 | 品牌配置存于通用 MySQL 设置记录，加载与写入缓存后注入上传约束，资源目录由实例目录计算。 | 待统一验收 |
| `backend/internal/core/branding/models.go` | .go | 已全文阅读 | 站点名称、资源文件名与上传限制模型，扩展名响应排序但不筛布尔值，名称长度按字节限制，默认映射每次独立创建。 | 待统一验收 |
| `backend/internal/core/branding/utils.go` | .go | 已全文阅读 | 展开通配路径后保留指定文件并逐项删除其他项，匹配和删除失败只记录日志，不递归删除。 | 待统一验收 |
| `backend/internal/core/database/backup.go` | .go | 已全文阅读 | 按表名导出完整字节载荷，校验表列后事务导入，清空而不导入会话并重建认证代次，调用方持有写入屏障与提交权。 | 待统一验收 |
| `backend/internal/core/database/backup_test.go` | .go | 已全文阅读 | 公开接口拒绝缺少格式、未知格式与非法表名的备份载荷，断言错误存在，不连接数据库。 | 待统一验收 |
| `backend/internal/core/database/database.go` | .go | 已全文阅读 | 从最近父目录 .env 读取并按进程变量覆盖连接参数，独立池与共享池分开管理，初始化缓存按完整 DSN 区分。 | 待统一验收 |
| `backend/internal/core/database/default.sql` | .sql | 已全文阅读 | 生成的 MySQL 完整 DDL 和缺失设置种子，来源为 container.mjs schema-write。全文核对表、索引、空值与默认配置；不手工变更生成结果。 | 待统一验收 |
| `backend/internal/core/database/executor.go` | .go | 已全文阅读 | 连接池与事务执行接口，只有具体 sql.DB 开启事务；借用事务不提交外层，记录游标与回滚错误合并返回。全文核实既有说明。 | 待统一验收 |
| `backend/internal/core/database/schema_stores.go` | .go | 已全文阅读 | 内容表与关系表建表语句的共同来源，串行执行失败即返回，不迁移已存在列，不以单一事务包住建表。 | 待统一验收 |
| `backend/internal/core/defaultdocs/content/markdown-extensions-and-troubleshooting.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/defaultdocs/content/markdown-getting-started.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/defaultdocs/content/markdown-links-images-and-tables.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/defaultdocs/content/markdown-lists-and-structure.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/defaultdocs/content/markdown-text-and-code.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/defaultdocs/documents.go` | .go | 已全文阅读 | 明确列出五篇教程并按序读取正文，不负责持久化或 Markdown 渲染。 | 待统一验收 |
| `backend/internal/core/defaultdocs/documents_test.go` | .go | 已全文阅读 | 解析教程链接和图片，核对资源存在、教程路径与 HTTPS 域名，不发起外链请求。 | 待统一验收 |
| `backend/internal/core/defaults/defaults.go` | .go | 已全文阅读 | 实例初始常量，区分默认参数与运行时数据库设置，运行账号未写入此文件。 | 待统一验收 |
| `backend/internal/core/email/client.go` | .go | 已全文阅读 | SMTP 连接、TLS 或 STARTTLS、条件认证与信封发送，期限写入连接；MIME 同时容纳纯文本和 HTML，头字段删除回车换行。 | 待统一验收 |
| `backend/internal/core/email/client_test.go` | .go | 已全文阅读 | 单连接本地 SMTP 记录真实信封与正文，核验头注入被删除；不回应握手的对端验证 TLS 期限、取消及连接释放。 | 待统一验收 |
| `backend/internal/core/email/config.go` | .go | 已全文阅读 | SMTP 配置按主机是否为空判断启用，安全模式与认证分别表达，非正超时使用十秒。 | 待统一验收 |
| `backend/internal/core/email/config_test.go` | .go | 已全文阅读 | 核对空主机禁用、非空主机启用和超时零值与正值，不连接邮件服务器。 | 待统一验收 |
| `backend/internal/core/email/service.go` | .go | 已全文阅读 | 按全局语言构造邀请与重置的纯文本和转义 HTML，为每次发送创建期限并延迟取消，不重试发送。 | 待统一验收 |
| `backend/internal/core/email/templates_test.go` | .go | 已全文阅读 | 五语言与邀请重置组合核对链接保留、脚本转义和操作文案，结束后语言提供器设为中文。 | 待统一验收 |
| `backend/internal/core/excerpt/excerpt.go` | .go | 已全文阅读 | 摘要移除代码，搜索保留普通代码而排除 Mermaid；围栏外处理双链与图片，提示块规范化后经 Markdown 和 HTML 清理形成文字。 | 待统一验收 |
| `backend/internal/core/excerpt/excerpt_test.go` | .go | 已全文阅读 | 逐例核验前置字段、摘要截断、空白合并、双链别名、代码保护、Mermaid 排除及引用块围栏，输入字符串保留原值。 | 待统一验收 |
| `backend/internal/core/favorites/favorites_store.go` | .go | 已全文阅读 | 按用户与页面保存收藏，重复添加保留原时间；普通连接串行访问并受暂停状态约束，外部事务视图不拥有提交或关闭权。 | 待统一验收 |
| `backend/internal/core/ignore/ignore.go` | .go | 已全文阅读 | 逐目录读取规则并交给匹配库，缓存按祖先到子层合并且不自动失效；读取失败在缓存入口跳过，独立读取入口保留错误。 | 待统一验收 |
| `backend/internal/core/ignore/ignore_test.go` | .go | 已全文阅读 | 临时目录核验目录专属、根锚定、否定、跨层通配与规则计数，分别检查缺失、目录占位、缓存复用和不完整模式加载。 | 待统一验收 |
| `backend/internal/core/importer/asset_transformer.go` | .go | 已全文阅读 | 按工作区或源文档目录定位附件，绝对路径层级检查不跟随符号链接；每页面每源文件缓存上传路径，扩展名分类不读取图像内容。 | 待统一验收 |
| `backend/internal/core/importer/content_transformer.go` | .go | 已全文阅读 | 计划索引依次支持源路径、唯一名称与唯一路径后缀，目标转换保留后缀；手写扫描器保护代码，不声称实现完整 Markdown 语法。 | 待统一验收 |
| `backend/internal/core/importer/content_transformer_test.go` | .go | 已全文阅读 | 表驱动验证导入正文的元信息保留、正文重写和不同语法处理。 | 待统一验收 |
| `backend/internal/core/importer/executor.go` | .go | 已全文阅读 | 按计划顺序创建页面、转换正文、上传附件、报告进度并处理取消、跳过和失败结果。 | 待统一验收 |
| `backend/internal/core/importer/executor_test.go` | .go | 已全文阅读 | 通过可记录的 Wiki 替身逐项验证导入执行器的摘要检查、动作分支、链接资源重写、错误累计和路径处理。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/fixture-1.zip` | .zip | 已全文阅读 | 二进制资源或占位文件，已读取元数据和哈希并核对引用位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/leafdoc-nested-package/docs/getting-started.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/leafdoc-nested-package/docs/guides/basic-guide.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/leafdoc-nested-package/docs/guides/index.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/leafdoc-nested-package/docs/index.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/leafdoc-nested-package/intro.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/link-assets-package/Guides/Setup.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/link-assets-package/Guides/images/logo.png` | .png | 已全文阅读 | 二进制资源或占位文件，已读取元数据和哈希并核对引用位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/link-assets-package/Guides/index.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/link-assets-package/README.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/link-assets-package/Reference/API.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/link-assets-package/Reference/Endpoints.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/link-assets-package/shared/manual.pdf` | .pdf | 已全文阅读 | 二进制资源或占位文件，已读取元数据和哈希并核对引用位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/obsidian-wikilinks-package/Archive/Meeting Notes.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/obsidian-wikilinks-package/Attachments/diagram.png` | .png | 已全文阅读 | 二进制资源或占位文件，已读取元数据和哈希并核对引用位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/obsidian-wikilinks-package/Daily/Brainstorm.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/obsidian-wikilinks-package/Daily/Meeting Notes.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/obsidian-wikilinks-package/Home.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/fixtures/obsidian-wikilinks-package/Project Plan.md` | .md | 已全文阅读 | 文档正文或导入测试样本，已全文读取并核对嵌入及测试引用。正文属于产品或测试输入，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/importer/importer_service.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/importer/importer_service_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/importer/importer_wiki.go` | .go | 已全文阅读 | 导入器通过最小接口读取树摘要、查路径、建路径、写页面和上传附件，身份与限制随调用传递。 | 待统一验收 |
| `backend/internal/core/importer/markdown_scanner.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/importer/plan_store.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/importer/plan_store_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/importer/planner.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/importer/planner_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/importer/wikilink_transformer.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/importer/zip_extractor.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/importer/zip_extractor_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/importer/zip_workspace.go` | .go | 已全文阅读 | 临时解压目录供导入阶段共享，空对象或空路径无需清理，其他路径交给 RemoveAll 并返回删除错误。 | 待统一验收 |
| `backend/internal/core/links/backlink.go` | .go | 已全文阅读 | 入链记录保存来源与目标身份，接口项另外提供从当前树读取的来源路径，结果数量独立保存。 | 待统一验收 |
| `backend/internal/core/links/helpers.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/links/link.go` | .go | 已全文阅读 | 按入链出链与失效分组定义响应及数量，结构不执行链接提取与数据库查询。 | 待统一验收 |
| `backend/internal/core/links/link_refactor.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/links/link_refactor_ast_test.go` | .go | 已全文阅读 | 正文含相同代码与真实链接时仅重写真正文，引用式链接不重写且报告警告。 | 待统一验收 |
| `backend/internal/core/links/link_refactor_replace_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/links/link_refactor_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/links/link_refactor_wikilinks_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/links/link_service.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/links/link_service_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/links/link_state_test.go` | .go | 已全文阅读 | 零、一、二、五个标题匹配分别核对失效、解析与歧义分类，不读取页面树。 | 待统一验收 |
| `backend/internal/core/links/links_store.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/links/outgoing.go` | .go | 已全文阅读 | 出链存储携带解析状态，接口 Broken 仅映射失效状态，标题歧义不等同失效。 | 待统一验收 |
| `backend/internal/core/links/preview_regex_bench_test.go` | .go | 已全文阅读 | 基准对比单页与百页预览，区分每页编译与每批编译，均报告分配且不作为正确性断言。 | 待统一验收 |
| `backend/internal/core/links/refactor.go` | .go | 已全文阅读 | 重构候选保存来源与目标路径，规则分别携带路径与标题映射，不在结构层解析或校验。 | 待统一验收 |
| `backend/internal/core/links/wikilink_fast_path_test.go` | .go | 已全文阅读 | 基准比较无双链、正文双链和仅代码双链的提取成本，样本循环外生成，不作功能结果断言。 | 待统一验收 |
| `backend/internal/core/localization/de.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/localization/en.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/localization/es.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/localization/fr.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/localization/mail.go` | .go | 已全文阅读 | 五种语言邮件文字，核实邀请索引与语言缺失回退，不改动产品翻译文本。 | 待统一验收 |
| `backend/internal/core/localization/messages.go` | .go | 已全文阅读 | 嵌入五种语言词条，核实语言提供器、模板查找、数字替换与缺词返回原文。 | 待统一验收 |
| `backend/internal/core/localization/zh.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `backend/internal/core/markdown/errors.go` | .go | 已全文阅读 | 声明页首解析分类错误，供调用方与 YAML 原因共同组成可识别错误链。 | 待统一验收 |
| `backend/internal/core/markdown/frontmatter.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/markdown/frontmatter_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/markdown/markdown.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/markdown/markdown_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/markdown/reserved.go` | .go | 已全文阅读 | trim 后小写比较 tags 与 leafdoc_ 前缀，只分类不删除字段或直接执行权限检查。 | 待统一验收 |
| `backend/internal/core/markdown/reserved_test.go` | .go | 已全文阅读 | 逐表格输入验证 tags 大小写与 leafdoc_ 前缀，普通 title 和空键不属于系统保留键。 | 待统一验收 |
| `backend/internal/core/preferences/service.go` | .go | 已全文阅读 | 实例语言日期时间偏好，核实候选副本、枚举校验及空 patch 仍写入通知。 | 待统一验收 |
| `backend/internal/core/properties/properties_service.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/properties/properties_service_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/properties/properties_store.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/publicaccess/errors.go` | .go | 已全文阅读 | 环境托管公开开关的拒写使用固定业务码与翻译模板，不包装底层原因。 | 待统一验收 |
| `backend/internal/core/publicaccess/provider.go` | .go | 已全文阅读 | 公开访问只读接口和布尔适配，不在接口内执行请求权限判断。 | 待统一验收 |
| `backend/internal/core/publicaccess/service.go` | .go | 已全文阅读 | 固定或数据库托管的公开访问开关，核实只读错误转换与保存失败包装。 | 待统一验收 |
| `backend/internal/core/publicaccess/service_test.go` | .go | 已全文阅读 | 无数据库构造固定公开来源，核对读值、来源、拒写本地化错误和状态未变。 | 待统一验收 |
| `backend/internal/core/restore/config.go` | .go | 已全文阅读 | 恢复配置组合目录、快照、写门禁和重载依赖，运行态重载回调优先于重新扫描触发器。 | 待统一验收 |
| `backend/internal/core/restore/job.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/restore/job_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/restore/manager.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/restore/mysql_restore_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/restore/offline.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/restore/progress_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/restore/selfrestart_unix.go` | .go | 已全文阅读 | 非 Windows 构建用当前可执行文件、完整参数和环境替换进程映像，失败返回系统错误。 | 待统一验收 |
| `backend/internal/core/restore/selfrestart_unix_test.go` | .go | 已全文阅读 | 替换系统执行函数捕获路径参数环境，并核对错误原样返回，不实际结束测试进程。 | 待统一验收 |
| `backend/internal/core/restore/selfrestart_windows.go` | .go | 已全文阅读 | Windows 创建继承标准流的同程序子进程，Start 成功后退出当前进程，不等待健康检查或服务清理。 | 待统一验收 |
| `backend/internal/core/restore/swap.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/restore/writegate.go` | .go | 已全文阅读 | 写入准入与在途计数，核对加锁登记、单次离开约束和二十毫秒轮询，不把它描述为全部后台暂停。 | 待统一验收 |
| `backend/internal/core/restore/writegate_test.go` | .go | 已全文阅读 | 测试写门禁计数、关闭与排空等并发状态，围绕执行器进入和释放验证恢复期间的写入协调。 | 待统一验收 |
| `backend/internal/core/revision/assets.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/revision/mysql_store.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/revision/read.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/revision/record.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/revision/restore.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/revision/retention.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/revision/service.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/revision/types.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/search/indexing.go` | .go | 已全文阅读 | 单页入口委托批量索引并优先返回事务错误，删除批量使用语句与执行器事务，绑定执行器时借用外部提交范围。 | 待统一验收 |
| `backend/internal/core/search/indexing_status.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/search/mysql_index.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/search/mysql_index_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/search/query.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/search/search_result.go` | .go | 已全文阅读 | 定义分页结果、单页分数与转义摘要和标签统计，空标签字段省略不代表正文没有标签。 | 待统一验收 |
| `backend/internal/core/serviceports/ports.go` | .go | 已全文阅读 | 嵌入端口表查询和文本校验，核实每次解析、排序和未知服务 panic。 | 待统一验收 |
| `backend/internal/core/serviceports/ports.json` | .json | 已全文阅读 | 共享端口表，前端 24691、后端 24692、网关管理 24693；Go 嵌入、Vite 和启动脚本读取同一文件，其余键对应指标与测试服务。 | 待统一验收 |
| `backend/internal/core/serviceports/ports_test.go` | .go | 已全文阅读 | 核对端口范围、排序后相邻重复和端口文本拒绝案例。 | 待统一验收 |
| `backend/internal/core/serviceports/release.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/settings/events.go` | .go | 已全文阅读 | 进程内版本与容量一订阅通道，说明信号合并、取消不关闭通道和退出责任。 | 待统一验收 |
| `backend/internal/core/settings/store.go` | .go | 已全文阅读 | MySQL 配置记录和缓存，核实写入顺序、回调、顶层 null 检查和重载写回。 | 待统一验收 |
| `backend/internal/core/settings/value.go` | .go | 已全文阅读 | 统一固定值与数据库值接口，核实只读错误、重载空操作和浅复制语义。 | 待统一验收 |
| `backend/internal/core/settings/value_test.go` | .go | 已全文阅读 | 固定值读取、拒绝写入、重载和值来源测试，不连接数据库。 | 待统一验收 |
| `backend/internal/core/shared/atomic_file.go` | .go | 已全文阅读 | 临时文件写入与平台替换，核实同步顺序、失败清理与源流所有权。 | 待统一验收 |
| `backend/internal/core/shared/close.go` | .go | 已全文阅读 | 关闭函数失败记录到默认日志，不重试、不吞掉 panic。 | 待统一验收 |
| `backend/internal/core/shared/errors/field_error.go` | .go | 已全文阅读 | 字段错误有序集合，说明空切片构造与结构零值的不同以及条目计数。 | 待统一验收 |
| `backend/internal/core/shared/errors/localized_error.go` | .go | 已全文阅读 | 本地化描述与底层错误链，核实 nil 接收者、参数切片复制及 Cause 引用。 | 待统一验收 |
| `backend/internal/core/shared/extraction.go` | .go | 已全文阅读 | 限额流复制，核实额外一字节、预算扣减、整数比例和失败时已有输出。 | 待统一验收 |
| `backend/internal/core/shared/htmlutil/html.go` | .go | 已全文阅读 | 全文转义后按序替换显式标记，说明替换片段不再次转义。 | 待统一验收 |
| `backend/internal/core/shared/htmlutil/html_test.go` | .go | 已全文阅读 | 核对特殊字符实体、允许标记和空标记，断言直接比较输出文本。 | 待统一验收 |
| `backend/internal/core/shared/random.go` | .go | 已全文阅读 | 短 ID 与随机 ASCII 密码，核实长度边界、均匀下标和随机失败返回。 | 待统一验收 |
| `backend/internal/core/shared/replace_other.go` | .go | 已全文阅读 | 非 Windows 文件重命名实现，保留 go:build 原文并解释平台选择。 | 待统一验收 |
| `backend/internal/core/shared/replace_windows.go` | .go | 已全文阅读 | Windows 宽字符路径转换后 MoveFileEx 覆盖目标，两次转换均先于系统调用。 | 待统一验收 |
| `backend/internal/core/shared/utils_test.go` | .go | 已全文阅读 | 临时目录内检查原子写入、路径分隔符和三类解压限额错误，现有说明与断言对应。 | 待统一验收 |
| `backend/internal/core/snapshot/manager.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/snapshot/scheduler.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/snapshot/scheduler_test.go` | .go | 已全文阅读 | 固定时刻核对上海每日调度、恰好调度时刻和午夜边界，不启动真实计时器。 | 待统一验收 |
| `backend/internal/core/snapshot/settings.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/snapshot/settings_test.go` | .go | 已全文阅读 | 有效每日配置逐项改变保留数量、时区和小时，要求返回错误，不依赖 map 顺序。 | 待统一验收 |
| `backend/internal/core/snapshot/snapshot.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/snapshot/snapshot_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/snapshot/status.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/tags/excerpt.go` | .go | 已全文阅读 | 标签摘要委托公共 excerpt 包，提取原始含前置字段的正文，不独立定义截断规则。 | 待统一验收 |
| `backend/internal/core/tags/excerpt_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/tags/tags_service.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/tags/tags_service_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/tags/tags_store.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/tocdisplay/provider.go` | .go | 已全文阅读 | 固定布尔目录提供器，仅表达 always 和 auto，不表达 never。 | 待统一验收 |
| `backend/internal/core/tocdisplay/service.go` | .go | 已全文阅读 | 目录显示模式的反序列化优先级、布尔兼容和模式保存。 | 待统一验收 |
| `backend/internal/core/tree/ensure_path_result.go` | .go | 已全文阅读 | 确保路径结果保存目标节点、存在状态与已建节点引用，不复制页面内容。 | 待统一验收 |
| `backend/internal/core/tree/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/tree/filename_sort_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/tree/helpers.go` | .go | 已全文阅读 | 版本辅助自身不加锁，处理显式跳过与空节点版本，路径辅助递归保留根 slug，不读文件系统。 | 待统一验收 |
| `backend/internal/core/tree/helpers_test.go` | .go | 已全文阅读 | 内存节点核对缺版本、版本冲突、匹配和显式跳过，另验证祖先 slug 递归拼接包含虚拟根。 | 待统一验收 |
| `backend/internal/core/tree/node_store.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/tree/node_store_reconstruct.go` | .go | 已全文阅读 | 目录项按小写名稳定排序，名称相同再比原始文本，不区分目录文件或读取内容。 | 待统一验收 |
| `backend/internal/core/tree/node_store_write.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/tree/page.go` | .go | 已全文阅读 | 页面嵌入节点引用与正文，原始含前置字段内容不序列化，版本委托节点并接受空页面。 | 待统一验收 |
| `backend/internal/core/tree/page_node.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/tree/path_lookup.go` | .go | 已全文阅读 | 路径解析响应逐段描述 slug、存在、类型、标题与身份，指针字段区分缺失值，结构不执行创建。 | 待统一验收 |
| `backend/internal/core/tree/slug_service.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/tree/slug_service_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/tree/tree_index.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/tree/tree_mutations.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/tree/tree_reads.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/tree/tree_service.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/core/tree/unit_of_work.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/http/dist/.gitkeep` | .gitkeep | 已全文阅读 | 二进制资源或占位文件，已读取元数据和哈希并核对引用位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `backend/internal/http/dto/page.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/http/dto/property.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/http/dto/tag.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/http/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/http/export_test.go` | .go | 已全文阅读 | 测试构建才暴露 HTML 助手引用给外部测试包，两个变量直接引用生产函数，不复制实现。 | 待统一验收 |
| `backend/internal/http/frontend_template_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/http/localized_json.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/http/localized_json_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/http/metrics/metrics.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/http/metrics/metrics_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/http/metrics/runtime_collector.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/http/metrics/runtime_collector_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/http/middleware/auth/apikey.go` | .go | 已全文阅读 | Bearer 密钥识别、网络对端次数限制与成功标记，核对存储不可用单独返回和凭据管理限制。 | 待统一验收 |
| `backend/internal/http/middleware/auth/auth.go` | .go | 已全文阅读 | 严格身份解析、可选认证与角色门禁，说明已有上下文的优先级及可选认证不核对其类型。 | 待统一验收 |
| `backend/internal/http/middleware/auth/auth_cookie.go` | .go | 已全文阅读 | 按连接模式选择认证 Cookie 名称，核实 HttpOnly、有效期和删除不撤销会话。 | 待统一验收 |
| `backend/internal/http/middleware/auth/current_user.go` | .go | 已全文阅读 | 读取上下文用户与密钥标记，核对缺失值、类型错误、空指针与中止响应的差别。 | 待统一验收 |
| `backend/internal/http/middleware/auth/password_change.go` | .go | 已全文阅读 | 强制改密路径门禁，核对接口后缀匹配、普通页面放行以及不检查首次设密标记。 | 待统一验收 |
| `backend/internal/http/middleware/auth/require_auth_or_public.go` | .go | 已全文阅读 | 公开读取门禁只对未认证结果使用实时开关，服务器错误仍拒绝，HTTP 方法由路由限定。 | 待统一验收 |
| `backend/internal/http/middleware/auth/require_auth_or_public_test.go` | .go | 已全文阅读 | 通过模拟请求验证公开访问和身份门禁，测试正文说明准备的配置、身份和对应响应。 | 待统一验收 |
| `backend/internal/http/middleware/auth/reverse_proxy.go` | .go | 已全文阅读 | 受信代理身份注入与按配置创建账号，核对来源、空身份、邮箱冲突及不签发 Cookie。 | 待统一验收 |
| `backend/internal/http/middleware/auth/trusted_proxies.go` | .go | 已全文阅读 | 逗号分隔 IP 与 CIDR 白名单，核对空项、无效地址、连接端口拆分与网段匹配。 | 待统一验收 |
| `backend/internal/http/middleware/auth/trusted_proxies_test.go` | .go | 已全文阅读 | 测试可信代理地址解析与拒绝行为，覆盖地址白名单和转发链选择，文件说明指明测试的请求环境。 | 待统一验收 |
| `backend/internal/http/middleware/maintenance/writegate.go` | .go | 已全文阅读 | HTTP 写请求登记与恢复豁免路径，核对路径段连续匹配和延迟释放计数。 | 待统一验收 |
| `backend/internal/http/middleware/maintenance/writegate_test.go` | .go | 已全文阅读 | 验证维护写门禁对 HTTP 方法与维护状态的组合处理，区分读取、写入和拒绝响应。 | 待统一验收 |
| `backend/internal/http/middleware/security/csrf.go` | .go | 已全文阅读 | 双重提交令牌核对，核对安全方法、已解析密钥豁免、请求头优先和常量时间比较。 | 待统一验收 |
| `backend/internal/http/middleware/security/csrf_cookie.go` | .go | 已全文阅读 | 可读 CSRF Cookie 的生成复用和清除，核实复用不续期与响应头输出。 | 待统一验收 |
| `backend/internal/http/middleware/security/csrf_cookie_test.go` | .go | 已全文阅读 | 验证 CSRF Cookie 的生成、已有值复用及响应属性，Cookie 复用路径不续期，使用实际请求头与响应头断言。 | 待统一验收 |
| `backend/internal/http/middleware/security/csrf_test.go` | .go | 已全文阅读 | 验证写请求的 Cookie 与请求头比较、公开路径及读取请求的处理，测试说明区分放行和拒绝条件。 | 待统一验收 |
| `backend/internal/http/middleware/security/rate_limiter.go` | .go | 已全文阅读 | 进程内滑动窗口计数，核对严格窗口边界、拒绝不登记、成功清空与 RemoteAddr 来源。 | 待统一验收 |
| `backend/internal/http/middleware/security/rate_limiter_test.go` | .go | 已全文阅读 | 限流测试按独立来源与固定时钟核对计数和窗口，文件头说明内存状态边界。 | 待统一验收 |
| `backend/internal/http/middleware/security/untrusted_file.go` | .go | 已全文阅读 | 上传文件响应头，核对扩展名内联集合、PDF 的 CSP 分支和文件名编码，不推断文件正文。 | 待统一验收 |
| `backend/internal/http/middleware/security/untrusted_file_test.go` | .go | 已全文阅读 | 核对不可信文件的响应安全头与路径行为，测试不承担内容扫描和文件转换。 | 待统一验收 |
| `backend/internal/http/middleware/utils/require_secure.go` | .go | 已全文阅读 | Cookie 安全连接判断，核对直接 TLS、代理头字符串规则与允许明文的返回值。 | 待统一验收 |
| `backend/internal/http/registrar.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/http/router.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/http/router_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/http/settings_events.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/http/upload.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/testutil/common.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/apikeys/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/apikeys/errors_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/apikeys/routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/apikeys/use_cases.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/apikeys/use_cases_test.go` | .go | 已全文阅读 | 三个独立用例分别确认创建、列表和撤销的 nil 服务分支都返回 ErrAPIKeysDisabled。 | 待统一验收 |
| `backend/internal/wiki/assets/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/assets/routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/assets/serve_headers.go` | .go | 已全文阅读 | 按实际请求路径调用共享不可信附件头策略后继续中间件链，本文件没有独立后缀规则。 | 待统一验收 |
| `backend/internal/wiki/assets/serve_headers_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/assets/use_cases.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/auth/email_use_cases_test.go` | .go | 已全文阅读 | 密码重置服务提供者返回 nil，执行返回原邮件关闭哨兵，不启动后台发送。 | 待统一验收 |
| `backend/internal/wiki/auth/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/auth/errors_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/auth/routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/auth/use_cases.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/avatar/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/avatar/errors_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/avatar/routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/avatar/use_cases.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/avatar/use_cases_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/branding/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/branding/errors_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/branding/routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/branding/use_cases.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/default_documents.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/health/routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/health/use_cases.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/import_adapter.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/importer/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/importer/routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/importer/use_cases.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/instancesettings/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/instancesettings/routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/links/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/links/errors_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/links/routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/links/use_cases.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/metrics_runtime.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pages/add_favorite.go` | .go | 已全文阅读 | 先确保页面可读取，再写用户收藏，身份由上层提供，错误原样返回，不在用例内部认证。 | 待统一验收 |
| `backend/internal/wiki/pages/convert_page.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pages/copy_page.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pages/create_page.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pages/delete_page.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pages/ensure_path.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pages/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pages/errors_test.go` | .go | 已全文阅读 | 构造收藏存储不可用本地化错误，真实 Gin 响应入口断言 503，不暂停实际服务。 | 待统一验收 |
| `backend/internal/wiki/pages/get_page.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pages/get_page_test.go` | .go | 已全文阅读 | 空白或纯符号标题经真实路径建议用例返回空 slug，未加载页面树即可覆盖该提前返回分支。 | 待统一验收 |
| `backend/internal/wiki/pages/helpers.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pages/list_favorites.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pages/move_page.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pages/pin_page.go` | .go | 已全文阅读 | 页面 ID、预期版本与置顶值直接交给树服务，不走通用保存编排器，成功返回页面引用。 | 待统一验收 |
| `backend/internal/wiki/pages/read_routes.go` | .go | 已全文阅读 | 页面树可全量或指定 depth 转换，非整数深度按负一处理，单页读取裁剪 ID 后统一响应。 | 待统一验收 |
| `backend/internal/wiki/pages/refactor.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pages/remove_favorite.go` | .go | 已全文阅读 | 取消收藏用例直接调用收藏存储删除，不读取页面或检查上下文，关系缺失视为正常。 | 待统一验收 |
| `backend/internal/wiki/pages/routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pages/routes_flatten_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pages/update_page.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pages/write_routes.go` | .go | 已全文阅读 | 裁剪页面路由 ID 并读取认证用户，向取消收藏用例传入两个标识，错误统一响应，成功返回 JSON 消息。 | 待统一验收 |
| `backend/internal/wiki/pagesave/effects_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pagesave/event.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pagesave/link_effect.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pagesave/orchestrator.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pagesave/properties_effect.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pagesave/revision_effect.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pagesave/search_effect.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/pagesave/tags_effect.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/properties/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/properties/routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/properties/use_cases.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/restore/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/restore/routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/restore/routes_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/resync/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/resync/job.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/resync/job_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/resync/routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/resync/routes_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/resync/use_cases.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/resync/use_cases_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/revisions/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/revisions/routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/revisions/use_cases.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/search/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/search/errors_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/search/routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/search/use_cases.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/snapshot/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/snapshot/routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/snapshot/routes_test.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/tags/errors.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/tags/routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/tags/use_cases.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/wiki.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/wiki_boot.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/wiki_resync.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `backend/internal/wiki/wiki_routes.go` | .go | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/.gitignore` | .gitignore | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/.npmrc` | .npmrc | 已全文阅读 | 依赖安装禁止生命周期脚本，说明其与显式 npm 命令的区别。 | 待统一验收 |
| `frontend/.prettierrc` | .prettierrc | 已全文阅读 | 格式参数使用两空格、单引号和无分号，插件处理 Tailwind 类顺序。严格 JSON 不插入注释。 | 待统一验收 |
| `frontend/eslint.config.js` | .js | 已全文阅读 | 源码路径与安全语法检查，说明排除范围、匹配形式及工具豁免要求。 | 待统一验收 |
| `frontend/feature-interfaces.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/index.html` | .html | 已全文阅读 | 站点模板占位符、React 根节点及构建脚本入口，HTML 注释不改变节点属性。 | 待统一验收 |
| `frontend/package-lock.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/package.json` | .json | 已全文阅读 | 前端依赖与检查命令入口，build 同步词条后执行类型检查和 Vite；声明 Linux 运行约束。依赖声明不改写。 | 待统一验收 |
| `frontend/runtimeHtml.ts` | .ts | 已全文阅读 | 开发和预览 HTML 中间件，核实后端站点查询、占位符、路径过滤、503 与无缓存响应。 | 待统一验收 |
| `frontend/src/app/App.tsx` | .tsx | 已全文阅读 | 根组件协调配置、品牌、身份、收藏、设置同步与路由生命周期，现有说明逐段核对。 | 待统一验收 |
| `frontend/src/app/bootstrapAuth.test.ts` | .ts | 已全文阅读 | 七个案例核实请求选择、401 清理与网络错误保留，已逐例说明模拟条件和断言。 | 待统一验收 |
| `frontend/src/app/bootstrapAuth.ts` | .ts | 已全文阅读 | 按模式、缓存用户和访问到期时间选择身份查询或刷新，局部清理标志不取消请求。 | 待统一验收 |
| `frontend/src/app/components/DialogManager.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/app/components/HotKeyHandler.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/app/components/HotKeyHandler.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/app/dialogRegistrations.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/app/layout/AppLayout.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/app/lazy-dialogs.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/app/registrations.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/app/router/RouterAuthWrapper.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/app/router/RouterAuthWrapper.tsx` | .tsx | 已全文阅读 | 匿名只读公开分支与认证包装，错误边界按 pathname 重置。 | 待统一验收 |
| `frontend/src/app/router/lazy-routes.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/app/router/router.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/app/router/router.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/app/settingsSectionRegistry.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/app/settingsSectionRegistry.ts` | .ts | 已全文阅读 | 将共享设置元数据映射到应用懒组件，复制元数据并增加 Component，类型断言不检查缺失 ID，不在映射时加载组件。 | 待统一验收 |
| `frontend/src/app/settingsSync.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/app/settingsSync.ts` | .ts | 已全文阅读 | 长轮询游标、串行刷新合并和可见性刷新，核实停止标志与计时器清理。 | 待统一验收 |
| `frontend/src/app/useApplyDesignMode.tsx` | .tsx | 已全文阅读 | 布局副作用按主题切换根元素 dark 类，不在卸载时恢复旧值。 | 待统一验收 |
| `frontend/src/app/useTrackLastWikiLocation.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/app/useTrackLastWikiLocation.ts` | .ts | 已全文阅读 | 记录非设置页面的路径、搜索串和导航 state，不记录 hash。 | 待统一验收 |
| `frontend/src/features/apikeys/ApiKeyFormDialog.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/apikeys/ApiKeyFormDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/apikeys/ApiKeysManagement.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/apikeys/ApiKeysManagement.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/apikeys/CreateApiKeyButton.tsx` | .tsx | 已全文阅读 | 仅打开全局密钥表单且不传参数，输入与提交由对话框承担，按钮译文来自 apikeys。 | 待统一验收 |
| `frontend/src/features/apikeys/DeleteApiKeyButton.tsx` | .tsx | 已全文阅读 | 图标按钮以同一译文设置名称和提示，仅传选中密钥 ID 与名称打开撤销确认框。 | 待统一验收 |
| `frontend/src/features/apikeys/DeleteApiKeyDialog.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/apikeys/DeleteApiKeyDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/apikeys/state/apikeys.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/apikeys/state/apikeys.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/assets/AssetItem.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/assets/AssetManager.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/assets/AssetManagerDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/assets/AssetPreviewTooltip.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/auth/AcceptInvitePage.tsx` | .tsx | 已全文阅读 | 路由包装选择 invite 模式，核实共享表单及认证接口后确认成功建立会话并进入根路由。 | 待统一验收 |
| `frontend/src/features/auth/AuthRedirect.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/auth/ExternalRedirect.tsx` | .tsx | 已全文阅读 | 副作用调用统一外部跳转入口，本组件不绘制内容。 | 待统一验收 |
| `frontend/src/features/auth/ForcePasswordChange.tsx` | .tsx | 已全文阅读 | 复用账号设置面板显示强制改密页面，选择显示时机由应用根组件决定。 | 待统一验收 |
| `frontend/src/features/auth/ForgotPasswordForm.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/auth/ForgotPasswordForm.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/auth/LoginForm.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/auth/LoginForm.tsx` | .tsx | 已全文阅读 | 账号密码提交、错误提示和返回路由选择，说明浏览器必填与原样提交。 | 待统一验收 |
| `frontend/src/features/auth/RequireAuth.tsx` | .tsx | 已全文阅读 | 无用户时等待刷新或进入登录，有用户时渲染，后端请求仍独立鉴权。 | 待统一验收 |
| `frontend/src/features/auth/ResetPasswordPage.tsx` | .tsx | 已全文阅读 | 路由包装传入 reset 模式，实际输入校验、请求与确认页均由 SetPasswordForm 处理，不自动登录。 | 待统一验收 |
| `frontend/src/features/auth/SetPasswordForm.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/auth/SetPasswordForm.tsx` | .tsx | 已全文阅读 | 邀请与重置共用令牌表单，长度按 UTF-16 单元本地校验；邀请应用会话并替换导航，重置仅显示成功页，finally 解除加载。 | 待统一验收 |
| `frontend/src/features/auth/components/RoleGuard.tsx` | .tsx | 已全文阅读 | 使用会话角色与共享 hasRole 判定内容渲染，空角色拒绝，空允许列表接受已有角色，不发认证请求或替代服务器校验。 | 待统一验收 |
| `frontend/src/features/auth/components/UserMenu.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/auth/components/UserMenu.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/branding/BrandingAsset.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/branding/BrandingAsset.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/branding/BrandingSettings.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/branding/BrandingSettings.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/branding/assets.ts` | .ts | 已全文阅读 | 按资源类型与配置文件名选择品牌接口或内置资源，挂载前缀统一处理，版本查询参数参与缓存键，不验证文件存在。 | 待统一验收 |
| `frontend/src/features/branding/initialBranding.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/branding/initialBranding.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/branding/state/branding.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/designtoggle/DesignToggle.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/designtoggle/designmode.ts` | .ts | 已全文阅读 | 本地存储非空文本经类型断言作为模式，无运行时枚举校验，设置先写存储再更新状态，真实深浅色由解析钩子负责。 | 待统一验收 |
| `frontend/src/features/designtoggle/useResolvedDesignMode.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/designtoggle/useResolvedDesignMode.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/EditorTitleBar.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/LinkInsertDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/MarkdownCodeEditor.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/MarkdownCodeEditor.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/MarkdownEditor.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/MarkdownEditor.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/MarkdownToolbar.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/MarkdownToolbar.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/PageEditor.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/PageFrontmatterPanel.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/components/UnsavedChangesDialog.tsx` | .tsx | 已全文阅读 | 未保存内容确认框只调用确认或取消回调并让基础弹窗关闭，默认聚焦取消，不自行保存或导航。 | 待统一验收 |
| `frontend/src/features/editor/editorCommands.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/editorCommands.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/frontmatter.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/frontmatter.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/htmlToMarkdown.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/htmlToMarkdown.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/internalLinkCompletion.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/internalLinkCompletion.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/pageEditorStore.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/pageEditorStore.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/pasteImageUpload.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/pasteImageUpload.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/state/editor.ts` | .ts | 已全文阅读 | 编辑器预览布局和换行持久化，自动保存状态不序列化，反转动作读取当前状态，容器本身不执行保存。 | 待统一验收 |
| `frontend/src/features/editor/useAutoSave.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/useAutoSave.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/useEditorFileUpload.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/useEditorFileUpload.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/useExternalUnloadBlocker.ts` | .ts | 已全文阅读 | beforeunload 监听按标志请求浏览器离开确认，依赖变化与卸载撤销原监听，浏览器负责提示文字。 | 待统一验收 |
| `frontend/src/features/editor/useNavigationGuard.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/usePreviewScrollSync.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/usePreviewScrollSync.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/useSplitPaneResize.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/useSplitPaneResize.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/editor/useToolbarActions.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/favorites/FavoriteItem.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/favorites/FavoriteToggleButton.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/favorites/FavoritesSection.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/favorites/FavoritesSection.tsx` | .tsx | 已全文阅读 | 收藏 ID 映射树索引，丢弃无法解析项后按标题 localeCompare 排序，无可展示节点不渲染容器，也不清理收藏状态。 | 待统一验收 |
| `frontend/src/features/favorites/state/favorites.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/favorites/state/favorites.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/history/HistoryNotices.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/history/PageHistoryContent.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/history/RestoreRevisionDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/history/RevisionDetailPanels.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/history/lineDiff.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/history/lineDiff.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/history/pageHistory.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/history/restoreRevisionDialogState.ts` | .ts | 已全文阅读 | 将版本恢复确认回调封装为 Promise，布尔或 null 原样返回，没有请求、超时与主动拒绝。 | 待统一验收 |
| `frontend/src/features/imagepreview/ImagePreviewDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/importer/Importer.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/importer/Importer.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/importer/importerState.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/importer/importerState.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/importer/state/import.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/importer/useImporter.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/links/BrokenLinks.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/links/LinkInfo.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/links/linkStatusStore.ts` | .ts | 已全文阅读 | 保存单份最近链接结果，空 ID 本地报错；请求失败保留旧结果，clear 不取消请求，并发响应没有序号保护。 | 待统一验收 |
| `frontend/src/features/page-switcher/PageQuickSwitcherDialog.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page-switcher/PageQuickSwitcherDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page-switcher/PageQuickSwitcherTrigger.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page-switcher/pageQuickSwitcher.ts` | .ts | 已全文阅读 | 空白查询截取原列表前若干项，非空查询沿用共享页面搜索评分，类型仅为共享条目别名。 | 待统一验收 |
| `frontend/src/features/page/AddPageDialog.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/AddPageDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/CopyPageDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/CreatePageByPathDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/DeletePageDialog.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/DeletePageDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/EditPageMetadataDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/MovePageDialog.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/MovePageDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/PageHistoryPage.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/PageRefactorDialog.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/PageRefactorDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/PageSelect.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/PermalinkDialog.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/PermalinkDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/PermalinkRedirect.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/RootRedirect.tsx` | .tsx | 已全文阅读 | 等待页面树并选择首个顶层子节点，不递归寻找页面；替换当前历史项并保留导航状态。 | 待统一验收 |
| `frontend/src/features/page/SlugInputWithSuggestion.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/SlugInputWithSuggestion.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/SortPagesDialog.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/SortPagesDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/pageMutationRefresh.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/page/pageRefactorDialogState.ts` | .ts | 已全文阅读 | 功能关闭或无受影响页面且无警告时返回 false，否则打开确认弹窗并转交布尔或 null，实际变更由调用方执行。 | 待统一验收 |
| `frontend/src/features/preview/DefaultDocuments.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/Headline.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/Headline.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/MarkdownCodeBlock.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/MarkdownImage.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/MarkdownImage.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/MarkdownInlineCode.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/MarkdownInlineCode.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/MarkdownLink.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/MarkdownLink.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/MarkdownPdfEmbed.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/MarkdownPdfEmbed.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/MarkdownPreview.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/MarkdownPreview.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/MarkdownPreviewErrorBoundary.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/MermaidBlock.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/MermaidBlock.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/TocDropdownButton.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/TocDropdownButton.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/TocSidePanel.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/TocSidePanel.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/extractTocEntries.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/markdownPreviewCodeTheme.css` | .css | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/normalizeMarkdownBlocks.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/normalizeMarkdownBlocks.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/normalizeMarkdownListIndentation.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/previewHelpers.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/previewHelpers.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/previewSanitizeSchema.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/rehypeCodeFenceLineNumbers.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/rehypeCodeFenceLineNumbers.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/rehypeLineNumber.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/rehypeWhitelistStyles.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/remarkImageSize.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/remarkImageSize.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/splitHighlightedLines.test.ts` | .ts | 已全文阅读 | 三项纯字符串测试核验普通拆行、末尾换行与中间空行，不把测试范围扩大到元素克隆。 | 待统一验收 |
| `frontend/src/features/preview/splitHighlightedLines.ts` | .ts | 已全文阅读 | 拆分文本换行，含直接文本子节点的元素按行克隆，其他元素整体保留；只去除空末行，中间空行保留。 | 待统一验收 |
| `frontend/src/features/preview/state/tocPanel.test.ts` | .ts | 已全文阅读 | 每例先设展开值，核验显式赋值和连续反转，默认值断言实际观察准备阶段状态，不声称覆盖全新模块初始化。 | 待统一验收 |
| `frontend/src/features/preview/state/tocPanel.ts` | .ts | 已全文阅读 | 会话内目录折叠布尔值，默认展开，支持显式赋值与反转，不写全局设置或持久化。 | 待统一验收 |
| `frontend/src/features/preview/syntaxHighlightLanguages.ts` | .ts | 已全文阅读 | 合并 lowlight 常用语言与显式高亮语法，sh 指向 bash，console 和 shellsession 指向 shell，不在导出对象处执行渲染。 | 待统一验收 |
| `frontend/src/features/preview/tocTitleKey.test.ts` | .ts | 已全文阅读 | 分别核验标题优先、只有下载和两者皆空的标题键，不依赖真实 DOM。 | 待统一验收 |
| `frontend/src/features/preview/tocTitleKey.ts` | .ts | 已全文阅读 | 优先根据非零标题数返回页内目录键，没有标题且有附件时返回下载键，两者皆空仍有目录键。 | 待统一验收 |
| `frontend/src/features/preview/tocVisibility.test.ts` | .ts | 已全文阅读 | 八组输入覆盖零项和三种模式，auto 在三项与四项之间切换初始折叠状态。 | 待统一验收 |
| `frontend/src/features/preview/tocVisibility.ts` | .ts | 已全文阅读 | 空目录或 never 折叠，auto 三项以内折叠，其他模式展开，此纯函数不读取面板临时状态。 | 待统一验收 |
| `frontend/src/features/preview/useCodeCopy.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/useMermaidInjector.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/preview/useTocScrollSpy.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/progressbar/Progressbar.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/progressbar/progressbarStore.ts` | .ts | 已全文阅读 | 共享容器仅存加载布尔值，直接写入不做并行请求计数，实际进度由组件计算。 | 待统一验收 |
| `frontend/src/features/restore/state/restore.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/restore/state/restore.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/search/HighlightedSearchTitle.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/search/Search.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/search/SearchResultCard.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/PreferencesSettings.tsx` | .tsx | 已全文阅读 | 偏好设置页面只提供翻译标题与布局，共用 PreferencesPanel 执行实例级偏好保存。 | 待统一验收 |
| `frontend/src/features/settings/PublicAccessSettings.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/PublicAccessSettings.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/SettingsIndexRedirect.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/SettingsIndexRedirect.tsx` | .tsx | 已全文阅读 | 按登记顺序选择可见且非外链的首个设置分区，无目标回根路由，替换历史项。英文说明已按实际筛选条件重写。 | 待统一验收 |
| `frontend/src/features/settings/SettingsLayout.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/SettingsLayout.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/SettingsNav.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/SettingsNav.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/SettingsSectionGuard.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/SettingsSectionGuard.tsx` | .tsx | 已全文阅读 | 不可见分区与非空外链分区均回设置根页，其余原样展示子节点，不在此组件直接离站跳转。 | 待统一验收 |
| `frontend/src/features/settings/TocDisplaySettings.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/TocDisplaySettings.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/account/AccountSettings.tsx` | .tsx | 已全文阅读 | 组合头像和自身密码面板，并将本地化账号标题同步到浏览器，子面板分别负责提交。 | 待统一验收 |
| `frontend/src/features/settings/account/AvatarPanel.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/account/AvatarPanel.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/account/ChangeOwnPasswordPanel.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/account/ChangeOwnPasswordPanel.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/account/PreferencesPanel.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/account/PreferencesPanel.tsx` | .tsx | 已全文阅读 | 三个受控选择器分别订阅共享状态，选择即保存并映射失败提示，日期时间稳定值与译文键分离。 | 待统一验收 |
| `frontend/src/features/settings/account/state/avatar.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/account/state/avatar.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/settings/adminSections.ts` | .ts | 已全文阅读 | 集中导出八个管理设置组件供动态导入，导出不构成权限限制，账号和偏好不在此列表。 | 待统一验收 |
| `frontend/src/features/settings/state/resync.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/shortcuts/ShortcutsDialog.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/shortcuts/ShortcutsDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/sidebar/Sidebar.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/sidebar/Sidebar.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/sidebar/SidebarAccordionSection.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/sidebar/SidebarAccordionSection.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/sidebar/state/sidebar.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/sidebar/state/sidebarPanels.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/sidebar/state/sidebarPanels.ts` | .ts | 已全文阅读 | 本地持久化展开 ID 数组，合并时仅接受全字符串数组，空数组有效，不校验 ID 登记存在与否。 | 待统一验收 |
| `frontend/src/features/sidebar/useAutoCloseSidebarOnMobile.ts` | .ts | 已全文阅读 | 布局副作用只比较 pathname，桌面同步上次路径，移动路径变化且侧栏打开才关闭，查询串和锚点不触发该规则。 | 待统一验收 |
| `frontend/src/features/snapshot/SnapshotSettings.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/snapshot/scheduleOptions.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/snapshot/scheduleOptions.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/snapshot/state/snapshot.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/snapshot/state/snapshot.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/snapshot/useSnapshotSettings.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/tags/components/TagInputWithSuggestions.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/toolbar/Toolbar.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/toolbar/ToolbarButton.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/toolbar/toolbarStore.ts` | .ts | 已全文阅读 | 保存工具栏按钮数据与回调，整体替换列表并直接返回当前数组引用，不执行动作或注册快捷键。 | 待统一验收 |
| `frontend/src/features/toolbar/useExitModeButton.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/toolbar/useExitModeButton.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/tree/PinnedPageItem.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/tree/PinnedSection.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/tree/TreeDnd.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/tree/TreeNode.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/tree/TreeNode.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/tree/TreeNodeActionsMenu.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/tree/TreeNodeActionsMenu.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/tree/TreeView.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/tree/TreeView.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/tree/TreeViewActionButton.tsx` | .tsx | 已全文阅读 | 包装树行操作提示和名称，默认 button 类型，调用外部点击后停止冒泡；固定 className 等属性覆盖透传同名值。 | 待统一验收 |
| `frontend/src/features/tree/state/tree.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/tree/state/tree.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/tree/treeDndMove.test.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/tree/treeDndMove.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/tree/treeDndStore.ts` | .ts | 已全文阅读 | 只保存拖放启用、拖动节点与落点初值，未定义动作或持久化，外部调用状态接口写入。 | 待统一验收 |
| `frontend/src/features/tree/treeDndUtils.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/tree/treeNodeActionsMenus.ts` | .ts | 已全文阅读 | 共享单个打开菜单节点 ID，空值关闭全部菜单，由具体菜单按 ID 比较决定显示。 | 待统一验收 |
| `frontend/src/features/users/ChangePasswordButton.test.tsx` | .tsx | 已全文阅读 | 管理员会话渲染目标不同 ID 与相同 ID 两种情况，仅检查按钮存在性，不调用密码接口。 | 待统一验收 |
| `frontend/src/features/users/ChangePasswordButton.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/users/ChangePasswordDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/users/CreateEditUserButton.tsx` | .tsx | 已全文阅读 | 传入用户选择编辑，无用户选择创建；仅编辑且 menuItem 为真才使用菜单项，点击只打开共享表单。 | 待统一验收 |
| `frontend/src/features/users/DeleteUserButton.tsx` | .tsx | 已全文阅读 | 相同会话 ID 隐藏入口，其余根据菜单参数渲染，仅把 ID 与用户名交确认弹窗，权限仍由后端执行。 | 待统一验收 |
| `frontend/src/features/users/DeleteUserDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/users/ResendInviteButton.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/users/UserFormDialog.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/users/UserFormDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/users/UserManagement.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/users/UserRowActions.test.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/users/UserRowActions.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/users/state/users.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/viewer/Breadcrumbs.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/viewer/EmptySectionChildrenList.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/viewer/PageMetadata.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/viewer/PageViewer.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/viewer/sectionNavigation.test.ts` | .ts | 已全文阅读 | 内存节点区分页面和空章节，三例验证原顺序、深度优先以及完全没有后代页面。 | 待统一验收 |
| `frontend/src/features/viewer/sectionNavigation.ts` | .ts | 已全文阅读 | 按传入 children 顺序深度优先寻找首个后代页面，起始节点不参与类型判断，无页面返回 null。 | 待统一验收 |
| `frontend/src/features/viewer/setTitle.tsx` | .tsx | 已全文阅读 | 副作用按标题和站点名更新 document.title，空标题仅显示站点名，卸载不恢复旧值。 | 待统一验收 |
| `frontend/src/features/viewer/useScrollToHeadline.tsx` | .tsx | 已全文阅读 | 正文加载完成且非空并含片段才启动共享标题滚动，副作用取消函数在依赖变化或卸载时执行。 | 待统一验收 |
| `frontend/src/features/viewer/useScrollToSearchQuery.tsx` | .tsx | 已全文阅读 | 提取并裁剪 q 参数，正文就绪、没有锚点且至少三个 UTF-16 单元时调用搜索定位，副作用返回清理函数。 | 待统一验收 |
| `frontend/src/features/viewer/useSetPageTitle.tsx` | .tsx | 已全文阅读 | 监听页面标题和站点名生成浏览器标题，页面其他字段变化不触发标题副作用，空页面显示站点名。 | 待统一验收 |
| `frontend/src/features/viewer/useToolbarActions.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/viewer/viewer.ts` | .ts | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/features/wikilinks/WikiLinkDisambiguationDialog.tsx` | .tsx | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/index.css` | .css | 已全文阅读 | 已全文读取并核对调用关系、输入输出、分支和错误路径，对应源码注释按真实实现保持或补充。 | 待统一验收 |
| `frontend/src/jest-dom-vitest.d.ts` | .ts | 已全文阅读 | 按 Vitest 当前签名合并 DOM 断言与非对称匹配接口，运行时匹配器仍由设置入口加载，ESLint 指令保持机器文本。 | 待统一验收 |
| `frontend/src/main.tsx` | .tsx | 已全文阅读 | 载入样式和字体并以 StrictMode 挂载 React，root 非空断言不提供运行时检查。 | 待统一验收 |
| `frontend/src/shared/components/BaseDialog.tsx` | .tsx | 已全文阅读 | 全局类型控制打开，按钮确认使用动作名，快捷键确认固定 confirm。取消按钮使用 onClose 返回值，其他关闭路径不使用该返回值。 | 待统一验收 |
| `frontend/src/shared/components/ErrorBoundary.tsx` | .tsx | 已全文阅读 | 渲染异常进入错误状态并记录组件栈，重试清状态、重载刷新整页，resetKey 变化清除旧错误，界面可展开错误详情。 | 待统一验收 |
| `frontend/src/shared/components/FormInput.tsx` | .tsx | 已全文阅读 | 受控文本输入透传原始字符串和快捷键允许属性，显示可选标题与错误，不生成标签控件 ID 关联。 | 待统一验收 |
| `frontend/src/shared/components/ListView.tsx` | .tsx | 已全文阅读 | 提供列表外框、标题、条目与状态布局，选中只决定样式，条目使用 button 避免默认表单提交。 | 待统一验收 |
| `frontend/src/shared/components/Page404.tsx` | .tsx | 已全文阅读 | 目标允许创建时查询路径，结果须对应当前路径且账号可编辑才显示按钮，卸载后阻止迟到状态写入，不取消请求。 | 待统一验收 |
| `frontend/src/shared/components/Pagination.test.tsx` | .tsx | 已全文阅读 | 逐例说明单页隐藏、显示页码换算和首页按钮禁用，英文人工说明改写，业务断言文字保持原值。 | 待统一验收 |
| `frontend/src/shared/components/Pagination.tsx` | .tsx | 已全文阅读 | 总数与每页大小计算页数，内部索引从零开始，单页不显示，前后按钮报告有界索引不加载数据。 | 待统一验收 |
| `frontend/src/shared/components/ScrollableContainer.tsx` | .tsx | 已全文阅读 | 布局占满父级并纵向滚动，hidden 仅切换 CSS 显示，子组件不因该标志卸载。 | 待统一验收 |
| `frontend/src/shared/components/SettingsTable.test.tsx` | .tsx | 已全文阅读 | 通过真实按钮键盘操作和尺寸观察器替身验证两类单元格，说明属性测量、tabindex 切换与卸载释放。 | 待统一验收 |
| `frontend/src/shared/components/SettingsTable.tsx` | .tsx | 已全文阅读 | 统一列数与布局，文字单元格按溢出决定提示和焦点，尺寸与文字观察器在卸载时断开，控件单元格不测量文字。 | 待统一验收 |
| `frontend/src/shared/components/TooltipWrapper.tsx` | .tsx | 已全文阅读 | 窄屏不创建提示浮层，桌面渲染默认顶部起始对齐的提示，asChild 决定是否包裹额外 div。 | 待统一验收 |
| `frontend/src/shared/components/ui/accordion.tsx` | .tsx | 已全文阅读 | 逐项解释折叠根、末项边框、状态箭头与内容动画，className 同时作用于内容外层和间距内层。 | 待统一验收 |
| `frontend/src/shared/components/ui/alert-dialog.tsx` | .tsx | 已全文阅读 | 逐组件说明确认对话框遮罩、传送内容、响应式按钮布局和复用按钮样式，业务动作由调用方提供。 | 待统一验收 |
| `frontend/src/shared/components/ui/avatar.tsx` | .tsx | 已全文阅读 | 说明圆形容器、图片属性透传与固定内置回退头像，回退组件不渲染传入 children。 | 待统一验收 |
| `frontend/src/shared/components/ui/button.tsx` | .tsx | 已全文阅读 | 说明样式变体、尺寸、默认值与 Slot 模式，明确组件未指定 type，英文焦点说明改写。 | 待统一验收 |
| `frontend/src/shared/components/ui/checkbox.tsx` | .tsx | 已全文阅读 | 说明状态由原语处理、数据属性控制颜色，指示器固定使用勾号，没有混合状态专用图形。 | 待统一验收 |
| `frontend/src/shared/components/ui/dialog.tsx` | .tsx | 已全文阅读 | 说明传送层、自动遮罩、关闭按钮和标题描述包装，关闭文字在渲染时从语言资源读取。 | 待统一验收 |
| `frontend/src/shared/components/ui/dropdown-menu.tsx` | .tsx | 已全文阅读 | 分别说明主菜单与子菜单传送差异、普通项与单多选指示器、缩进、分隔线和纯文字快捷键提示。 | 待统一验收 |
| `frontend/src/shared/components/ui/input.tsx` | .tsx | 已全文阅读 | 英文说明改写，说明时间原生选择器伪元素、ref 与属性透传，以及组合外框为输入和装饰分配空间。 | 待统一验收 |
| `frontend/src/shared/components/ui/label.tsx` | .tsx | 已全文阅读 | 说明原生标签与 ref 透传，禁用相邻控件样式和调用方负责的 htmlFor 关联。 | 待统一验收 |
| `frontend/src/shared/components/ui/progress.tsx` | .tsx | 已全文阅读 | 说明 value 仅计算指示器平移，没有传给根原语，假值按零处理且没有百分比范围限制。 | 待统一验收 |
| `frontend/src/shared/components/ui/select.tsx` | .tsx | 已全文阅读 | 分别说明触发器、滚动按钮、Portal、视口尺寸、候选指示器和分隔，组件不维护候选数据或另一份选中值。 | 待统一验收 |
| `frontend/src/shared/components/ui/tabs.tsx` | .tsx | 已全文阅读 | 说明根原语、选项条、active 样式和内容属性透传，当前值和挂载行为交给原语。 | 待统一验收 |
| `frontend/src/shared/components/ui/tooltip.tsx` | .tsx | 已全文阅读 | 说明提示提供器与触发器原语，正文默认边距、Portal 和可用宽高约束，不维护额外计时器。 | 待统一验收 |
| `frontend/src/shared/contracts/auth.ts` | .ts | 已全文阅读 | 认证、用户和密钥共享角色类型及无密码会话用户形状，类型本身不承担运行时校验。 | 待统一验收 |
| `frontend/src/shared/contracts/page.ts` | .ts | 已全文阅读 | 集中定义页面与章节类别的传输字符串和类型，不在运行时验证响应。 | 待统一验收 |
| `frontend/src/shared/lib/api/apikeys.ts` | .ts | 已全文阅读 | 密钥接口描述密钥元数据并发送列表、创建和删除请求，明文秘密值只出现在创建响应的数据结构中。 | 待统一验收 |
| `frontend/src/shared/lib/api/assets.test.ts` | .ts | 已全文阅读 | 独立测试页标识规范化、并发去重、引用复用、失败后的再请求与上传清缓存，说明替身响应和断言的对应关系。 | 待统一验收 |
| `frontend/src/shared/lib/api/assets.ts` | .ts | 已全文阅读 | 附件列表按页面缓存并合并同页并发读取，上传和删除清除缓存。删除时编码文件名，缓存没有定时过期机制。 | 待统一验收 |
| `frontend/src/shared/lib/api/auth.test.ts` | .ts | 已全文阅读 | 逐例核对请求头、刷新与重试、已确认认证模式、错误结构与响应解析。英文人工说明已重写，测试输入与用例名称保持原值。 | 待统一验收 |
| `frontend/src/shared/lib/api/auth.ts` | .ts | 已全文阅读 | 公开认证请求及会话状态应用，核实结构化错误优先与 logout 的 finally 清理。 | 待统一验收 |
| `frontend/src/shared/lib/api/avatar.ts` | .ts | 已全文阅读 | 头像读取禁用浏览器缓存，上传使用 FormData。头像地址由接口基址、账号标识与可选版本参数组成。 | 待统一验收 |
| `frontend/src/shared/lib/api/branding.ts` | .ts | 已全文阅读 | 品牌图片上传使用表单数据，品牌配置通过接口读写，模块不维护独立的配置缓存。 | 待统一验收 |
| `frontend/src/shared/lib/api/client.ts` | .ts | 已全文阅读 | 携带 Cookie、CSRF、共享刷新 Promise 和错误解释，核实一次 401 重试与响应类型断言。 | 待统一验收 |
| `frontend/src/shared/lib/api/config.ts` | .ts | 已全文阅读 | 封装公共配置读取请求，返回服务端配置数据供配置状态模块应用。 | 待统一验收 |
| `frontend/src/shared/lib/api/errors.test.ts` | .ts | 已全文阅读 | 核对字段翻译、模板缺失保留原消息及网络异常映射，英文人工说明已改写，断言与业务字符串保持原值。 | 待统一验收 |
| `frontend/src/shared/lib/api/errors.ts` | .ts | 已全文阅读 | 识别结构化接口错误，使用错误模板与参数生成提示，字段错误和网络异常分别处理，模板缺失时保留接口消息。 | 待统一验收 |
| `frontend/src/shared/lib/api/favorites.ts` | .ts | 已全文阅读 | 收藏接口按服务端响应返回页面集合，不在客户端排序或重新计算收藏顺序。 | 待统一验收 |
| `frontend/src/shared/lib/api/import.ts` | .ts | 已全文阅读 | 导入接口封装上传、执行和取消，取消响应只按空值或 execution_status 字段结构识别，不执行完整对象校验。 | 待统一验收 |
| `frontend/src/shared/lib/api/instanceSettings.ts` | .ts | 已全文阅读 | 以接口请求读取和保存实例设置，类型描述配置字段，实际配置校验与保存由服务端执行。 | 待统一验收 |
| `frontend/src/shared/lib/api/links.ts` | .ts | 已全文阅读 | 链接接口在页面标识为空时直接抛错，标识直接放入路径，返回服务端计算的链接关系。 | 待统一验收 |
| `frontend/src/shared/lib/api/pages.ts` | .ts | 已全文阅读 | 页面接口封装树、内容、编辑和重命名请求。路径查询编码查询值，保存携带版本供服务端处理并发，重构由服务端执行。 | 待统一验收 |
| `frontend/src/shared/lib/api/restore.ts` | .ts | 已全文阅读 | 恢复接口携带恢复令牌读取状态，专用请求不发送 Cookie，触发响应作为任务受理信息返回。 | 待统一验收 |
| `frontend/src/shared/lib/api/resync.ts` | .ts | 已全文阅读 | 封装文档重新扫描请求与执行状态读取，客户端请求负责触发流程，不在浏览器读取服务器文件。 | 待统一验收 |
| `frontend/src/shared/lib/api/revisions.ts` | .ts | 已全文阅读 | 修订接口提供游标列表、正文、附件地址和恢复请求，附件路径按段编码，恢复通过 POST 触发。 | 待统一验收 |
| `frontend/src/shared/lib/api/runtime.ts` | .ts | 已全文阅读 | 集中提供接口调用使用的运行时配置、会话读写与认证模式依赖，避免接口模块直接引用应用状态。 | 待统一验收 |
| `frontend/src/shared/lib/api/search.ts` | .ts | 已全文阅读 | 搜索参数约束偏移和每页条数，标签使用重复参数。无查询和标签时返回空结果，检索由服务端执行。 | 待统一验收 |
| `frontend/src/shared/lib/api/snapshot.ts` | .ts | 已全文阅读 | 快照接口触发任务并读取列表，列表空值转换为空数组。下载地址基于接口基址，触发请求不等待备份任务完成。 | 待统一验收 |
| `frontend/src/shared/lib/api/tags.ts` | .ts | 已全文阅读 | 标签接口把已选标签编码为重复查询参数，返回标签统计或页面标签数据，不在客户端重算计数。 | 待统一验收 |
| `frontend/src/shared/lib/api/users.ts` | .ts | 已全文阅读 | 用户接口封装列表、创建、编辑和删除请求，参数按请求体传输，不在此模块写入认证状态。 | 待统一验收 |
| `frontend/src/shared/lib/assetSrc.test.ts` | .ts | 已全文阅读 | 逐例说明外部和内联地址、站内路径、版本保留与片段位置，测试不实际下载资源。 | 待统一验收 |
| `frontend/src/shared/lib/assetSrc.ts` | .ts | 已全文阅读 | 规范附件与默认资源部署路径，版本函数只对识别的原始附件或接口前缀追加时间戳，已有 v 参数保持原值。 | 待统一验收 |
| `frontend/src/shared/lib/brandDefaults.ts` | .ts | 已全文阅读 | 集中声明站点名和默认头像资源路径，调用方负责部署前缀与外观配置选择。 | 待统一验收 |
| `frontend/src/shared/lib/config.ts` | .ts | 已全文阅读 | 模块加载时读取 meta 基础路径，集中提供上传显示缺省值与媒体扩展名，字节格式按 1024 换算并使用界面语言。 | 待统一验收 |
| `frontend/src/shared/lib/dateLocale.ts` | .ts | 已全文阅读 | 界面短语言代码映射日期库与 Intl 区域标签，模块级监听语言变化并更新日期库默认选项。 | 待统一验收 |
| `frontend/src/shared/lib/deferState.ts` | .ts | 已全文阅读 | 通过 Promise 微任务调度状态回调，不返回 Promise，也没有取消入口。 | 待统一验收 |
| `frontend/src/shared/lib/dynamicImportError.test.ts` | .ts | 已全文阅读 | 逐例核对三种浏览器提示与无关内容错误，只验证文本分类。 | 待统一验收 |
| `frontend/src/shared/lib/dynamicImportError.ts` | .ts | 已全文阅读 | 按三种浏览器提示正则分类动态模块加载错误，不验证网络状态或执行恢复。 | 待统一验收 |
| `frontend/src/shared/lib/formatDate.test.ts` | .ts | 已全文阅读 | 本地时间构造后转 ISO 输入，核对默认格式、显式偏好、中文相对时间与三个区域标签。 | 待统一验收 |
| `frontend/src/shared/lib/formatDate.ts` | .ts | 已全文阅读 | 时间文本解析后按用户日期与时间偏好格式化，无效日期返回空串，相对时间读取日期库全局语言。 | 待统一验收 |
| `frontend/src/shared/lib/handleFieldErrors.ts` | .ts | 已全文阅读 | 使用统一接口错误映射设置字段提示并显示全局通知，无字段信息时不清空原字段状态。 | 待统一验收 |
| `frontend/src/shared/lib/hotkeys.ts` | .ts | 已全文阅读 | 统一声明与事件的修饰键和物理键名称，按属性允许列表判断输入区快捷键，不在此注册事件或阻止默认动作。 | 待统一验收 |
| `frontend/src/shared/lib/i18n.test.ts` | .ts | 已全文阅读 | 说明资源发现与递归叶子路径收集，核对语言自称、命名空间集合及键集合，不比较译文内容。 | 待统一验收 |
| `frontend/src/shared/lib/i18n.ts` | .ts | 已全文阅读 | 按预加载语言资源组织命名空间，简体中文为初始语言，英语为回退语言，语言清单按自称名称排序。 | 待统一验收 |
| `frontend/src/shared/lib/instanceDefaults.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/lib/katexDependency.test.ts` | .ts | 已全文阅读 | 读取锁文件核对全部数学渲染依赖版本，并从消费包解析实际副本，测试继承 trust 的危险链接与普通公式。 | 待统一验收 |
| `frontend/src/shared/lib/navigationVisit.ts` | .ts | 已全文阅读 | 给浅拷贝导航状态写入访问标识，读取时依次采用有效标识、路由 key 与 pathname，回退随机值不作为认证凭据。 | 待统一验收 |
| `frontend/src/shared/lib/pageSearch.test.ts` | .ts | 已全文阅读 | 说明查询内部空格对分词回退与标题直接命中的区别，核对有限结果、空查询和零上限。 | 待统一验收 |
| `frontend/src/shared/lib/pageSearch.ts` | .ts | 已全文阅读 | 深度遍历根子节点生成搜索项，以标题、路径与面包屑顺序评分并保留有限结果，空查询保持输入顺序。 | 待统一验收 |
| `frontend/src/shared/lib/preprocessWikilinks.test.ts` | .ts | 已全文阅读 | 逐例解释标题与路径优先级、空格和尖括号处理、代码保护、大小写缓存及 Bash 样本排除，英文与历史过程说明已重写。 | 待统一验收 |
| `frontend/src/shared/lib/preprocessWikilinks.ts` | .ts | 已全文阅读 | 英文说明按真实正则与分支重写，明确代码片段识别范围、调用内缓存、标题优先和路径提示。标题索引实际按对象枚举顺序收集而非遍历树。 | 待统一验收 |
| `frontend/src/shared/lib/redirectToExternal.test.ts` | .ts | 已全文阅读 | 用可写 location 替身核对返回参数覆盖、原查询保留和片段位置，每例恢复原属性。 | 待统一验收 |
| `frontend/src/shared/lib/redirectToExternal.ts` | .ts | 已全文阅读 | 写 location.href 执行外部导航，绝对地址用 URL 设置返回参数，解析失败按片段边界拼接，不校验主机许可。 | 待统一验收 |
| `frontend/src/shared/lib/registries/dialogRegistry.ts` | .ts | 已全文阅读 | 对话框按类型注册，重复类型抛错，查询返回对象引用，列表返回新数组。 | 待统一验收 |
| `frontend/src/shared/lib/registries/instances.ts` | .ts | 已全文阅读 | 创建注册器单例并集中维护面板与对话框身份，组件由应用装配层登记。 | 待统一验收 |
| `frontend/src/shared/lib/registries/panelItemRegistry.ts` | .ts | 已全文阅读 | 面板按 ID 去重并保留插入次序，启用条件和渲染函数由消费者执行。 | 待统一验收 |
| `frontend/src/shared/lib/registries/settingsSections.ts` | .ts | 已全文阅读 | 设置分区集中定义角色、配置启用条件与外部地址，导航按数组排列，订阅当前会话与实例配置。 | 待统一验收 |
| `frontend/src/shared/lib/roles.test.ts` | .ts | 已全文阅读 | 逐例解释角色包含、拒绝、空列表和缺少角色，明确测试不验证会话真实性。 | 待统一验收 |
| `frontend/src/shared/lib/roles.ts` | .ts | 已全文阅读 | 缺少角色拒绝，空允许列表接受非空角色，非空列表要求精确包含，不验证角色来源。 | 待统一验收 |
| `frontend/src/shared/lib/routePath.ts` | .ts | 已全文阅读 | 处理部署前缀、编辑历史查看路由和永久链接，分别说明前缀边界与路径编码规则，不查询页面存在性。 | 待统一验收 |
| `frontend/src/shared/lib/scrollToHeadline.test.ts` | .ts | 已全文阅读 | 真实 DOM 与历史函数替身验证标题命中、缺失告警和编码片段，关闭布局等待以同步断言。 | 待统一验收 |
| `frontend/src/shared/lib/scrollToHeadline.ts` | .ts | 已全文阅读 | 按高度采样等待后查找解码标题，命中时替换历史地址片段并滚动，取消只阻止回调实际滚动。 | 待统一验收 |
| `frontend/src/shared/lib/scrollToSearchQuery.test.ts` | .ts | 已全文阅读 | 构造两次关键词命中的正文，验证首个标记、最短长度、无命中、替换与清理后文本保留。 | 待统一验收 |
| `frontend/src/shared/lib/scrollToSearchQuery.ts` | .ts | 已全文阅读 | 逐文本节点查找首个不区分大小写的命中，使用 mark 包围并居中滚动，清理恢复文本，不跨节点拼接查询。 | 待统一验收 |
| `frontend/src/shared/lib/selectPlacement.ts` | .ts | 已全文阅读 | 固定下拉向下展开，打开时下方空间不足则滚动触发器至视口中部，不持续监听窗口。 | 待统一验收 |
| `frontend/src/shared/lib/selectorDependency.test.ts` | .ts | 已全文阅读 | 读取锁文件核对选择器依赖版本，实际解析五千类节点并核对输出文本和节点数。 | 待统一验收 |
| `frontend/src/shared/lib/shortcuts/shortcutCatalog.test.ts` | .ts | 已全文阅读 | 核对身份唯一、可见性完整列表与排序、平台文字和处理器装配，英文历史说明已改写。 | 待统一验收 |
| `frontend/src/shared/lib/shortcuts/shortcutCatalog.ts` | .ts | 已全文阅读 | 清单统一快捷键身份、组合和模式，说明显示标签、隐藏规则、对象共享与注册描述装配，不把元数据标记等同持久配置。 | 待统一验收 |
| `frontend/src/shared/lib/sleep.ts` | .ts | 已全文阅读 | 使用定时器返回等待指定毫秒数后完成的 Promise，调用者通过等待衔接重试过程。 | 待统一验收 |
| `frontend/src/shared/lib/theme-tokens.test.ts` | .ts | 已全文阅读 | 英文说明重写为实际文本比较行为，正则按声明出现顺序配对，不把文本不同等同视觉对比度证明。 | 待统一验收 |
| `frontend/src/shared/lib/useAppMode.ts` | .ts | 已全文阅读 | 按去除部署前缀后的路由文本返回工作模式，dialog 只存在类型中，设置使用字符串前缀匹配。 | 待统一验收 |
| `frontend/src/shared/lib/useDateTimeFormat.ts` | .ts | 已全文阅读 | 订阅语言和显示偏好，通过 useMemo 生成格式函数集合，不随时间流逝安排刷新计时器。 | 待统一验收 |
| `frontend/src/shared/lib/useDebounce.ts` | .ts | 已全文阅读 | 输入或延迟变化时取消上一计时器，只有最新等待到期才更新状态，卸载时清除计时器。 | 待统一验收 |
| `frontend/src/shared/lib/useIsMobile.ts` | .ts | 已全文阅读 | 使用最大 767 CSS 像素的媒体查询判断窄屏，注册一次变化监听并在卸载时移除。 | 待统一验收 |
| `frontend/src/shared/lib/useIsReadOnly.ts` | .ts | 已全文阅读 | 未登录和 viewer 在前端标记只读，其他非空用户可编辑，服务端仍负责实际权限。 | 待统一验收 |
| `frontend/src/shared/lib/useScrollRestoration.ts` | .ts | 已全文阅读 | 按访问标识在 sessionStorage 保存容器位置，有地址片段或至少三字符查询时跳过；动画帧不提供取消。 | 待统一验收 |
| `frontend/src/shared/lib/utils.ts` | .ts | 已全文阅读 | 先由 clsx 合并条件类名，再由 tailwind-merge 处理工具类冲突，无自定义冲突规则。 | 待统一验收 |
| `frontend/src/shared/lib/wikiPath.test.ts` | .ts | 已全文阅读 | 逐例解释页面规范路径、树查询键、工作模式转换、相对目录和删除跳转边界。 | 待统一验收 |
| `frontend/src/shared/lib/wikiPath.ts` | .ts | 已全文阅读 | 页面路径与路由前缀分别处理，覆盖查询键、相对链接、父路径和删除后导航目标，不读取实际页面。 | 待统一验收 |
| `frontend/src/shared/lib/wikilinkTitleIndex.test.ts` | .ts | 已全文阅读 | 验证小写标题、同名候选顺序、特殊 Map 键和重新构造索引的独立性。 | 待统一验收 |
| `frontend/src/shared/locales/de/apikeys.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/assets.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/auth.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/branding.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/brokenLinks.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/common.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/editor.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/errors.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/history.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/importer.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/page.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/publicAccess.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/restore.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/search.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/settings.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/snapshot.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/tocDisplay.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/users.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/de/viewer.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/apikeys.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/assets.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/auth.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/branding.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/brokenLinks.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/common.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/editor.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/errors.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/history.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/importer.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/page.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/publicAccess.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/restore.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/search.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/settings.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/snapshot.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/tocDisplay.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/users.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/en/viewer.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/apikeys.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/assets.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/auth.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/branding.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/brokenLinks.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/common.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/editor.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/errors.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/history.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/importer.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/page.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/publicAccess.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/restore.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/search.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/settings.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/snapshot.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/tocDisplay.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/users.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/es/viewer.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/apikeys.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/assets.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/auth.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/branding.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/brokenLinks.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/common.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/editor.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/errors.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/history.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/importer.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/page.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/publicAccess.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/restore.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/search.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/settings.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/snapshot.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/tocDisplay.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/users.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/fr/viewer.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/apikeys.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/assets.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/auth.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/branding.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/brokenLinks.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/common.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/editor.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/errors.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/history.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/importer.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/page.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/publicAccess.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/restore.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/search.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/settings.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/snapshot.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/tocDisplay.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/users.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/locales/zh/viewer.json` | .json | 已全文阅读 | 结构化配置或依赖清单，已全文读取并核对读取位置。格式不支持安全插入注释，原始内容保持不变。 | 待统一验收 |
| `frontend/src/shared/state/config.test.ts` | .ts | 已全文阅读 | 配置测试使用虚拟计时器和隔离状态，逐例说明重试、版本与请求顺序、配置应用和认证模式行为。 | 待统一验收 |
| `frontend/src/shared/state/config.ts` | .ts | 已全文阅读 | 配置加载最多尝试三次，延迟为 500 与 1000 毫秒。请求序号与实例版本阻止过期响应覆盖，配置加载完成标记不等同加载成功。 | 待统一验收 |
| `frontend/src/shared/state/dialogs.ts` | .ts | 已全文阅读 | 对话框状态只保存一个类型与属性集合，打开时替换当前内容，关闭时清除类型，通过类型是否为空判断打开状态。 | 待统一验收 |
| `frontend/src/shared/state/hotkeys.test.ts` | .ts | 已全文阅读 | 核对相同键名下处理器的注册和注销，说明对象身份、顺序与末项删除的区别。 | 待统一验收 |
| `frontend/src/shared/state/hotkeys.ts` | .ts | 已全文阅读 | 快捷键以标准化键名维护处理器顺序，相同对象的注销移除其全部登记，只有该键没有处理器时才移除键项。 | 待统一验收 |
| `frontend/src/shared/state/lastWikiLocation.test.ts` | .ts | 已全文阅读 | 逐例核对缺省值、保存值与替换位置，说明每例前的内存状态重置和对象引用结果。 | 待统一验收 |
| `frontend/src/shared/state/lastWikiLocation.ts` | .ts | 已全文阅读 | 最近文档位置只保存在内存，读取返回保存的对象引用，不写入浏览器持久存储。 | 待统一验收 |
| `frontend/src/shared/state/session.ts` | .ts | 已全文阅读 | Zustand 持久化用户和到期秒数，刷新标志仅保存在内存，令牌正文由 Cookie 管理。 | 待统一验收 |
| `frontend/src/shared/state/userSettings.test.ts` | .ts | 已全文阅读 | 使用接口和语言模块替身验证偏好保存、排队与失败恢复，说明服务响应如何成为最终状态。 | 待统一验收 |
| `frontend/src/shared/state/userSettings.ts` | .ts | 已全文阅读 | 偏好写入串行排队，前一请求失败不阻断后续任务，服务端返回值写入共享状态，语言变化调用国际化模块。 | 待统一验收 |
| `frontend/src/shared/testutil/expectedDiagnostics.ts` | .ts | 已全文阅读 | 诊断替身按次数与参数逐条核对，可把预期输出写入环境指定路径，结束时通过 finally 恢复原输出函数。英文人工说明已改写。 | 待统一验收 |
| `frontend/src/test-setup.ts` | .ts | 已全文阅读 | 预载 DOM 断言，只有缺失 localStorage 时安装四方法 Map 替代，每例前将共享语言设为 en，特定语言测试自行切换。 | 待统一验收 |
| `frontend/src/vite-env.d.ts` | .ts | 已全文阅读 | Vite 客户端三斜线指令保持原语法；声明字体副作用模块及 GFM 插件五个函数形状，不执行运行时注册。 | 待统一验收 |
| `frontend/tsconfig.app.json` | .json | 已全文阅读 | 应用与测试的类型检查配置，JSONC 支持原生注释，说明模块解析与别名。 | 待统一验收 |
| `frontend/tsconfig.json` | .json | 已全文阅读 | 根类型检查项目引用应用和 Node 配置，自身 files 为空。 | 待统一验收 |
| `frontend/tsconfig.node.json` | .json | 已全文阅读 | Vite 配置检查项目，说明依赖缓存位置、无输出与严格检查。 | 待统一验收 |
| `frontend/vite.config.test.ts` | .ts | 已全文阅读 | 比较分块名生成结果，下划线与点前缀剔除，普通名保持；此测试不执行 Go 嵌入或实际浏览器加载。 | 待统一验收 |
| `frontend/vite.config.ts` | .ts | 已全文阅读 | 共享端口、版本、Mermaid 资源复制及构建分块，核实开发和生产模板的差异。 | 待统一验收 |
| `frontend/vitest.config.ts` | .ts | 已全文阅读 | React 与 jsdom 测试配置，公共设置与全局函数启用，V8 覆盖率过滤单独配置，别名按配置目录定位。 | 待统一验收 |
| `other/.golangci-lint-version` | .golangci-lint-version | 已全文阅读 | 固定检查器版本 v2.15.0，由 container.mjs 读取构建 lint 命名上下文，不添加行内注释。 | 待统一验收 |
| `other/.golangci.yml` | .yml | 已全文阅读 | 检查器第二版配置格式，声明静态检查、格式工具和排除路径。 | 待统一验收 |
| `other/.nvmrc` | .nvmrc | 已全文阅读 | Node 主版本 26 的版本声明，由工具链核对脚本读取，不支持行内说明。 | 待统一验收 |
| `other/Dockerfile` | 无扩展名 | 已全文阅读 | 前端、后端与运行三阶段镜像，核实依赖缓存、嵌入文件与链接参数。 | 待统一验收 |
| `other/Dockerfile.builder` | .builder | 已全文阅读 | 按架构和 CGO 参数输出 Linux 二进制的构建阶段，不提供运行入口。 | 待统一验收 |
| `other/Dockerfile.lint` | .lint | 已全文阅读 | 从命名上下文复制固定版本检查器，继承工具镜像中的源码与依赖。 | 待统一验收 |
| `other/Dockerfile.tools` | .tools | 已全文阅读 | 构建 Linux 工具与源码快照，依赖层分离于源码层。 | 待统一验收 |
| `other/Makefile` | 无扩展名 | 已全文阅读 | 容器任务统一入口和 Windows 启动目标，cross 实际运行 vet，release 实际构建本地镜像。 | 待统一验收 |
| `other/docker-entrypoint.sh` | .sh | 已全文阅读 | 扫描 host 参数并填充默认环境，日志没有时间戳，exec 直接运行主程序。 | 待统一验收 |
| `other/scripts/check-architecture.mjs` | .mjs | 已全文阅读 | 解析顶层静态导入导出，核实边界判断先于类型过滤及运行依赖循环检测。 | 待统一验收 |
| `other/scripts/check-commit-messages.sh` | .sh | 已全文阅读 | 提交标题策略检查，作为受保护策略保持原样，包括其 Usage 原文。 | 待统一验收 |
| `other/scripts/check-database-files.mjs` | .mjs | 已全文阅读 | 递归检查禁止数据库格式和 SQLite 头部，仅允许固定默认 SQL。 | 待统一验收 |
| `other/scripts/check-private-references.sh` | .sh | 已全文阅读 | 私有标记与主目录路径策略检查，保持策略文件原样。 | 待统一验收 |
| `other/scripts/check-project-structure.mjs` | .mjs | 已全文阅读 | 检查根业务目录集合和必要入口是否存在，不创建或移动目录。 | 待统一验收 |
| `other/scripts/check-toolchain-versions.sh` | .sh | 已全文阅读 | 读取 go.mod、.nvmrc 与 Dockerfile，检查 Go 完整版本和 Node 主版本及 mise 配置来源的一致性。注释已改为简体中文，保留 shellcheck 标记与错误输出协议。 | 待统一验收 |
| `other/scripts/check-toolchain-versions.test.sh` | .sh | 已全文阅读 | 在临时仓库中构造版本、容器配置和缺失文件样本，逐项核对检查脚本的成功与失败退出结果。英文人工说明已改为中文，测试输入字符串保持原样。 | 待统一验收 |
| `other/scripts/container.mjs` | .mjs | 已全文阅读 | 根据动作表在 Linux 容器中执行构建、测试、静态检查和前端验证，管理容器生命周期与命名卷。 | 待统一验收 |
| `other/scripts/dev.sh` | .sh | 已全文阅读 | 开发入口转交 PowerShell 持久启动脚本并保持参数边界。 | 待统一验收 |
| `other/scripts/doctor.sh` | .sh | 已全文阅读 | 检查宿主 Node、Docker 和 Linux 模式，不检查业务登录。 | 待统一验收 |
| `other/scripts/lib/project-root.sh` | .sh | 已全文阅读 | shell 根据脚本实际位置调用同一 Node 定位器。 | 待统一验收 |
| `other/scripts/lib/toolchain-versions.sh` | .sh | 已全文阅读 | 提供从 go.mod 和 .nvmrc 提取 Go 完整版本及 Node 主版本的 shell 函数，供版本检查与诊断脚本复用。 | 待统一验收 |
| `other/scripts/project-root.mjs` | .mjs | 已全文阅读 | 按前端清单和自身定位脚本向父级查找根目录，根目录未命中时终止。 | 待统一验收 |
| `other/scripts/project-root.ps1` | .ps1 | 已全文阅读 | PowerShell 调用共享 Node 根目录解析器，失败或空输出抛错。 | 待统一验收 |
| `other/scripts/release-service-ports.d.mts` | .mts | 已全文阅读 | 为端口释放同步入口声明服务名数组和无返回值函数形状。 | 待统一验收 |
| `other/scripts/release-service-ports.mjs` | .mjs | 已全文阅读 | 按宿主系统转交 PowerShell 或 Go 端口释放命令，导入时不执行，直接运行时解析命令行服务名。 | 待统一验收 |
| `other/scripts/release-service-ports.ps1` | .ps1 | 已全文阅读 | 从共享端口表解析服务，停止占用 Windows 监听端点的进程并复核端口已释放。 | 待统一验收 |
| `other/scripts/resolve-version.mjs` | .mjs | 已全文阅读 | 环境 APP_VERSION 优先，否则读取 Git 最近标签，查询失败用初始版本。 | 待统一验收 |
| `other/scripts/resolve-version.sh` | .sh | 已全文阅读 | shell 使用 exec 转交共享 Node 版本入口。 | 待统一验收 |
| `other/scripts/server.sh` | .sh | 已全文阅读 | Ubuntu systemd 安装与服务控制，模板属于生成内容；尚需核实模板中的英文说明处理。 | 待统一验收 |
| `other/scripts/start.ps1` | .ps1 | 已全文阅读 | 检查 Docker、Caddy 和证书，按共享端口表启动后台 Linux 容器及 HTTPS 网关，执行就绪探测后保存运行状态。 | 待统一验收 |
| `other/scripts/sync-error-catalogs.mjs` | .mjs | 已全文阅读 | 把各语言 errors.json 复制到后端，仅统一 CRLF，不解析或翻译内容。 | 待统一验收 |
| `other/scripts/verify-global-settings.py` | .py | 已全文阅读 | 启动一次性真实 LeafDoc 进程，覆盖默认配置、登录改密、权限、事件、文档、附件、搜索、快照恢复与重启持久化验证。 | 待统一验收 |

| `other/comment-review.md` | .md | 持续维护 | 逐文件阅读与核验记录，不把待阅读项视为已完成。 | 待统一验收 |
