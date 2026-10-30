# LeafDoc 配置来源与默认行为

本文件按命令行声明整理全部启动选项。实际程序声明是参数类型和校验的权威来源，`go run ./cmd/leafdoc --help` 可核查完整说明。实例可修改设置由 MySQL settings 记录保存，接口和页面读取同一份值。

## 来源顺序

命令行显式参数优先于环境变量。Windows 启动脚本仅用 UTF-8 `.env` 补齐未设置的进程环境变量。未显式提供的参数使用声明默认值或运行时解析结果。保存后的实例设置持续生效，公开访问的显式环境覆盖会在页面标记为环境管理。语言设置以已经保存的实例偏好为准，启动默认语言只用于初始回退。

## 启动参数清单

| 参数 | 环境变量 | 默认值或解析方式 | 行为 |
| --- | --- | --- | --- |
| `jwt-secret` | `LEAFDOC_JWT_SECRET` | `未配置` | Windows 启动脚本自动生成并保存在实例运行目录；直接运行程序须提供真实密钥 |
| `public-access` | `LEAFDOC_PUBLIC_ACCESS` | `true` | 显式值覆盖默认值 |
| `access-token-timeout` | `LEAFDOC_ACCESS_TOKEN_TIMEOUT` | `15m` | 显式值覆盖默认值 |
| `refresh-token-timeout` | `LEAFDOC_REFRESH_TOKEN_TIMEOUT` | `168h` | 显式值覆盖默认值 |
| `admin-username` | `LEAFDOC_ADMIN_USERNAME` | `Nian` | 显式值覆盖默认值 |
| `admin-email` | `LEAFDOC_ADMIN_EMAIL` | `Nian@LeafDoc.doc` | 显式值覆盖默认值 |
| `admin-password` | `LEAFDOC_ADMIN_PASSWORD` | `Nian123456，保存为 bcrypt 散列` | 显式值覆盖默认值 |
| `editor-limit` | `LEAFDOC_EDITOR_LIMIT` | `0` | 显式值覆盖默认值 |
| `snapshot` | `LEAFDOC_SNAPSHOT` | `true` | 显式值覆盖默认值 |
| `snapshot-interval` | `LEAFDOC_SNAPSHOT_INTERVAL` | `24h` | 显式值覆盖默认值 |
| `snapshot-retention` | `LEAFDOC_SNAPSHOT_RETENTION` | `10` | 显式值覆盖默认值 |
| `snapshot-dir` | `LEAFDOC_SNAPSHOT_DIR` | `<data-dir>/snapshots` | 显式值覆盖默认值 |
| `restore-upload-max-size` | `LEAFDOC_RESTORE_UPLOAD_MAX_SIZE` | `"500MiB"` | 显式值覆盖默认值 |
| `smtp-host` | `LEAFDOC_SMTP_HOST` | `未配置` | 可选功能不启用；路径和部署地址由运行上下文解析，不填入虚构值 |
| `smtp-port` | `LEAFDOC_SMTP_PORT` | `587` | 显式值覆盖默认值 |
| `smtp-username` | `LEAFDOC_SMTP_USERNAME` | `未配置` | 可选功能不启用；路径和部署地址由运行上下文解析，不填入虚构值 |
| `smtp-password` | `LEAFDOC_SMTP_PASSWORD` | `未配置` | 可选功能不启用；路径和部署地址由运行上下文解析，不填入虚构值 |
| `smtp-from` | `LEAFDOC_SMTP_FROM` | `未配置` | 可选功能不启用；路径和部署地址由运行上下文解析，不填入虚构值 |
| `smtp-from-name` | `LEAFDOC_SMTP_FROM_NAME` | `"LeafDoc"` | 显式值覆盖默认值 |
| `smtp-security` | `LEAFDOC_SMTP_SECURITY` | `"starttls"` | 显式值覆盖默认值 |
| `smtp-insecure-skip-verify` | `LEAFDOC_SMTP_INSECURE_SKIP_VERIFY` | `false` | 显式值覆盖默认值 |
| `smtp-timeout` | `LEAFDOC_SMTP_TIMEOUT` | `10s` | 显式值覆盖默认值 |
| `public-url` | `LEAFDOC_PUBLIC_URL` | `未配置` | 可选功能不启用；路径和部署地址由运行上下文解析，不填入虚构值 |
| `default-language` | `LEAFDOC_DEFAULT_LANGUAGE` | `zh` | 显式值覆盖默认值 |
| `inject-code-in-header` | `LEAFDOC_INJECT_CODE_IN_HEADER` | `未配置` | 可选功能不启用；路径和部署地址由运行上下文解析，不填入虚构值 |
| `custom-stylesheet` | `LEAFDOC_CUSTOM_STYLESHEET` | `未配置` | 可选功能不启用；路径和部署地址由运行上下文解析，不填入虚构值 |
| `hide-link-metadata-section` | `LEAFDOC_HIDE_LINK_METADATA_SECTION` | `false` | 显式值覆盖默认值 |
| `max-asset-upload-size` | `LEAFDOC_MAX_ASSET_UPLOAD_SIZE` | `"50MiB"` | 显式值覆盖默认值 |
| `enable-revision` | `LEAFDOC_ENABLE_REVISION` | `false` | 显式值覆盖默认值 |
| `enable-link-refactor` | `LEAFDOC_ENABLE_LINK_REFACTOR` | `false` | 显式值覆盖默认值 |
| `enable-api-key-management` | `LEAFDOC_ENABLE_API_KEY_MANAGEMENT` | `false` | 显式值覆盖默认值 |
| `max-revision-history` | `LEAFDOC_MAX_REVISION_HISTORY` | `100` | 显式值覆盖默认值 |
| `revision-coalesce-window` | `LEAFDOC_REVISION_COALESCE_WINDOW` | `5m` | 显式值覆盖默认值 |
| `enable-metrics` | `LEAFDOC_ENABLE_METRICS` | `false` | 显式值覆盖默认值 |
| `metrics-host` | `LEAFDOC_METRICS_HOST` | `"127.0.0.1"` | 显式值覆盖默认值 |
| `metrics-port` | `LEAFDOC_METRICS_PORT` | `serviceports.Port("metrics")` | 显式值覆盖默认值 |
| `enable-http-remote-user` | `LEAFDOC_ENABLE_HTTP_REMOTE_USER` | `false` | 显式值覆盖默认值 |
| `http-remote-user-header-name` | `LEAFDOC_HTTP_REMOTE_USER_HEADER_NAME` | `"Remote-User"` | 显式值覆盖默认值 |
| `enable-http-remote-user-auto-create` | `LEAFDOC_ENABLE_HTTP_REMOTE_USER_AUTO_CREATE` | `false` | 显式值覆盖默认值 |
| `http-remote-user-email-header-name` | `LEAFDOC_HTTP_REMOTE_USER_EMAIL_HEADER_NAME` | `未配置` | 可选功能不启用；路径和部署地址由运行上下文解析，不填入虚构值 |
| `http-remote-user-default-role` | `LEAFDOC_HTTP_REMOTE_USER_DEFAULT_ROLE` | `"viewer"` | 显式值覆盖默认值 |
| `trusted-proxy-ips` | `LEAFDOC_TRUSTED_PROXY_IPS` | `未配置` | 可选功能不启用；路径和部署地址由运行上下文解析，不填入虚构值 |
| `http-remote-user-logout-url` | `LEAFDOC_HTTP_REMOTE_USER_LOGOUT_URL` | `未配置` | 可选功能不启用；路径和部署地址由运行上下文解析，不填入虚构值 |
| `login-url` | `LEAFDOC_LOGIN_URL` | `未配置` | 可选功能不启用；路径和部署地址由运行上下文解析，不填入虚构值 |
| `logout-url` | `LEAFDOC_LOGOUT_URL` | `未配置` | 可选功能不启用；路径和部署地址由运行上下文解析，不填入虚构值 |
| `user-management-url` | `LEAFDOC_USER_MANAGEMENT_URL` | `未配置` | 可选功能不启用；路径和部署地址由运行上下文解析，不填入虚构值 |
| `host` | `LEAFDOC_HOST` | `"127.0.0.1"` | 显式值覆盖默认值 |
| `port` | `LEAFDOC_PORT` | `serviceports.Port("backend")` | 显式值覆盖默认值 |
| `unix-socket` | `LEAFDOC_UNIX_SOCKET` | `未配置` | 可选功能不启用；路径和部署地址由运行上下文解析，不填入虚构值 |
| `data-dir` | `LEAFDOC_DATA_DIR` | `"./data"` | 显式值覆盖默认值 |
| `base-path` | `LEAFDOC_BASE_PATH` | `未配置` | 可选功能不启用；路径和部署地址由运行上下文解析，不填入虚构值 |
| `allow-insecure` | `LEAFDOC_ALLOW_INSECURE` | `false` | 显式值覆盖默认值 |
| `log-format` | `LEAFDOC_LOG_FORMAT` | `"text"` | 显式值覆盖默认值 |
| `disable-request-log` | `LEAFDOC_DISABLE_REQUEST_LOG` | `false` | 显式值覆盖默认值 |

