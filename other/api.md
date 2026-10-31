# LeafDoc HTTP 接口参考

本文用于直接调用当前服务。文中接口路径均为实例基础路径之后的完整路径，不需要再额外添加 `/api`。基础路径为空时入口为 `https://localhost:24691/api`；设置基础路径后，在 `/api` 前加入该路径。静态资源使用各自入口。

## 登录与请求约定

登录提交 JSON 字段 `identifier` 和 `password`，成功返回 `message`、`user` 和 `accessTokenExpiresAt`。访问与刷新令牌通过 Cookie 设置，登录 JSON 不返回访问令牌。`GET /api/auth/me` 无身份时返回 `200` 和 `null`。刷新请求使用刷新 Cookie，缺少或无效时返回 `422`。

登录、刷新及无需预先登录的邮件入口不使用业务鉴权组的 CSRF 中间件。登录示例无需提交 CSRF 请求头。已经登录后的业务写请求需要 CSRF 校验。先读取 `/api/config` 获取 CSRF Cookie，再将相同值放入 `X-CSRF-Token`。表单也接受 `csrf_token`。已被服务器成功解析的 API 密钥身份免除这项校验，单独设置 Bearer 请求头不能绕过身份检查。GET、HEAD、OPTIONS 不执行 CSRF 校验。

认证默认要求 HTTPS。使用受信任的 CA，通过 `curl --cacert` 指定证书来源。不要把忽略证书验证写成正常调用步骤。账号写入、API 密钥管理和部分快照恢复操作拒绝 API 密钥身份。

```bash
read -rp '实例 HTTPS 地址: ' LEAFDOC_URL
read -rp 'CA 文件路径: ' LEAFDOC_CA
curl --fail --cacert "$LEAFDOC_CA" -c cookies.txt "$LEAFDOC_URL/api/config"
# login.json 必须包含实际 identifier 和 password，文件仅允许当前用户读取。
curl --fail --cacert "$LEAFDOC_CA" -b cookies.txt -c cookies.txt \
  -H 'Content-Type: application/json' --data-binary @login.json "$LEAFDOC_URL/api/auth/login"
curl --fail --cacert "$LEAFDOC_CA" -b cookies.txt "$LEAFDOC_URL/api/auth/me"
```

登录后调用业务写接口时，可动态提取当前 CSRF Cookie，再通过请求头提交。

```bash
LEAFDOC_CSRF=$(awk '$6 == "__Host-leafdoc_csrf" || $6 == "leafdoc_csrf" { value=$7 } END { print value }' cookies.txt)
test -n "$LEAFDOC_CSRF"
curl --fail --cacert "$LEAFDOC_CA" -b cookies.txt \
  -H "X-CSRF-Token: $LEAFDOC_CSRF" -X POST "$LEAFDOC_URL/api/auth/logout"
```

上述示例会退出登录。登录文件和 Cookie 文件应放在私有目录，用完后删除。写请求的 CSRF 值来自当前 Cookie，不能使用固定示例值。登录每地址五分钟十次，成功清空该地址的登录历史。刷新每分钟三十次，成功不清空计数。

## 页面编辑与资源传输

创建页面提交 `parentId`、`title`、`slug`、`kind`，成功返回 `201` 和页面对象。更新时提交读取结果中的 `version`、`title`、`slug`，正文使用 `content`，标签使用 `tags`，属性使用 `properties`。省略正文与提交空字符串不同，空字符串表示清空正文。版本冲突后重新读取并合并，不能反复用旧版本覆盖。

删除页面从查询参数读取 `version` 和 `recursive`。上传附件使用 multipart 的 `file` 字段，整个请求体受到大小限制，包括表单边界。成功返回 `201` 与资源地址。资源列表返回页面关联的资源，下载时使用实际返回的地址。

导入先上传 ZIP 到 `/api/import/plan`，表单可带 `targetBasePath`，核对计划后调用 `/api/import/execute`。返回 `202` 表示任务启动或仍在运行，需继续读取计划确认结束。删除计划会清理或取消任务，尚未退出时返回 `202`。导入请求限制为 500 MiB，还受到解压安全检查约束。

快照下载是文件响应。恢复会改变实例数据，触发后应读取 `/api/admin/restore/status` 和 `/api/restore/progress`，不能将触发成功认定为恢复完成。恢复进度入口不要求账号会话，必须通过 `X-Restore-Token` 传递任务专用凭据，无效或过期返回 `401`，凭据不可放入 URL。管理状态入口仍要求管理员。快照和版本功能均有独立启用条件。

## 领域路由清单

下列路径均为实例基础路径之后的完整路径。健康接口为 `/api/health`，公开头像和品牌静态文件没有 `/api` 前缀。表中字段来自处理器实际读取的 JSON、查询和表单名称。`:id` 等路径参数须替换为服务器返回的真实标识。返回对象没有统一的 `data` 包装。

### apikeys

| 方法 | 路径 | 权限 | 请求字段 |
| --- | --- | --- | --- |
| POST | `/api/api-keys` | 管理员，拒绝 API 密钥 | `name`、`userId`、`role`、`expiresAt` |
| GET | `/api/api-keys` | 管理员，拒绝 API 密钥 | 路径参数或无独立字段 |
| DELETE | `/api/api-keys/:id` | 管理员，拒绝 API 密钥 | 路径参数或无独立字段 |


### assets

| 方法 | 路径 | 权限 | 请求字段 |
| --- | --- | --- | --- |
| GET | `/api/pages/:id/assets` | 按读取设置 | 路径参数或无独立字段 |
| POST | `/api/pages/:id/assets` | 编辑者或管理员 | multipart `file` |
| PUT | `/api/pages/:id/assets/rename` | 编辑者或管理员 | `old_filename`、`new_filename` |
| DELETE | `/api/pages/:id/assets/:name` | 编辑者或管理员 | 路径参数或无独立字段 |


### auth

| 方法 | 路径 | 权限 | 请求字段 |
| --- | --- | --- | --- |
| POST | `/api/auth/login` | 无需预先登录 | `identifier`、`password` |
| POST | `/api/auth/refresh-token` | 无需预先登录 | 路径参数或无独立字段 |
| GET | `/api/config` | 无需预先登录 | 路径参数或无独立字段 |
| GET | `/api/auth/me` | 无需预先登录 | 路径参数或无独立字段 |
| POST | `/api/auth/logout` | 登录身份，拒绝 API 密钥 | 路径参数或无独立字段 |
| POST | `/api/users` | 管理员，拒绝 API 密钥 | `username`、`email`、`password`、`role` |
| GET | `/api/users` | 管理员 | 路径参数或无独立字段 |
| PUT | `/api/users/:id` | 管理员，拒绝 API 密钥 | `username`、`email`、`password`、`role` |
| DELETE | `/api/users/:id` | 管理员，拒绝 API 密钥 | 路径参数或无独立字段 |
| POST | `/api/users/invite` | 管理员，拒绝 API 密钥 | `username`、`email`、`role` |
| POST | `/api/users/:id/invite/resend` | 管理员，拒绝 API 密钥 | 路径参数或无独立字段 |
| PUT | `/api/users/me/password` | 登录身份，拒绝 API 密钥 | `oldPassword`、`newPassword` |
| POST | `/api/auth/password/forgot` | 无需预先登录 | `identifier` |
| POST | `/api/auth/password/reset` | 无需预先登录 | `token`、`newPassword` |
| POST | `/api/auth/invite/accept` | 无需预先登录 | `token`、`newPassword` |


### avatar

| 方法 | 路径 | 权限 | 请求字段 |
| --- | --- | --- | --- |
| GET | `/avatars/:userId` | 无需预先登录 | 路径参数或无独立字段 |
| GET | `/api/user/avatar` | 登录身份 | 路径参数或无独立字段 |
| POST | `/api/user/avatar` | 登录身份 | multipart `file` |
| DELETE | `/api/user/avatar` | 登录身份 | 路径参数或无独立字段 |


### branding

| 方法 | 路径 | 权限 | 请求字段 |
| --- | --- | --- | --- |
| GET | `/api/branding` | 无需预先登录 | 路径参数或无独立字段 |
| GET | `/branding/:filename` | 无需预先登录 | 路径参数或无独立字段 |
| GET | `/favicon.ico` | 无需预先登录 | 路径参数或无独立字段 |
| PUT | `/api/branding` | 管理员 | `siteName` |
| POST | `/api/branding/logo` | 管理员 | multipart `file` |
| POST | `/api/branding/favicon` | 管理员 | multipart `file` |
| DELETE | `/api/branding/logo` | 管理员 | 路径参数或无独立字段 |
| DELETE | `/api/branding/favicon` | 管理员 | 路径参数或无独立字段 |


### health

| 方法 | 路径 | 权限 | 请求字段 |
| --- | --- | --- | --- |
| GET | `/api/health` | 无需预先登录 | 路径参数或无独立字段 |