## 数据库与部署

MySQL 使用 `LEAFDOC_MYSQL_HOST`、`LEAFDOC_MYSQL_PORT`、`LEAFDOC_MYSQL_DATABASE`、`LEAFDOC_MYSQL_USER`、`LEAFDOC_MYSQL_PASSWORD`。端口缺省为 3306，主机、数据库和用户必须为真实配置，密码按数据库实际认证配置提供。连接使用 utf8mb4、UTC、10 秒连接超时及 30 秒读写超时。不会回退到其他存储引擎。

Windows 启动脚本从 `LEAFDOC_DATA_DIR` 或项目 data 目录解析实例目录，从 `LEAFDOC_TLS_CERT`、`LEAFDOC_TLS_KEY` 或系统公共数据目录 LeafDoc/tls 解析证书。`LEAFDOC_BASE_PATH` 默认为根路径。服务端口统一读取 serviceports 配置，不复制端口定义。JWT 密钥不存在时生成，不作为固定默认字符串。

## 实例设置

语言 `zh`，日期 `iso`，时间 `24h`。公开访问启用且只允许匿名读取。目录显示 `never`，界面仍保留用户对当前文档的临时展开操作。站点名称 LeafDoc，未上传品牌图片和头像时使用内嵌资源，因此资源路径空值表示使用内置资源，不表示缺失默认值。

自动计划启用，类型 `daily`，时间 `00:00`，时区 `Asia/Shanghai`，启动时备份开启，保留最近 10 份。备用间隔为 24 小时。显式的间隔 0 保留手动备份模式。用户保存的 false、0 和有效配置不会在重启时被默认值覆盖。

设置记录缺失时写入默认值，缺少字段时合并对应默认字段并持久化，JSON null、错误类型和无法解析的内容会明确报错。用户名、密码、邮箱、语言和日期格式等按现有服务规则校验。邮件、代理认证、自定义注入和自定义样式未配置时不启用，不填入无法连接的服务地址。

## 结构与默认值核查

唯一 SQL 默认文件由 `go run ./cmd/schema-defaults` 从实际 Go 建表定义和服务默认值生成，同时生成前端默认值 JSON。`go run ./cmd/schema-defaults -check` 检查内容漂移。SQL 与应用建表结果通过隔离 MySQL 数据库对照验证，不包含生产连接密码和 TLS 私钥。