### importer

| 方法 | 路径 | 权限 | 请求字段 |
| --- | --- | --- | --- |
| POST | `/api/import/plan` | 编辑者或管理员 | `file`、`targetBasePath` |
| GET | `/api/import/plan` | 编辑者或管理员 | 路径参数或无独立字段 |
| POST | `/api/import/execute` | 编辑者或管理员 | 路径参数或无独立字段 |
| DELETE | `/api/import/plan` | 编辑者或管理员 | 路径参数或无独立字段 |


### instancesettings

| 方法 | 路径 | 权限 | 请求字段 |
| --- | --- | --- | --- |
| GET | `/api/admin/settings/preferences` | 管理员 | 路径参数或无独立字段 |
| PUT | `/api/admin/settings/preferences` | 管理员，拒绝 API 密钥 | 偏好设置局部 JSON 对象 |
| PUT | `/api/admin/settings/public-access` | 管理员，拒绝 API 密钥 | `enabled` 布尔值 |
| PUT | `/api/admin/settings/toc-display` | 管理员 | `mode` 或 `alwaysShow` |


### links

| 方法 | 路径 | 权限 | 请求字段 |
| --- | --- | --- | --- |
| GET | `/api/pages/:id/links` | 按读取设置 | 路径参数或无独立字段 |
| GET | `/api/links/broken` | 管理员 | 路径参数或无独立字段 |


### pages

| 方法 | 路径 | 权限 | 请求字段 |
| --- | --- | --- | --- |
| GET | `/api/tree` | 按读取设置 | 路径参数或无独立字段 |
| GET | `/api/pages/:id` | 按读取设置 | 路径参数或无独立字段 |
| GET | `/api/pages/lookup` | 按读取设置 | `path` |
| GET | `/api/pages/by-path` | 按读取设置 | `path` |
| GET | `/api/pages/by-title` | 按读取设置 | `title` |
| GET | `/api/pages/permalink/:id` | 按读取设置 | 路径参数或无独立字段 |
| GET | `/api/pages/slug-suggestion` | 编辑者或管理员 | `title`、`parentId`、`currentId` |
| POST | `/api/pages` | 编辑者或管理员 | `parentId`、`title`、`slug`、`kind` |
| PUT | `/api/pages/:id` | 编辑者或管理员 | `version`、`title`、`slug`、`content`、`tags`、`properties` |
| DELETE | `/api/pages/:id` | 编辑者或管理员 | `recursive`、`version` |
| PUT | `/api/pages/:id/move` | 编辑者或管理员 | `version`、`parentId`、`position` |
| PUT | `/api/pages/:id/sort` | 编辑者或管理员 | `orderedIds` |
| PUT | `/api/pages/:id/pin` | 编辑者或管理员 | `version`、`pinned` |
| PUT | `/api/pages/:id/favorite` | 登录身份 | 路径参数或无独立字段 |
| DELETE | `/api/pages/:id/favorite` | 登录身份 | 路径参数或无独立字段 |
| GET | `/api/favorites` | 登录身份 | 路径参数或无独立字段 |
| POST | `/api/pages/ensure` | 编辑者或管理员 | `path`、`title`、`kind` |
| POST | `/api/pages/convert/:id` | 编辑者或管理员 | `targetKind`、`version` |
| POST | `/api/pages/copy/:id` | 编辑者或管理员 | `targetParentId`、`title`、`slug` |
| POST | `/api/pages/:id/refactor/preview` | 编辑者或管理员 | `kind`、`title`、`slug`、`content`、`parentId` |
| POST | `/api/pages/:id/refactor/apply` | 编辑者或管理员 | `version`、`kind`、`title`、`slug`、`content`、`parentId`、`rewriteLinks`、`position` |


### properties

| 方法 | 路径 | 权限 | 请求字段 |
| --- | --- | --- | --- |
| GET | `/api/properties` | 按读取设置 | `q`、`limit` |
| GET | `/api/properties/pages` | 按读取设置 | `key`、`value` |


### restore

| 方法 | 路径 | 权限 | 请求字段 |
| --- | --- | --- | --- |
| GET | `/api/restore/progress` | 专用恢复进度凭据 | `X-Restore-Token` 请求头 |
| POST | `/api/admin/restore/upload` | 管理员，拒绝 API 密钥 | `file` |
| POST | `/api/admin/restore/:id` | 管理员，拒绝 API 密钥 | 路径参数或无独立字段 |
| GET | `/api/admin/restore/status` | 管理员 | 路径参数或无独立字段 |
| POST | `/api/admin/restore/self-restart` | 管理员，拒绝 API 密钥 | 路径参数或无独立字段 |


### resync

| 方法 | 路径 | 权限 | 请求字段 |
| --- | --- | --- | --- |
| POST | `/api/admin/resync` | 管理员 | 路径参数或无独立字段 |
| GET | `/api/admin/resync/status` | 管理员 | 路径参数或无独立字段 |


### revisions

| 方法 | 路径 | 权限 | 请求字段 |
| --- | --- | --- | --- |
| GET | `/api/pages/:id/revisions` | 登录身份 | `limit`、`cursor` |
| GET | `/api/pages/:id/revisions/latest` | 登录身份 | 路径参数或无独立字段 |
| GET | `/api/pages/:id/revisions/compare` | 登录身份 | `base`、`target` |
| GET | `/api/pages/:id/revisions/:revisionId/assets/*name` | 登录身份 | 路径参数或无独立字段 |
| GET | `/api/pages/:id/revisions/:revisionId` | 登录身份 | 路径参数或无独立字段 |
| POST | `/api/pages/:id/revisions/:revisionId/restore` | 编辑者或管理员 | 路径参数或无独立字段 |


### search

| 方法 | 路径 | 权限 | 请求字段 |
| --- | --- | --- | --- |
| GET | `/api/search/status` | 按读取设置 | 路径参数或无独立字段 |
| GET | `/api/search` | 按读取设置 | `q`、`offset`、`limit` |


### snapshot

| 方法 | 路径 | 权限 | 请求字段 |
| --- | --- | --- | --- |
| GET | `/api/admin/snapshot/status` | 管理员 | 路径参数或无独立字段 |
| PUT | `/api/admin/snapshot/settings` | 管理员 | 路径参数或无独立字段 |
| GET | `/api/admin/snapshot` | 管理员 | 路径参数或无独立字段 |
| POST | `/api/admin/snapshot` | 管理员 | 路径参数或无独立字段 |
| GET | `/api/admin/snapshot/:id/download` | 管理员，拒绝 API 密钥 | 路径参数或无独立字段 |
| DELETE | `/api/admin/snapshot/:id` | 管理员 | 路径参数或无独立字段 |


### tags

| 方法 | 路径 | 权限 | 请求字段 |
| --- | --- | --- | --- |
| GET | `/api/tags` | 按读取设置 | `q`、`limit` |
| GET | `/api/tags/pages` | 按读取设置 | 路径参数或无独立字段 |

## API 密钥和权限

管理员创建密钥时提交 `name`、`userId`、`role`、`expiresAt`。非空到期时间采用 RFC3339。成功返回 `key` 和一次性的 `secret`，列表不返回密钥明文。撤销成功返回 `204`。调用业务接口使用 `Authorization: Bearer <secret>`，仍受用户和密钥角色约束。

读取组遵循实例当前公开访问设置，关闭公开访问后要求登录。编辑接口要求编辑者或管理员，管理接口要求管理员。邮件邀请入口始终注册，邮件服务未启用时返回功能关闭错误。版本接口仅在功能启用时注册。调用前读取 `/api/config` 核对 `enableRevision`、`enableLinkRefactor`、`enableApiKeyManagement`、`snapshotEnabled`、`smtpEnabled` 等实际设置。

## 静态内容和事件流

`/assets/` 提供页面资源，要求公开读取或有效身份，禁止目录列表。`/avatars/:userId` 提供公开头像，无需预先登录。`/branding/:filename` 与 `/favicon.ico` 提供公开品牌资源，无需预先登录。`/custom.css` 仅在实例配置该内容时提供。`GET /api/settings/events` 提供设置变化事件流，应流式读取，不能等待完整 JSON。

## 状态和错误处理

`200` 为同步成功，`201` 为创建成功，`202` 为异步处理，`204` 没有响应正文。`400` 表示输入无效，`401` 表示缺少有效身份，`403` 表示权限、CSRF 或功能限制，`404` 表示目标不存在，`409` 表示版本或状态冲突，`413` 表示上传限制，`429` 表示限流。刷新凭据无效使用 `422`。

错误响应包含可读说明，领域错误还提供错误代码。调用方应同时检查 HTTP 状态和响应字段，不应只匹配英文消息文本。静态资源、下载和事件流不是普通 JSON，必须按照响应 Content-Type 读取。内部错误结合服务器日志排查，不以无限重试代替定位。
