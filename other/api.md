# HTTP 接口与调用参考

## 地址、身份与请求顺序

业务 API 路径前添加实例 BASE_PATH。前端入口当前为 24691，后端 24692。健康和配置读取不代表已登录。登录成功写入认证与 CSRF Cookie，不直接在响应中返回访问令牌明文。Cookie 写请求携带 X-CSRF-Token，值从 CSRF Cookie 读取；密钥身份仅能使用路由允许的接口，Cookie 专用操作不能绕过。

1. GET /api/config 获取运行信息。
2. POST /api/auth/login，JSON 为 identifier 与 password。
3. 保留服务器 Cookie，GET /api/auth/me 确认身份。
4. GET /api/tree，按实际页面 ID 读取正文。
5. 写操作使用当前版本，成功后重新读取确认，不复用过期 version。

## 最小调用

```sh
read -r -p "服务 URL，含可选前缀: " BASE
curl --fail "$BASE/api/health"
read -r -p "管理员标识: " IDENTIFIER
read -r -s -p "密码: " PASSWORD
printf "\n"
LOGIN=$(IDENTIFIER="$IDENTIFIER" PASSWORD="$PASSWORD" python3 -c 'import os,json; print(json.dumps({"identifier":os.environ["IDENTIFIER"],"password":os.environ["PASSWORD"]}))')
curl --fail -c /tmp/leafdoc-cookies -H "Content-Type: application/json" -d "$LOGIN" "$BASE/api/auth/login"
unset PASSWORD LOGIN
curl --fail -b /tmp/leafdoc-cookies "$BASE/api/auth/me"
curl --fail -b /tmp/leafdoc-cookies "$BASE/api/tree"
rm -f /tmp/leafdoc-cookies
```

生产自动化应使用权限匹配的凭据并保护 Cookie 文件。前述示例只执行读操作，写操作还需 CSRF。HTTPS 应使用可信 CA，不在示例中关闭证书校验。

## 页面写入约束

| 操作 | 输入与结果 |
| --- | --- |
| POST /api/pages | title、slug、可空 parentId、kind。kind 为 page 或 section，父节点必须是 section。返回新页面 |
| PUT /api/pages/:id | 使用最新 version，title、slug、content 与支持的元数据。返回保存后的页面 |
| DELETE /api/pages/:id | 查询 version 与 recursive=true，按页面删除用例清理子树及关联内容，成功返回确认 |
| 导入 | 先上传创建计划，再检查、执行；计划过期或树变化应重新规划 |
| 快照恢复 | 管理员会话启动，任务凭据查询进度，恢复期间遵循写入门禁 |

## 路由逐项参考

以下每节从当前注册与处理器提取。路径参数中的 :id 和 *name 应替换为实际值，不原样发送。输入代码中的 json 标签是字段名称，binding 是必填约束，Query、FormFile 表示查询串和 multipart。响应表达式对应处理器真实 JSON、文件或状态返回，具体对象由列出的类型和用例构造，不额外编造返回字段。

### POST /api/api-keys

要求名称与所属用户，按 RFC3339 解析非空到期时间，再交给用例验证。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | apikeys / handleCreateAPIKey |
| 身份 | 需要认证；管理员；Cookie 会话专用 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithAPIKeyStatusError(c, http.StatusBadRequest, ErrCodeAPIKeyInvalidRequest, "Invalid request", "invalid request")；respondWithAPIKeyStatusError(c, http.StatusBadRequest, ErrCodeAPIKeyInvalidExpiry, "Invalid expiry date", "invalid expiry date")；respondWithAPIKeyError(c, err)；c.JSON(http.StatusCreated, gin.H{ |
| 源码 | backend/internal/wiki/apikeys/routes.go |

请求与响应实现：

```go
func (r *Routes) handleCreateAPIKey(c *gin.Context) {
	admin := authmw.MustGetUser(c)
	if admin == nil {
		return
	}

	var req struct {
		Name      string `json:"name" binding:"required"`
		UserID    string `json:"userId" binding:"required"`
		Role      string `json:"role"`
		ExpiresAt string `json:"expiresAt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithAPIKeyStatusError(c, http.StatusBadRequest, ErrCodeAPIKeyInvalidRequest, "Invalid request", "invalid request")
		return
	}

	var expiresAt *time.Time
	// 空白判断不改变解析输入，非空值仍用原字符串解析，两端多余空白不会被去掉。
	if strings.TrimSpace(req.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			respondWithAPIKeyStatusError(c, http.StatusBadRequest, ErrCodeAPIKeyInvalidExpiry, "Invalid expiry date", "invalid expiry date")
			return
		}
		expiresAt = &parsed
	}

	out, err := r.createAPIKey.Execute(c.Request.Context(), CreateAPIKeyInput{
		Name: req.Name, UserID: req.UserID, Role: req.Role, ExpiresAt: expiresAt, CreatedBy: admin.ID,
	})
	if err != nil {
		respondWithAPIKeyError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"key":    apiKeyResponse(out.Key),
		"secret": out.Secret,
	})
}

// handleListAPIKeys 将每条领域记录转换为公开字段，输出数组而非直接编码含哈希的结构体。
```

### GET /api/api-keys

将每条领域记录转换为公开字段，输出数组而非直接编码含哈希的结构体。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | apikeys / handleListAPIKeys |
| 身份 | 需要认证；管理员；Cookie 会话专用 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithAPIKeyError(c, err)；c.JSON(http.StatusOK, keys) |
| 源码 | backend/internal/wiki/apikeys/routes.go |

请求与响应实现：

```go
func (r *Routes) handleListAPIKeys(c *gin.Context) {
	out, err := r.listAPIKeys.Execute(c.Request.Context())
	if err != nil {
		respondWithAPIKeyError(c, err)
		return
	}
	keys := make([]gin.H, len(out.Keys))
	for i, k := range out.Keys {
		keys[i] = apiKeyResponse(k)
	}
	c.JSON(http.StatusOK, keys)
}

// handleRevokeAPIKey 按 URL 记录标识撤销，成功返回无响应体的 204。
```

### DELETE /api/api-keys/:id

按 URL 记录标识撤销，成功返回无响应体的 204。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | apikeys / handleRevokeAPIKey |
| 身份 | 需要认证；管理员；Cookie 会话专用 |
| 路由参数与查询 | c.Param("id") |
| 响应调用 | respondWithAPIKeyError(c, err)；c.Status(http.StatusNoContent) |
| 源码 | backend/internal/wiki/apikeys/routes.go |

请求与响应实现：

```go
func (r *Routes) handleRevokeAPIKey(c *gin.Context) {
	id := c.Param("id")
	if err := r.revokeAPIKey.Execute(c.Request.Context(), RevokeAPIKeyInput{ID: id}); err != nil {
		respondWithAPIKeyError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// apiKeyResponse 明确选择可公开字段，不包含密钥哈希或明文令牌。
// 时间统一转为 UTC RFC3339 文本，可选时间指针为空时省略相应字段。
// 调用方必须传入非空记录。
func apiKeyResponse(key *coreauth.APIKey) gin.H {
	resp := gin.H{
		"id":        key.ID,
		"name":      key.Name,
		"userId":    key.UserID,
		"prefix":    key.Prefix,
		"role":      key.Role,
		"createdBy": key.CreatedBy,
		"createdAt": key.CreatedAt.UTC().Format(time.RFC3339),
	}
	if key.ExpiresAt != nil {
		resp["expiresAt"] = key.ExpiresAt.UTC().Format(time.RFC3339)
	}
	if key.LastUsedAt != nil {
		resp["lastUsedAt"] = key.LastUsedAt.UTC().Format(time.RFC3339)
	}
	if key.RevokedAt != nil {
		resp["revokedAt"] = key.RevokedAt.UTC().Format(time.RFC3339)
	}
	return resp
}
```

### GET /api/pages/:id/assets

按页面标识取得资源地址列表，以 files 字段返回，失败交给资源错误映射。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | assets / handleList |
| 身份 | 公开读取按实例设置检查 |
| 路由参数与查询 | c.Param("id") |
| 响应调用 | respondWithAssetError(c, err)；c.JSON(http.StatusOK, gin.H{"files": out.Files}) |
| 源码 | backend/internal/wiki/assets/routes.go |

请求与响应实现：

```go
func (r *Routes) handleList(c *gin.Context) {
	pageID := c.Param("id")
	out, err := r.list.Execute(c.Request.Context(), ListAssetsInput{PageID: pageID})
	if err != nil {
		respondWithAssetError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"files": out.Files})
}

// handleRename 要求两个非空文件名，从鉴权上下文取得操作者后交给重命名用例。
// 返回的 url 是资源服务形成的访问地址，不是磁盘文件路径。
```

### POST /api/pages/:id/assets

限制整个请求体的字节数，再解析名为 file 的 multipart 文件。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | assets / handleUpload |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | c.Param("id") |
| 响应调用 | respondWithAssetStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeAssetFileTooLarge, "File too large", "file too large")；respondWithAssetStatusError(c, http.StatusBadRequest, ErrCodeAssetMissingFile, "Missing file", "missing file")；respondWithAssetError(c, err)；c.JSON(http.StatusCreated, gin.H{"file": out.URL}) |
| 源码 | backend/internal/wiki/assets/routes.go |

请求与响应实现：

```go
func (r *Routes) handleUpload(maxUploadSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 限制包括 multipart 边界与字段开销，不只计算单个文件的内容字节。
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

		if err := c.Request.ParseMultipartForm(maxMultipartMemory); err != nil {
			respondWithAssetStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeAssetFileTooLarge, "File too large", "file too large")
			return
		}

		pageID := c.Param("id")
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			respondWithAssetStatusError(c, http.StatusBadRequest, ErrCodeAssetMissingFile, "Missing file", "missing file")
			return
		}
		defer func() {
			if err := file.Close(); err != nil {
				r.log.Error("could not close uploaded file", "error", err)
			}
		}()

		user := authmw.MustGetUser(c)
		if user == nil {
			return
		}

		out, err := r.upload.Execute(c.Request.Context(), UploadAssetInput{
			UserID: user.ID, PageID: pageID, File: file, Filename: header.Filename, MaxBytes: maxUploadSize,
		})
		if err != nil {
			respondWithAssetError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"file": out.URL})
	}
}

// handleList 按页面标识取得资源地址列表，以 files 字段返回，失败交给资源错误映射。
```

### PUT /api/pages/:id/assets/rename

要求两个非空文件名，从鉴权上下文取得操作者后交给重命名用例。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | assets / handleRename |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | c.Param("id") |
| 响应调用 | respondWithAssetStatusError(c, http.StatusBadRequest, ErrCodeAssetInvalidPayload, "Invalid payload", "invalid payload")；respondWithAssetError(c, err)；c.JSON(http.StatusOK, gin.H{"url": out.URL}) |
| 源码 | backend/internal/wiki/assets/routes.go |

请求与响应实现：

```go
func (r *Routes) handleRename(c *gin.Context) {
	pageID := c.Param("id")
	var req struct {
		OldFilename string `json:"old_filename" binding:"required"`
		NewFilename string `json:"new_filename" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithAssetStatusError(c, http.StatusBadRequest, ErrCodeAssetInvalidPayload, "Invalid payload", "invalid payload")
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	out, err := r.rename.Execute(c.Request.Context(), RenameAssetInput{
		UserID: user.ID, PageID: pageID, OldFilename: req.OldFilename, NewFilename: req.NewFilename,
	})
	if err != nil {
		respondWithAssetError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": out.URL})
}

// handleDelete 拒绝空文件名，再按页面与文件名执行删除，成功返回固定确认消息。
```

### DELETE /api/pages/:id/assets/:name

拒绝空文件名，再按页面与文件名执行删除，成功返回固定确认消息。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | assets / handleDelete |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | c.Param("id")；c.Param("name") |
| 响应调用 | respondWithAssetStatusError(c, http.StatusBadRequest, ErrCodeAssetMissingName, "Missing filename", "missing filename")；respondWithAssetError(c, err)；c.JSON(http.StatusOK, gin.H{"message": "asset deleted"}) |
| 源码 | backend/internal/wiki/assets/routes.go |

请求与响应实现：

```go
func (r *Routes) handleDelete(c *gin.Context) {
	pageID := c.Param("id")
	filename := c.Param("name")
	if filename == "" {
		respondWithAssetStatusError(c, http.StatusBadRequest, ErrCodeAssetMissingName, "Missing filename", "missing filename")
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	if err := r.delete.Execute(c.Request.Context(), DeleteAssetInput{
		UserID: user.ID, PageID: pageID, Filename: filename,
	}); err != nil {
		respondWithAssetError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "asset deleted"})
}
```

### POST /api/auth/login

要求标识和密码，通过用例签发会话后写入 CSRF 与认证 Cookie。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | auth / handleLogin |
| 身份 | 无需账号会话，仍受限流或专用凭据约束 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidPayload, "Invalid login payload", "invalid login payload")；respondWithAuthError(c, err)；respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed,；respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed, "Failed to set authentication cookies", "failed to set authentication cookies")；c.JSON(http.StatusOK, gin.H{ |
| 源码 | backend/internal/wiki/auth/routes.go |

请求与响应实现：

```go
func (r *Routes) handleLogin(rctx httpinternal.RouterContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Identifier string `json:"identifier" binding:"required"`
			Password   string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidPayload, "Invalid login payload", "invalid login payload")
			return
		}
		out, err := r.login.Execute(c.Request.Context(), LoginInput{
			Identifier: req.Identifier, Password: req.Password,
		})
		if err != nil {
			respondWithAuthError(c, err)
			return
		}
		if _, err := rctx.CSRFCookie.Issue(c); err != nil {
			writeAuthCookieError(c, err,
				"HTTPS is required for login cookies. Use HTTPS or start LeafDoc with --allow-insecure for trusted plain HTTP setups.",
				errFailedToIssueCSRFCookie,
				"failed to issue login CSRF cookie",
			)
			return
		}
		if err := rctx.AuthCookies.Set(c, out.Token.Token, out.Token.RefreshToken); err != nil {
			if errors.Is(err, utils.ErrHTTPSRequired) {
				respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed,
					httpsRequiredUserMsg,
					httpsRequiredLogMsg)
				return
			}
			respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed, "Failed to set authentication cookies", "failed to set authentication cookies")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":              "Login successful",
			"user":                 out.Token.User,
			"accessTokenExpiresAt": out.Token.AccessTokenExpiresAt,
		})
	}
}

// handleLogout 尝试读取并撤销刷新会话，再依次清理认证及 CSRF Cookie。
// 读取或撤销失败只记录警告，Cookie 清理失败才在该路径返回错误。
// 清理完成返回成功，不据此保证已签发的访问 JWT 失效。
```

### POST /api/auth/refresh-token

从 Cookie 读取刷新令牌，缺失或读取错误统一返回 422。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | auth / handleRefreshToken |
| 身份 | 无需账号会话，仍受限流或专用凭据约束 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithAuthStatusError(c, http.StatusUnprocessableEntity, ErrCodeAuthInvalidRefreshToken, "Missing or invalid refresh token", "missing or invalid refresh token")；respondWithAuthError(c, err)；respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed,；respondWithAuthStatusError(c, http.StatusInternalServerError, ErrCodeAuthCookieFailed, "Failed to set authentication cookies", "failed to set authentication cookies")；c.JSON(http.StatusOK, gin.H{ |
| 源码 | backend/internal/wiki/auth/routes.go |

请求与响应实现：

```go
func (r *Routes) handleRefreshToken(rctx httpinternal.RouterContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		rt, err := rctx.AuthCookies.ReadRefresh(c)
		if err != nil || rt == "" {
			respondWithAuthStatusError(c, http.StatusUnprocessableEntity, ErrCodeAuthInvalidRefreshToken, "Missing or invalid refresh token", "missing or invalid refresh token")
			return
		}
		out, err := r.refreshToken.Execute(c.Request.Context(), RefreshTokenInput{RefreshToken: rt})
		if err != nil {
			respondWithAuthError(c, err)
			return
		}
		if _, err := rctx.CSRFCookie.Issue(c); err != nil {
			writeAuthCookieError(c, err,
				httpsRequiredUserMsg,
				errFailedToIssueCSRFCookie,
				"failed to issue refresh CSRF cookie",
			)
			return
		}
		if err := rctx.AuthCookies.Set(c, out.Token.Token, out.Token.RefreshToken); err != nil {
			if errors.Is(err, utils.ErrHTTPSRequired) {
				respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed,
					httpsRequiredUserMsg,
					httpsRequiredLogMsg)
				return
			}
			respondWithAuthStatusError(c, http.StatusInternalServerError, ErrCodeAuthCookieFailed, "Failed to set authentication cookies", "failed to set authentication cookies")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":              "Token refreshed",
			"user":                 out.Token.User,
			"accessTokenExpiresAt": out.Token.AccessTokenExpiresAt,
		})
	}
}

// handleCreateUser 解析四个必填字段，将进一步验证和创建交给用例，成功返回 201 公开账号。
```

### GET /api/config

返回功能设置、动态公开与目录状态及偏好数据，并禁止客户端存储响应。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | auth / handleConfig |
| 身份 | 无需账号会话，仍受限流或专用凭据约束 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | c.Header("Cache-Control", "no-store")；c.JSON(http.StatusOK, config) |
| 源码 | backend/internal/wiki/auth/routes.go |

请求与响应实现：

```go
func (r *Routes) handleConfig(ctx httpinternal.RouterContext) gin.HandlerFunc {
	opts := ctx.Opts
	return func(c *gin.Context) {
		if _, err := ctx.CSRFCookie.Issue(c); err != nil {
			writeAuthCookieError(c, err,
				httpsRequiredUserMsg,
				errFailedToIssueCSRFCookie,
				"failed to issue config CSRF cookie",
			)
			return
		}
		config := gin.H{
			"publicAccess":            opts.PublicAccess.Enabled(),
			"publicAccessEnvManaged":  opts.PublicAccess.EnvManaged(),
			"alwaysShowToc":           opts.AlwaysShowToc.AlwaysShow(),
			"editorLimit":             opts.EditorLimit,
			"hideLinkMetadataSection": opts.HideLinkMetadataSection,
			"basePath":                opts.BasePath,
			"maxAssetUploadSizeBytes": opts.MaxAssetUploadSizeBytes,
			// 头像限制直接使用头像模块的权威常量与扩展名列表，不从资源上传限制推导。
			"maxAvatarUploadSizeBytes": coreavatar.MaxUploadSize,
			"avatarAllowedExts":        coreavatar.AllowedExts(),
			"enableRevision":           opts.EnableRevision,
			"enableLinkRefactor":       opts.EnableLinkRefactor,
			"enableApiKeyManagement":   opts.EnableAPIKeyManagement,
			"snapshotEnabled":          opts.SnapshotEnabled,
			"smtpEnabled":              opts.SMTPEnabled,
			"httpRemoteUserEnabled":    opts.HTTPRemoteUser.Enabled,
			"loginUrl":                 opts.LoginURL,
			"logoutUrl":                opts.LogoutURL,
			"userManagementUrl":        opts.UserManagementURL,
			"defaultLanguage":          opts.DefaultLanguage,
		}
		prefs := preferences.Defaults()
		if opts.Preferences != nil {
			prefs = opts.Preferences.Get()
		}
		config["preferences"] = prefs
		if opts.SettingsVersion != nil {
			config["settingsVersion"] = opts.SettingsVersion()
			config["settingsInstance"] = opts.SettingsInstance
		}
		config["defaultLanguage"] = prefs.Language
		config["tocMode"] = opts.AlwaysShowToc.Mode()
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, config)
	}
}

// handleMe 设置禁止缓存的响应头，从已有上下文返回有限的用户身份字段。
// 无用户时返回 200 与 null，不在这里触发必须登录的错误响应。
```

### GET /api/auth/me

设置禁止缓存的响应头，从已有上下文返回有限的用户身份字段。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | auth / handleMe |
| 身份 | 需要认证 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | c.Header("Cache-Control", "no-store")；c.Header("Pragma", "no-cache")；c.Header("Expires", time.Unix(0, 0).UTC().Format(http.TimeFormat))；c.JSON(http.StatusOK, nil)；c.JSON(http.StatusOK, gin.H{ |
| 源码 | backend/internal/wiki/auth/routes.go |

请求与响应实现：

```go
func (r *Routes) handleMe(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", time.Unix(0, 0).UTC().Format(http.TimeFormat))

	user := authmw.TryGetUser(c)
	if user == nil {
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"mustChangePassword": user.MustChangePassword,
		"id":                 user.ID,
		"username":           user.Username,
		"email":              user.Email,
		"role":               user.Role,
	})
}

// handleLogin 要求标识和密码，通过用例签发会话后写入 CSRF 与认证 Cookie。
// Cookie 失败时不撤销已经签发的服务端会话，成功响应不直接返回令牌明文。
```

### POST /api/auth/logout

尝试读取并撤销刷新会话，再依次清理认证及 CSRF Cookie。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | auth / handleLogout |
| 身份 | 需要认证；Cookie 会话专用 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed, "Failed to clear authentication cookies", "failed to clear authentication cookies")；respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCsrfFailed, "Failed to clear CSRF cookie", "failed to clear csrf cookie")；c.JSON(http.StatusOK, gin.H{"message": "Logout successful"}) |
| 源码 | backend/internal/wiki/auth/routes.go |

请求与响应实现：

```go
func (r *Routes) handleLogout(rctx httpinternal.RouterContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		refreshToken, err := rctx.AuthCookies.ReadRefresh(c)
		if err != nil {
			slog.Default().Warn("could not read refresh token during logout; refresh token will not be revoked", "error", err)
		}
		if refreshToken != "" {
			if err := r.logout.Execute(c.Request.Context(), LogoutInput{RefreshToken: refreshToken}); err != nil {
				slog.Default().Warn("failed to revoke refresh token during logout", "error", err)
			}
		}
		if err := rctx.AuthCookies.Clear(c); err != nil {
			respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed, "Failed to clear authentication cookies", "failed to clear authentication cookies")
			return
		}
		if err := rctx.CSRFCookie.Clear(c); err != nil {
			respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCsrfFailed, "Failed to clear CSRF cookie", "failed to clear csrf cookie")
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
	}
}

// handleRefreshToken 从 Cookie 读取刷新令牌，缺失或读取错误统一返回 422。
// 刷新成功后先签发或复用 CSRF，再写认证 Cookie，Cookie 失败不回滚令牌轮换。
```

### POST /api/users

解析四个必填字段，将进一步验证和创建交给用例，成功返回 201 公开账号。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | auth / handleCreateUser |
| 身份 | 需要认证；管理员；Cookie 会话专用 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)；respondWithAuthError(c, err)；c.JSON(http.StatusCreated, out.User) |
| 源码 | backend/internal/wiki/auth/routes.go |

请求与响应实现：

```go
func (r *Routes) handleCreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	out, err := r.createUser.Execute(c.Request.Context(), CreateUserInput{
		Username: req.Username, Email: req.Email, Password: req.Password, Role: req.Role,
	})
	if err != nil {
		respondWithAuthError(c, err)
		return
	}
	c.JSON(http.StatusCreated, out.User)
}

// handleGetUsers 返回公开账号数组，不将用户存储的密码摘要编码到响应。
```

### GET /api/users

返回公开账号数组，不将用户存储的密码摘要编码到响应。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | auth / handleGetUsers |
| 身份 | 需要认证；管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithAuthError(c, err)；c.JSON(http.StatusOK, out.Users) |
| 源码 | backend/internal/wiki/auth/routes.go |

请求与响应实现：

```go
func (r *Routes) handleGetUsers(c *gin.Context) {
	out, err := r.getUsers.Execute(c.Request.Context())
	if err != nil {
		respondWithAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Users)
}

// handleUpdateUser 从路由取目标标识，从上下文取实际请求者权限，避免请求体伪造管理员标记。
// 用户名与邮箱必填，密码与角色可省略，保留规则由更新用例处理。
```

### PUT /api/users/:id

从路由取目标标识，从上下文取实际请求者权限，避免请求体伪造管理员标记。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | auth / handleUpdateUser |
| 身份 | 需要认证；管理员；Cookie 会话专用 |
| 路由参数与查询 | c.Param("id") |
| 响应调用 | respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)；respondWithAuthError(c, err)；c.JSON(http.StatusOK, out.User) |
| 源码 | backend/internal/wiki/auth/routes.go |

请求与响应实现：

```go
func (r *Routes) handleUpdateUser(c *gin.Context) {
	id := c.Param("id")
	requester := authmw.MustGetUser(c)
	if requester == nil {
		return
	}
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	out, err := r.updateUser.Execute(c.Request.Context(), UpdateUserInput{
		ID: id, Username: req.Username, Email: req.Email, Password: req.Password, Role: req.Role,
		RequesterID: requester.ID, RequesterIsAdmin: requester.HasRole(coreauth.RoleAdmin),
	})
	if err != nil {
		respondWithAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.User)
}

// handleDeleteUser 将路由标识交给删除用例，成功返回无响应体的 204。
```

### DELETE /api/users/:id

将路由标识交给删除用例，成功返回无响应体的 204。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | auth / handleDeleteUser |
| 身份 | 需要认证；管理员；Cookie 会话专用 |
| 路由参数与查询 | c.Param("id") |
| 响应调用 | respondWithAuthError(c, err)；c.Status(http.StatusNoContent) |
| 源码 | backend/internal/wiki/auth/routes.go |

请求与响应实现：

```go
func (r *Routes) handleDeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := r.deleteUser.Execute(c.Request.Context(), DeleteUserInput{ID: id}); err != nil {
		respondWithAuthError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// handleChangeOwnPassword 以当前用户标识提交新旧密码，不允许请求体选择其他账号。
// 当前刷新 Cookie 交给用例决定保留哪个持久化会话，读取失败仍继续改密流程。
```

### POST /api/users/invite

返回 201 与账号和发送标记，邮件发送失败不等于账号创建失败。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | auth / handleInviteUser |
| 身份 | 需要认证；管理员；Cookie 会话专用 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)；respondWithAuthError(c, err)；c.JSON(http.StatusCreated, gin.H{ |
| 源码 | backend/internal/wiki/auth/routes.go |

请求与响应实现：

```go
func (r *Routes) handleInviteUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Role     string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	out, err := r.inviteUser.Execute(c.Request.Context(), InviteUserInput{
		Username: req.Username, Email: req.Email, Role: req.Role,
	})
	if err != nil {
		respondWithAuthError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"user":      out.User,
		"emailSent": out.EmailSent,
	})
}

// handleResendInvite 仅按现有账号标识重新发送，成功返回 204，失败按鉴权领域错误映射。
```

### POST /api/users/:id/invite/resend

仅按现有账号标识重新发送，成功返回 204，失败按鉴权领域错误映射。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | auth / handleResendInvite |
| 身份 | 需要认证；管理员；Cookie 会话专用 |
| 路由参数与查询 | c.Param("id") |
| 响应调用 | respondWithAuthError(c, err)；c.Status(http.StatusNoContent) |
| 源码 | backend/internal/wiki/auth/routes.go |

请求与响应实现：

```go
func (r *Routes) handleResendInvite(c *gin.Context) {
	id := c.Param("id")
	if err := r.resendInvite.Execute(c.Request.Context(), ResendInviteInput{UserID: id}); err != nil {
		respondWithAuthError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// handleConfirmInvite 完成邀请与会话签发后写入 CSRF 和认证 Cookie，使成功响应具备登录状态。
// Cookie 写入失败不撤销已经完成的邀请或会话，响应只返回用户与访问到期时刻。
```

### PUT /api/users/me/password

以当前用户标识提交新旧密码，不允许请求体选择其他账号。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | auth / handleChangeOwnPassword |
| 身份 | 需要认证；Cookie 会话专用 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)；respondWithAuthError(c, err)；c.Status(http.StatusNoContent) |
| 源码 | backend/internal/wiki/auth/routes.go |

请求与响应实现：

```go
func (r *Routes) handleChangeOwnPassword(rctx httpinternal.RouterContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := authmw.MustGetUser(c)
		if user == nil {
			return
		}
		var req struct {
			OldPassword string `json:"oldPassword" binding:"required"`
			NewPassword string `json:"newPassword" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
			return
		}
		refreshToken, err := rctx.AuthCookies.ReadRefresh(c)
		if err != nil {
			// 空令牌使服务采用撤销全部刷新会话的回退，警告便于定位后续重新登录的原因。
			// 该操作不直接撤销访问 JWT，撤销失败的处理由用例负责。
			slog.Default().Warn("could not read refresh token while changing password; will revoke all sessions", "userID", user.ID, "error", err)
		}
		if err := r.changeOwnPassword.Execute(c.Request.Context(), ChangeOwnPasswordInput{
			UserID: user.ID, OldPassword: req.OldPassword, NewPassword: req.NewPassword, CurrentRefreshToken: refreshToken,
		}); err != nil {
			respondWithAuthError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// handleRequestPasswordReset 按原始标识检查独立额度，允许时提交异步重置请求。
// 标识额度耗尽时不调用用例，仍返回相同通用成功体，不透露是否执行发送。
```

### POST /api/auth/password/forgot

按原始标识检查独立额度，允许时提交异步重置请求。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | auth / handleRequestPasswordReset |
| 身份 | 无需账号会话，仍受限流或专用凭据约束 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)；respondWithAuthError(c, err)；c.JSON(http.StatusOK, gin.H{ |
| 源码 | backend/internal/wiki/auth/routes.go |

请求与响应实现：

```go
func (r *Routes) handleRequestPasswordReset(identifierLimiter *security.KeyedLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Identifier string `json:"identifier" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
			return
		}

		// 额度按输入文本本身区分，这里不统一大小写或裁空白；耗尽只跳过发送入口。
		if identifierLimiter.Allow(req.Identifier) {
			if err := r.requestPasswordReset.Execute(c.Request.Context(), RequestPasswordResetInput{Identifier: req.Identifier}); err != nil {
				respondWithAuthError(c, err)
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "If an account exists for that identifier, a password reset email has been sent.",
		})
	}
}

// handleConfirmPasswordReset 要求令牌和新密码，成功只返回公开用户，不建立登录 Cookie。
```

### POST /api/auth/password/reset

要求令牌和新密码，成功只返回公开用户，不建立登录 Cookie。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | auth / handleConfirmPasswordReset |
| 身份 | 无需账号会话，仍受限流或专用凭据约束 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)；respondWithAuthError(c, err)；c.JSON(http.StatusOK, gin.H{"user": out.User}) |
| 源码 | backend/internal/wiki/auth/routes.go |

请求与响应实现：

```go
func (r *Routes) handleConfirmPasswordReset(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	out, err := r.confirmPasswordReset.Execute(c.Request.Context(), ConfirmPasswordResetInput{
		Token: req.Token, NewPassword: req.NewPassword,
	})
	if err != nil {
		respondWithAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": out.User})
}

// handleInviteUser 返回 201 与账号和发送标记，邮件发送失败不等于账号创建失败。
```

### POST /api/auth/invite/accept

完成邀请与会话签发后写入 CSRF 和认证 Cookie，使成功响应具备登录状态。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | auth / handleConfirmInvite |
| 身份 | 无需账号会话，仍受限流或专用凭据约束 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)；respondWithAuthError(c, err)；respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed,；respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed, "Failed to set authentication cookies", "failed to set authentication cookies")；c.JSON(http.StatusOK, gin.H{ |
| 源码 | backend/internal/wiki/auth/routes.go |

请求与响应实现：

```go
func (r *Routes) handleConfirmInvite(rctx httpinternal.RouterContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Token       string `json:"token" binding:"required"`
			NewPassword string `json:"newPassword" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
			return
		}
		out, err := r.confirmInvite.Execute(c.Request.Context(), ConfirmInviteInput{
			Token: req.Token, NewPassword: req.NewPassword,
		})
		if err != nil {
			respondWithAuthError(c, err)
			return
		}
		if _, err := rctx.CSRFCookie.Issue(c); err != nil {
			writeAuthCookieError(c, err,
				"HTTPS is required for login cookies. Use HTTPS or start LeafDoc with --allow-insecure for trusted plain HTTP setups.",
				errFailedToIssueCSRFCookie,
				"failed to issue invite-accept CSRF cookie",
			)
			return
		}
		if err := rctx.AuthCookies.Set(c, out.Token.Token, out.Token.RefreshToken); err != nil {
			if errors.Is(err, utils.ErrHTTPSRequired) {
				respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed,
					httpsRequiredUserMsg,
					httpsRequiredLogMsg)
				return
			}
			respondWithAuthStatusError(c, http.StatusBadRequest, ErrCodeAuthCookieFailed, "Failed to set authentication cookies", "failed to set authentication cookies")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":              "Invite accepted",
			"user":                 out.Token.User,
			"accessTokenExpiresAt": out.Token.AccessTokenExpiresAt,
		})
	}
}
```

### GET /avatars/:userId

拒绝危险路径文本，检查账号存在，再选择上传文件或内置 WebP 默认图。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | avatar / handleServeAvatar |
| 身份 | 无需账号会话，仍受限流或专用凭据约束 |
| 路由参数与查询 | c.Param("userId") |
| 响应调用 | c.Status(http.StatusNotFound)；c.Status(http.StatusNotFound)；c.Status(http.StatusInternalServerError)；c.Data(http.StatusOK, "image/webp", data)；c.File(path)；c.Header("Cache-Control", "no-store")；c.Header("Pragma", "no-cache")；c.Header("Expires", time.Unix(0, 0).UTC().Format(http.TimeFormat)) |
| 源码 | backend/internal/wiki/avatar/routes.go |

请求与响应实现：

```go
func (r *Routes) handleServeAvatar(c *gin.Context) {
	userID := c.Param("userId")

	// 在调用服务前拒绝斜杠、反斜杠或双点，头像服务自身仍执行其标识检查。
	if strings.ContainsAny(userID, "/\\") || strings.Contains(userID, "..") {
		c.Status(http.StatusNotFound)
		return
	}

	if _, err := r.userService().GetUserByID(userID); err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	path, found := r.avatarService.AvatarPath(userID)
	if !found {
		data, err := brandassets.Read(brandassets.AvatarFile)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		disableClientCache(c)
		c.Data(http.StatusOK, "image/webp", data)
		return
	}

	disableClientCache(c)
	c.File(path)
}

// disableClientCache 禁止浏览器及中间缓存保留头像或状态，避免同名文件更新后显示旧内容。
func disableClientCache(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", time.Unix(0, 0).UTC().Format(http.TimeFormat))
}
```

### GET /api/user/avatar

按当前用户检查自定义文件，并禁止缓存这个随上传变化的状态。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | avatar / handleAvatarStatus |
| 身份 | 需要认证 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | c.JSON(http.StatusOK, gin.H{"hasCustomAvatar": found}) |
| 源码 | backend/internal/wiki/avatar/routes.go |

请求与响应实现：

```go
func (r *Routes) handleAvatarStatus(c *gin.Context) {
	user := authmw.MustGetUser(c)
	_, found := r.avatarService.AvatarPath(user.ID)
	disableClientCache(c)
	c.JSON(http.StatusOK, gin.H{"hasCustomAvatar": found})
}

// handleUploadAvatar 使用头像模块的大小限制解析 file 字段，再提交当前用户的文件。
// 解析超限返回 413，其他解析错误返回缺文件的 400；成功后无响应体并返回 200。
// 文件延迟关闭，当前实现不报告关闭错误。
```

### POST /api/user/avatar

使用头像模块的大小限制解析 file 字段，再提交当前用户的文件。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | avatar / handleUploadAvatar |
| 身份 | 需要认证 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithAvatarStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeAvatarUploadFailed, "File too large", "file too large")；respondWithAvatarStatusError(c, http.StatusBadRequest, ErrCodeAvatarUploadFailed, "Missing file", "missing file")；respondWithAvatarError(c, err)；c.Status(http.StatusOK) |
| 源码 | backend/internal/wiki/avatar/routes.go |

请求与响应实现：

```go
func (r *Routes) handleUploadAvatar(c *gin.Context) {
	user := authmw.MustGetUser(c)

	file, header, err := httpinternal.ParseUploadedFile(c, coreavatar.MaxUploadSize, "file")
	if err != nil {
		if errors.Is(err, httpinternal.ErrUploadTooLarge) {
			respondWithAvatarStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeAvatarUploadFailed, "File too large", "file too large")
		} else {
			respondWithAvatarStatusError(c, http.StatusBadRequest, ErrCodeAvatarUploadFailed, "Missing file", "missing file")
		}
		return
	}
	defer func() { _ = file.Close() }()

	if err := r.uploadAvatar.Execute(c.Request.Context(), UploadAvatarInput{
		UserID:   user.ID,
		File:     file,
		Filename: header.Filename,
	}); err != nil {
		respondWithAvatarError(c, err)
		return
	}
	c.Status(http.StatusOK)
}

// handleDeleteAvatar 只删除当前账号头像，用例成功后返回 200，默认图片由公开读取路径回退提供。
```

### DELETE /api/user/avatar

只删除当前账号头像，用例成功后返回 200，默认图片由公开读取路径回退提供。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | avatar / handleDeleteAvatar |
| 身份 | 需要认证 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithAvatarError(c, err)；c.Status(http.StatusOK) |
| 源码 | backend/internal/wiki/avatar/routes.go |

请求与响应实现：

```go
func (r *Routes) handleDeleteAvatar(c *gin.Context) {
	user := authmw.MustGetUser(c)

	if err := r.deleteAvatar.Execute(c.Request.Context(), DeleteAvatarInput{UserID: user.ID}); err != nil {
		respondWithAvatarError(c, err)
		return
	}
	c.Status(http.StatusOK)
}

// handleServeAvatar 拒绝危险路径文本，检查账号存在，再选择上传文件或内置 WebP 默认图。
// 任意账号查找错误都返回 404，内置图片读取失败返回 500，成功图片响应禁止缓存。
```

### GET /api/branding

返回服务提供的品牌配置和约束，错误按品牌领域规则编码。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | branding / handleGetBranding |
| 身份 | 无需账号会话，仍受限流或专用凭据约束 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithBrandingError(c, err)；c.JSON(http.StatusOK, out.Config) |
| 源码 | backend/internal/wiki/branding/routes.go |

请求与响应实现：

```go
func (r *Routes) handleGetBranding(c *gin.Context) {
	out, err := r.getBranding.Execute(c.Request.Context())
	if err != nil {
		respondWithBrandingError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Config)
}

// handleUpdateBranding 解析 siteName 后交给用例验证与保存，不在路由中使用必填绑定规则。
```

### GET /branding/:filename

读取约束并验证请求文件名，成功后禁止缓存并发送磁盘文件。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | branding / handleServeBrandingAsset |
| 身份 | 无需账号会话，仍受限流或专用凭据约束 |
| 路由参数与查询 | c.Param("filename"), cfg) |
| 响应调用 | c.Status(http.StatusInternalServerError)；c.Status(status)；c.File(cleanPath) |
| 源码 | backend/internal/wiki/branding/routes.go |

请求与响应实现：

```go
func (r *Routes) handleServeBrandingAsset(c *gin.Context) {
	cfg, err := r.brandingService.GetBranding()
	if err != nil {
		r.log.Error("failed to get branding constraints", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	cleanPath, status := r.resolveBrandingAssetPath(c.Param("filename"), cfg)
	if status != http.StatusOK {
		c.Status(status)
		return
	}

	disableClientCache(c)
	c.File(cleanPath)
}

// handleServeCurrentFavicon 优先读取配置中的图标，路径禁止或文件不存在时回退内置 ICO。
// 品牌读取或文件系统内部错误返回 500，不在这些分支用默认图掩盖错误。
```

### GET /favicon.ico

优先读取配置中的图标，路径禁止或文件不存在时回退内置 ICO。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | branding / handleServeCurrentFavicon |
| 身份 | 无需账号会话，仍受限流或专用凭据约束 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | c.Status(http.StatusInternalServerError)；c.File(cleanPath)；c.Status(status)；c.Status(http.StatusInternalServerError)；c.Data(http.StatusOK, "image/x-icon", data) |
| 源码 | backend/internal/wiki/branding/routes.go |

请求与响应实现：

```go
func (r *Routes) handleServeCurrentFavicon(c *gin.Context) {
	cfg, err := r.brandingService.GetBranding()
	if err != nil {
		r.log.Error("failed to get branding config", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	if cfg.FaviconFile != "" {
		cleanPath, status := r.resolveBrandingAssetPath(cfg.FaviconFile, cfg)
		if status == http.StatusOK {
			disableClientCache(c)
			c.File(cleanPath)
			return
		}
		if status == http.StatusInternalServerError {
			c.Status(status)
			return
		}
	}

	disableClientCache(c)
	data, err := brandassets.Read(brandassets.FaviconFile)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Data(http.StatusOK, "image/x-icon", data)
}

// resolveBrandingAssetPath 按安全名称、标识及图标扩展名并集、目录相对关系和文件状态逐步检查。
// 采用路径清理与 Stat，不解析符号链接目标，也不验证文件内容或确认目标一定为普通文件。
```

### PUT /api/branding

解析 siteName 后交给用例验证与保存，不在路由中使用必填绑定规则。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | branding / handleUpdateBranding |
| 身份 | 需要认证；管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithBrandingStatusError(c, http.StatusBadRequest, ErrCodeBrandingInvalidPayload, "Invalid payload", "invalid payload")；respondWithBrandingError(c, err)；c.JSON(http.StatusOK, out.Config) |
| 源码 | backend/internal/wiki/branding/routes.go |

请求与响应实现：

```go
func (r *Routes) handleUpdateBranding(c *gin.Context) {
	var req struct {
		SiteName string `json:"siteName"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithBrandingStatusError(c, http.StatusBadRequest, ErrCodeBrandingInvalidPayload, "Invalid payload", "invalid payload")
		return
	}
	out, err := r.updateBranding.Execute(c.Request.Context(), UpdateBrandingInput{SiteName: req.SiteName})
	if err != nil {
		respondWithBrandingError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Config)
}

// handleUploadLogo 先读取当前品牌约束，以标识文件大小限制解析上传，再调用保存用例。
// 超限返回 413，其他文件解析错误归为缺文件；文件关闭失败单独记录日志。
```

### POST /api/branding/logo

先读取当前品牌约束，以标识文件大小限制解析上传，再调用保存用例。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | branding / handleUploadLogo |
| 身份 | 需要认证；管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithBrandingError(c, err)；respondWithBrandingStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeBrandingLogoTooLarge, "File too large", "file too large")；respondWithBrandingStatusError(c, http.StatusBadRequest, ErrCodeBrandingLogoMissing, "Missing file", "missing file")；respondWithBrandingError(c, err)；c.JSON(http.StatusOK, gin.H{"path": out.Path, "branding": out.Config}) |
| 源码 | backend/internal/wiki/branding/routes.go |

请求与响应实现：

```go
func (r *Routes) handleUploadLogo(c *gin.Context) {
	constraints, err := r.brandingService.GetBranding()
	if err != nil {
		respondWithBrandingError(c, err)
		return
	}
	maxSize := constraints.BrandingConstraints.MaxLogoSize
	file, header, err := httpinternal.ParseUploadedFile(c, maxSize, "file")
	if err != nil {
		if errors.Is(err, httpinternal.ErrUploadTooLarge) {
			respondWithBrandingStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeBrandingLogoTooLarge, "File too large", "file too large")
		} else {
			respondWithBrandingStatusError(c, http.StatusBadRequest, ErrCodeBrandingLogoMissing, "Missing file", "missing file")
		}
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			r.log.Error("could not close logo file", "error", err)
		}
	}()
	out, err := r.uploadLogo.Execute(c.Request.Context(), UploadLogoInput{File: file, Filename: header.Filename})
	if err != nil {
		respondWithBrandingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"path": out.Path, "branding": out.Config})
}

// handleDeleteLogo 调用删除用例后返回当前品牌配置，不在该路由删除内置默认资源。
```

### POST /api/branding/favicon

使用图标专属大小限制，成功返回存储服务给出的路径及最新配置。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | branding / handleUploadFavicon |
| 身份 | 需要认证；管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithBrandingError(c, err)；respondWithBrandingStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeBrandingFaviconTooLarge, "File too large", "file too large")；respondWithBrandingStatusError(c, http.StatusBadRequest, ErrCodeBrandingFaviconMissing, "Missing file", "missing file")；respondWithBrandingError(c, err)；c.JSON(http.StatusOK, gin.H{"path": out.Path, "branding": out.Config}) |
| 源码 | backend/internal/wiki/branding/routes.go |

请求与响应实现：

```go
func (r *Routes) handleUploadFavicon(c *gin.Context) {
	constraints, err := r.brandingService.GetBranding()
	if err != nil {
		respondWithBrandingError(c, err)
		return
	}
	maxSize := constraints.BrandingConstraints.MaxFaviconSize
	file, header, err := httpinternal.ParseUploadedFile(c, maxSize, "file")
	if err != nil {
		if errors.Is(err, httpinternal.ErrUploadTooLarge) {
			respondWithBrandingStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeBrandingFaviconTooLarge, "File too large", "file too large")
		} else {
			respondWithBrandingStatusError(c, http.StatusBadRequest, ErrCodeBrandingFaviconMissing, "Missing file", "missing file")
		}
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			r.log.Error("could not close favicon file", "error", err)
		}
	}()
	out, err := r.uploadFavicon.Execute(c.Request.Context(), UploadFaviconInput{File: file, Filename: header.Filename})
	if err != nil {
		respondWithBrandingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"path": out.Path, "branding": out.Config})
}

// handleDeleteFavicon 删除用户图标并返回配置，后续当前图标读取可使用默认图片回退。
```

### DELETE /api/branding/logo

调用删除用例后返回当前品牌配置，不在该路由删除内置默认资源。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | branding / handleDeleteLogo |
| 身份 | 需要认证；管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithBrandingError(c, err)；c.JSON(http.StatusOK, gin.H{"branding": out.Config}) |
| 源码 | backend/internal/wiki/branding/routes.go |

请求与响应实现：

```go
func (r *Routes) handleDeleteLogo(c *gin.Context) {
	out, err := r.deleteLogo.Execute(c.Request.Context())
	if err != nil {
		respondWithBrandingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"branding": out.Config})
}

// handleUploadFavicon 使用图标专属大小限制，成功返回存储服务给出的路径及最新配置。
// 上传文件由处理器延迟关闭，错误分别经过上传或品牌领域映射。
```

### DELETE /api/branding/favicon

删除用户图标并返回配置，后续当前图标读取可使用默认图片回退。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | branding / handleDeleteFavicon |
| 身份 | 需要认证；管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithBrandingError(c, err)；c.JSON(http.StatusOK, gin.H{"branding": out.Config}) |
| 源码 | backend/internal/wiki/branding/routes.go |

请求与响应实现：

```go
func (r *Routes) handleDeleteFavicon(c *gin.Context) {
	out, err := r.deleteFavicon.Execute(c.Request.Context())
	if err != nil {
		respondWithBrandingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"branding": out.Config})
}

// handleServeBrandingAsset 读取约束并验证请求文件名，成功后禁止缓存并发送磁盘文件。
// 路径校验状态原样返回，不将所有文件问题统一当作不存在。
```

### GET /api/health

将用例汇总结果映射为 200 或 503，并始终返回各项检查状态。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | health / handleHealth |
| 身份 | 无需账号会话，仍受限流或专用凭据约束 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | c.JSON(code, gin.H{ |
| 源码 | backend/internal/wiki/health/routes.go |

请求与响应实现：

```go
func (r *Routes) handleHealth(c *gin.Context) {
	healthy, checks := r.health.Execute()

	status := "ok"
	code := http.StatusOK
	if !healthy {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	c.JSON(code, gin.H{
		"status": status,
		"checks": checks,
	})
}
```

### POST /api/import/plan

验证操作者后解析 ZIP 文件与目标基础路径，创建计划但不执行内容导入。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | importer / handleCreatePlan |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | c.FormFile("file")；c.PostForm("targetBasePath") |
| 响应调用 | respondWithImporterStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeImporterUploadTooLarge, "Upload exceeds maximum size limit of 500 MiB", "upload exceeds maximum size limit")；respondWithImporterStatusError(c, http.StatusBadRequest, ErrCodeImporterMissingFile, "Missing file", "missing file")；respondWithImporterStatusError(c, http.StatusBadRequest, ErrCodeImporterFileOpenFailed, "Failed to open uploaded file", "failed to open uploaded file")；respondWithImporterError(c, err)；c.JSON(http.StatusOK, out.Plan) |
| 源码 | backend/internal/wiki/importer/routes.go |

请求与响应实现：

```go
func (r *Routes) handleCreatePlan(c *gin.Context) {
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, importMaxUploadSize)
	if err := c.Request.ParseMultipartForm(importMaxUploadSize); err != nil {
		respondWithImporterStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeImporterUploadTooLarge, "Upload exceeds maximum size limit of 500 MiB", "upload exceeds maximum size limit")
		return
	}

	fh, err := c.FormFile("file")
	if err != nil {
		respondWithImporterStatusError(c, http.StatusBadRequest, ErrCodeImporterMissingFile, "Missing file", "missing file")
		return
	}
	file, err := fh.Open()
	if err != nil {
		respondWithImporterStatusError(c, http.StatusBadRequest, ErrCodeImporterFileOpenFailed, "Failed to open uploaded file", "failed to open uploaded file")
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			r.log.Error("could not close uploaded import file", "error", err)
		}
	}()

	targetBasePath := c.PostForm("targetBasePath")
	out, err := r.createPlan.Execute(c.Request.Context(), CreateImportPlanInput{
		File: file, TargetBasePath: targetBasePath,
	})
	if err != nil {
		logRejectedZipExtraction(r.log, err)
		respondWithImporterError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Plan)
}

// logRejectedZipExtraction 仅对三种解压限制错误记录底层原因，保留字节数量及调用链信息。
// 对外消息由用例采用固定模板，内部原因通过日志单独保存，其他错误不在这里记录。
func logRejectedZipExtraction(log *slog.Logger, err error) {
	loc, ok := sharederrors.AsLocalizedError(err)
	if !ok {
		return
	}
	switch loc.Code {
	case ErrCodeImporterZipEntryTooLarge, ErrCodeImporterZipExtractedTooLarge, ErrCodeImporterZipRatioTooHigh:
		log.Warn("rejected import upload: zip extraction limit exceeded", "code", loc.Code, "error", loc.Cause)
	}
}

// handleGetPlan 返回当前计划快照，无计划经用例转换为空成功结果而不是 404。
```

### GET /api/import/plan

返回当前计划快照，无计划经用例转换为空成功结果而不是 404。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | importer / handleGetPlan |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithImporterError(c, err)；c.JSON(http.StatusOK, out.Plan) |
| 源码 | backend/internal/wiki/importer/routes.go |

请求与响应实现：

```go
func (r *Routes) handleGetPlan(c *gin.Context) {
	out, err := r.getPlan.Execute(c.Request.Context())
	if err != nil {
		respondWithImporterError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Plan)
}

// handleExecute 将当前用户作为导入作者提交启动请求，不等待后台导入结束。
// 刚启动或状态仍运行中时返回 202，其余成功状态返回 200。
```

### POST /api/import/execute

将当前用户作为导入作者提交启动请求，不等待后台导入结束。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | importer / handleExecute |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithImporterError(c, err)；c.JSON(statusCode, out.State) |
| 源码 | backend/internal/wiki/importer/routes.go |

请求与响应实现：

```go
func (r *Routes) handleExecute(c *gin.Context) {
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}

	out, err := r.execute.Execute(c.Request.Context(), ExecuteImportInput{UserID: user.ID})
	if err != nil {
		respondWithImporterError(c, err)
		return
	}

	statusCode := http.StatusOK
	if out.Started || out.State.ExecutionStatus == coreimporter.ExecutionStatusRunning {
		statusCode = http.StatusAccepted
	}
	c.JSON(statusCode, out.State)
}

// handleClearPlan 对未退出的取消任务返回 202 与状态，清理完成后返回 200 和空值。
```

### DELETE /api/import/plan

对未退出的取消任务返回 202 与状态，清理完成后返回 200 和空值。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | importer / handleClearPlan |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithImporterError(c, err)；c.JSON(http.StatusAccepted, state)；c.JSON(http.StatusOK, nil) |
| 源码 | backend/internal/wiki/importer/routes.go |

请求与响应实现：

```go
func (r *Routes) handleClearPlan(c *gin.Context) {
	state, err := r.clearPlan.Execute(c.Request.Context())
	if err != nil {
		respondWithImporterError(c, err)
		return
	}
	if state != nil {
		// 非空状态表示取消标记已设置，执行尚未结束，前端继续轮询。
		c.JSON(http.StatusAccepted, state)
		return
	}
	c.JSON(http.StatusOK, nil)
}
```

### PUT /api/admin/settings/public-access

使用布尔指针区分缺少字段与明确关闭，缺失或非法 JSON 返回 400。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | instancesettings / handleSetPublicAccess |
| 身份 | 需要认证；管理员；Cookie 会话专用 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithStatusError(c, http.StatusBadRequest, ErrCodeInvalidPayload, "Invalid payload", "invalid payload")；respondWithError(c, err)；c.JSON(http.StatusOK, gin.H{"enabled": r.publicAccess.Enabled()}) |
| 源码 | backend/internal/wiki/instancesettings/routes.go |

请求与响应实现：

```go
func (r *Routes) handleSetPublicAccess(c *gin.Context) {
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		respondWithStatusError(c, http.StatusBadRequest, ErrCodeInvalidPayload, "Invalid payload", "invalid payload")
		return
	}

	if err := r.publicAccess.SetEnabled(*req.Enabled); err != nil {
		// 环境变量管理的实例由服务返回冲突代码，经领域错误映射返回 409。
		respondWithError(c, err)
		return
	}

	r.log.Info("public access mode changed",
		"enabled", r.publicAccess.Enabled(),
		"actor_id", actorID(c),
	)
	c.JSON(http.StatusOK, gin.H{"enabled": r.publicAccess.Enabled()})
}

// handleSetTocDisplay 优先采用非空 mode；仅未提供模式时将 alwaysShow 转为 always 或 auto。
// 两种表示都未提供则拒绝，模式合法性由目录服务校验，响应返回服务实际保存的两种表示。
```

### PUT /api/admin/settings/toc-display

优先采用非空 mode；仅未提供模式时将 alwaysShow 转为 always 或 auto。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | instancesettings / handleSetTocDisplay |
| 身份 | 需要认证；管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithStatusError(c, http.StatusBadRequest, ErrCodeInvalidPayload, "Invalid payload", "invalid payload")；respondWithError(c, err)；c.JSON(http.StatusOK, gin.H{"alwaysShow": r.tocDisplay.AlwaysShow(), "mode": r.tocDisplay.Mode()}) |
| 源码 | backend/internal/wiki/instancesettings/routes.go |

请求与响应实现：

```go
func (r *Routes) handleSetTocDisplay(c *gin.Context) {
	var req struct {
		AlwaysShow *bool  `json:"alwaysShow"`
		Mode       string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.AlwaysShow == nil && req.Mode == "") {
		respondWithStatusError(c, http.StatusBadRequest, ErrCodeInvalidPayload, "Invalid payload", "invalid payload")
		return
	}

	mode := req.Mode
	if mode == "" {
		mode = "auto"
		if *req.AlwaysShow {
			mode = "always"
		}
	}
	if err := r.tocDisplay.SetMode(mode); err != nil {
		respondWithError(c, err)
		return
	}

	r.log.Info("toc always-show setting changed",
		"always_show", r.tocDisplay.AlwaysShow(),
		"actor_id", actorID(c),
	)
	c.JSON(http.StatusOK, gin.H{"alwaysShow": r.tocDisplay.AlwaysShow(), "mode": r.tocDisplay.Mode()})
}

// actorID 从已有鉴权上下文取用户标识，无法取得用户时返回空文本用于日志字段。
func actorID(c *gin.Context) string {
	if u := authmw.TryGetUser(c); u != nil {
		return u.ID
	}
	return ""
}
```

### GET /api/pages/:id/links

从路由取页面标识，成功响应直接编码链接服务状态结果。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | links / handleGetLinkStatus |
| 身份 | 公开读取按实例设置检查 |
| 路由参数与查询 | c.Param("id") |
| 响应调用 | respondWithLinkError(c, err)；c.JSON(http.StatusOK, out.Status) |
| 源码 | backend/internal/wiki/links/routes.go |

请求与响应实现：

```go
func (r *Routes) handleGetLinkStatus(c *gin.Context) {
	pageID := c.Param("id")
	out, err := r.getLinkStatus.Execute(c.Request.Context(), GetLinkStatusInput{PageID: pageID})
	if err != nil {
		respondWithLinkError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Status)
}

// handleGetBrokenLinks 返回带 links 字段的用例结果，错误统一通过链接领域响应映射。
```

### GET /api/links/broken

返回带 links 字段的用例结果，错误统一通过链接领域响应映射。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | links / handleGetBrokenLinks |
| 身份 | 需要认证；管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithLinkError(c, err)；c.JSON(http.StatusOK, out) |
| 源码 | backend/internal/wiki/links/routes.go |

请求与响应实现：

```go
func (r *Routes) handleGetBrokenLinks(c *gin.Context) {
	out, err := r.getBrokenLinks.Execute(c.Request.Context())
	if err != nil {
		respondWithLinkError(c, err)
		return
	}

	c.JSON(http.StatusOK, out)
}
```

### GET /api/tree

由 pages 模块处理，输入与响应见下表

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleGetTree |
| 身份 | 公开读取按实例设置检查 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | 通过对应业务输出或下载处理返回，参见下面实现 |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go

```

### GET /api/pages/:id

由 pages 模块处理，输入与响应见下表

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleGetPage |
| 身份 | 公开读取按实例设置检查 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | 通过对应业务输出或下载处理返回，参见下面实现 |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go

```

### GET /api/pages/lookup

将允许为空的路径交给逐段查询，返回存在与缺失段的解析结果。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleLookupPath |
| 身份 | 公开读取按实例设置检查 |
| 路由参数与查询 | c.Query("path")) |
| 响应调用 | respondWithPageError(c, err)；c.JSON(http.StatusOK, lookup) |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handleLookupPath(c *gin.Context) {
	path := strings.TrimSpace(c.Query("path"))
	lookup, err := r.queries.LookupPath(path)
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, lookup)
}

// handleResolvePermalink 从路由读取稳定页面标识，返回解析目标而不在此发起跳转。
```

### GET /api/pages/by-path

要求非空路径，分区响应包含一层子节点，普通页面深度为零。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleGetByPath |
| 身份 | 公开读取按实例设置检查 |
| 路由参数与查询 | c.Query("path")) |
| 响应调用 | respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageMissingPath, "Missing path", "missing path")；respondWithPageError(c, err)；r.respondPageWithDepth(c, http.StatusOK, page, depth) |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handleGetByPath(c *gin.Context) {
	routePath := strings.TrimSpace(c.Query("path"))
	if routePath == "" {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageMissingPath, "Missing path", "missing path")
		return
	}
	page, err := r.queries.FindByPath(routePath)
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	depth := 0
	if page.Kind == tree.NodeKindSection {
		depth = 1
	}
	r.respondPageWithDepth(c, http.StatusOK, page, depth)
}

// handleFindByTitle 去除查询两端空白后按标题查找，空值返回明确的缺少标题错误。
```

### GET /api/pages/by-title

去除查询两端空白后按标题查找，空值返回明确的缺少标题错误。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleFindByTitle |
| 身份 | 公开读取按实例设置检查 |
| 路由参数与查询 | c.Query("title")) |
| 响应调用 | respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageMissingTitle, "Missing title query parameter", "missing title")；c.JSON(http.StatusOK, out) |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handleFindByTitle(c *gin.Context) {
	title := strings.TrimSpace(c.Query("title"))
	if title == "" {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageMissingTitle, "Missing title query parameter", "missing title")
		return
	}
	out := r.queries.FindByTitle(title)
	c.JSON(http.StatusOK, out)
}

// handleLookupPath 将允许为空的路径交给逐段查询，返回存在与缺失段的解析结果。
```

### GET /api/pages/permalink/:id

从路由读取稳定页面标识，返回解析目标而不在此发起跳转。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleResolvePermalink |
| 身份 | 公开读取按实例设置检查 |
| 路由参数与查询 | c.Param("id")) |
| 响应调用 | respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageMissingID, "Page ID is required", "page id is required")；respondWithPageError(c, err)；c.JSON(http.StatusOK, target) |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handleResolvePermalink(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageMissingID, "Page ID is required", "page id is required")
		return
	}
	target, err := r.queries.ResolvePermalink(id)
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, target)
}

// handleSuggestSlug 拒绝空白标题，其余标题交给建议用例，包括纯标点标题。
// 建议为空仍以成功 JSON 返回，由输入校验流程处理最终创建名称。
```

### GET /api/pages/slug-suggestion

拒绝空白标题，其余标题交给建议用例，包括纯标点标题。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleSuggestSlug |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | c.Query("title"))；c.Query("parentId")),；c.Query("currentId")), |
| 响应调用 | respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageMissingTitle, "Title query param is required", "title query param is required")；respondWithPageError(c, err)；c.JSON(http.StatusOK, gin.H{"slug": out.Slug}) |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handleSuggestSlug(c *gin.Context) {
	title := strings.TrimSpace(c.Query("title"))
	if title == "" {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageMissingTitle, "Title query param is required", "title query param is required")
		return
	}
	out, err := r.suggestSlug.Execute(c.Request.Context(), SuggestSlugInput{
		ParentID:  strings.TrimSpace(c.Query("parentId")),
		CurrentID: strings.TrimSpace(c.Query("currentId")),
		Title:     title,
	})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"slug": out.Slug})
}

// handleCreate 绑定创建 JSON，取得认证用户后转换可选类型并调用创建用例。
// 未识别类型沿用 kindFromString 的页面默认值，成功响应使用 201。
```

### POST /api/pages

绑定创建 JSON，取得认证用户后转换可选类型并调用创建用例。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleCreate |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)；respondWithPageError(c, err)；r.respondPage(c, http.StatusCreated, out.Page) |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handleCreate(c *gin.Context) {
	var req struct {
		ParentID *string `json:"parentId"`
		Title    string  `json:"title" binding:"required"`
		Slug     string  `json:"slug" binding:"required"`
		Kind     *string `json:"kind"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	kind := kindFromString(req.Kind)
	out, err := r.createPage.Execute(c.Request.Context(), CreatePageInput{
		UserID: user.ID, ParentID: req.ParentID, Title: req.Title, Slug: req.Slug, Kind: &kind,
	})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	r.respondPage(c, http.StatusCreated, out.Page)
}

// handleUpdate 要求版本、标题和路径段，先校验结构化标签属性再进入保存用例。
// Content 指针区分省略正文与明确空正文，标签也保留 nil 与空列表的差别。
```

### PUT /api/pages/:id

要求版本、标题和路径段，先校验结构化标签属性再进入保存用例。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleUpdate |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | c.Param("id")) |
| 响应调用 | respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)；respondWithPageError(c, err)；respondWithPageError(c, err)；r.respondPage(c, http.StatusOK, out.Page) |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handleUpdate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req struct {
		Version    string            `json:"version" binding:"required"`
		Title      string            `json:"title" binding:"required"`
		Slug       string            `json:"slug" binding:"required"`
		Content    *string           `json:"content"`
		Tags       []string          `json:"tags"`
		Properties map[string]string `json:"properties"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	if err := validatePageMetadataInput(req.Tags, req.Properties); err != nil {
		respondWithPageError(c, err)
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}

	// 校验通过的标签再转换为小写，保存正文元信息与派生索引共用相同文字。
	// 未提供字段时保留 nil，明确空列表则用于清空标签。
	normalizedTags := req.Tags
	if req.Tags != nil {
		normalizedTags = normalizeTagInputs(req.Tags)
	}

	kind := tree.NodeKindPage
	out, err := r.updatePage.Execute(c.Request.Context(), UpdatePageInput{
		UserID: user.ID, ID: id, Version: req.Version, Title: req.Title, Slug: req.Slug,
		Content: req.Content, Kind: &kind,
		Tags: normalizedTags, Properties: req.Properties,
	})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	r.respondPage(c, http.StatusOK, out.Page)
}

// handleDelete 从查询串读取版本，recursive 仅在文字精确为 true 时启用。
// 用户身份取认证上下文，成功返回删除确认 JSON，不返回被删除的正文。
```

### DELETE /api/pages/:id

从查询串读取版本，recursive 仅在文字精确为 true 时启用。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleDelete |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | c.Param("id"))；c.DefaultQuery("recursive", "false") == "true"；c.Query("version") |
| 响应调用 | respondWithPageError(c, err)；c.JSON(http.StatusOK, gin.H{"message": "Page deleted"}) |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handleDelete(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	recursive := c.DefaultQuery("recursive", "false") == "true"
	version := c.Query("version")
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	if err := r.deletePage.Execute(c.Request.Context(), DeletePageInput{
		UserID: user.ID, ID: id, Version: version, Recursive: recursive,
	}); err != nil {
		respondWithPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Page deleted"})
}

// handleMove 要求版本字段，目标父标识和位置允许省略，直接传给移动用例。
```

### PUT /api/pages/:id/move

要求版本字段，目标父标识和位置允许省略，直接传给移动用例。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleMove |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | c.Param("id")) |
| 响应调用 | respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidPayload, "Invalid payload", "invalid payload")；respondWithPageError(c, err)；c.JSON(http.StatusOK, gin.H{"message": "Page moved"}) |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handleMove(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req struct {
		Version  string `json:"version" binding:"required"`
		ParentID string `json:"parentId"`
		Position *int   `json:"position"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidPayload, "Invalid payload", "invalid payload")
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	if err := r.movePage.Execute(c.Request.Context(), MovePageInput{
		UserID: user.ID, ID: id, Version: req.Version, ParentID: req.ParentID, Position: req.Position,
	}); err != nil {
		respondWithPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Page moved"})
}

// handleSort 将路由标识作为父节点，并提交完整子标识排列供树服务校验。
// JSON 绑定成功不代表列表有效，重复、遗漏和外部标识由下层拒绝。
```

### PUT /api/pages/:id/sort

将路由标识作为父节点，并提交完整子标识排列供树服务校验。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleSort |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | c.Param("id")) |
| 响应调用 | respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)；respondWithPageError(c, err)；c.JSON(http.StatusOK, gin.H{"message": "Pages sorted successfully"}) |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handleSort(c *gin.Context) {
	parentID := strings.TrimSpace(c.Param("id"))
	var req struct {
		OrderedIDs []string `json:"orderedIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	if err := r.sortPages.Execute(c.Request.Context(), SortPagesInput{
		ParentID: parentID, OrderedIDs: req.OrderedIDs,
	}); err != nil {
		respondWithPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pages sorted successfully"})
}

// handlePin 要求版本，省略 pinned 时使用布尔零值 false，即提交取消置顶。
// 成功返回组装后的页面响应，不单独读取认证用户字段。
```

### PUT /api/pages/:id/pin

要求版本，省略 pinned 时使用布尔零值 false，即提交取消置顶。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handlePin |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | c.Param("id")) |
| 响应调用 | respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)；respondWithPageError(c, err)；r.respondPage(c, http.StatusOK, out.Page) |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handlePin(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req struct {
		Version string `json:"version" binding:"required"`
		Pinned  bool   `json:"pinned"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	out, err := r.pinPage.Execute(c.Request.Context(), PinPageInput{
		ID:      id,
		Version: req.Version,
		Pinned:  req.Pinned,
	})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	r.respondPage(c, http.StatusOK, out.Page)
}

// handleAddFavorite 只为当前认证用户添加收藏，不从请求接收其他用户标识。
```

### PUT /api/pages/:id/favorite

只为当前认证用户添加收藏，不从请求接收其他用户标识。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleAddFavorite |
| 身份 | 需要认证 |
| 路由参数与查询 | c.Param("id")) |
| 响应调用 | respondWithPageError(c, err)；c.JSON(http.StatusOK, gin.H{"message": "Page favorited"}) |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handleAddFavorite(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	if err := r.addFavorite.Execute(c.Request.Context(), AddFavoriteInput{
		UserID: user.ID, PageID: id,
	}); err != nil {
		respondWithPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Page favorited"})
}

// handleListFavorites 读取当前用户收藏后转换为 API 页面，并从原始元信息填入标签属性。
// 单页元信息读取失败沿用 enrichPageMetadata 的空列表回退，不使整个响应失败。
```

### DELETE /api/pages/:id/favorite

由 pages 模块处理，输入与响应见下表

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleRemoveFavorite |
| 身份 | 需要认证 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | 通过对应业务输出或下载处理返回，参见下面实现 |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go

```

### GET /api/favorites

读取当前用户收藏后转换为 API 页面，并从原始元信息填入标签属性。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleListFavorites |
| 身份 | 需要认证 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithPageError(c, err)；c.JSON(http.StatusOK, gin.H{"pages": apiPages}) |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handleListFavorites(c *gin.Context) {
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	out, err := r.listFavorites.Execute(c.Request.Context(), ListFavoritesInput{UserID: user.ID})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	apiPages := make([]*dto.Page, 0, len(out.Pages))
	for _, p := range out.Pages {
		apiPage := dto.ToAPIPage(p, r.userResolver)
		r.enrichPageMetadata(apiPage)
		apiPages = append(apiPages, apiPage)
	}
	c.JSON(http.StatusOK, gin.H{"pages": apiPages})
}

// handleEnsurePath 要求路径和标题，类型由可选字符串转换，成功返回路径最终页面。
```

### POST /api/pages/ensure

要求路径和标题，类型由可选字符串转换，成功返回路径最终页面。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleEnsurePath |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)；respondWithPageError(c, err)；r.respondPage(c, http.StatusOK, out.Page) |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handleEnsurePath(c *gin.Context) {
	var req struct {
		Path  string  `json:"path" binding:"required"`
		Title string  `json:"title" binding:"required"`
		Kind  *string `json:"kind"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	kind := kindFromString(req.Kind)
	out, err := r.ensurePath.Execute(c.Request.Context(), EnsurePathInput{
		UserID: user.ID, TargetPath: req.Path, TargetTitle: req.Title, Kind: &kind,
	})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	r.respondPage(c, http.StatusOK, out.Page)
}

// handleConvert 要求版本和目标类型，并显式限制类型只能为 page 或 section。
// 转换完成返回 204，没有页面 JSON 正文。
```

### POST /api/pages/convert/:id

要求版本和目标类型，并显式限制类型只能为 page 或 section。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleConvert |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | c.Param("id")) |
| 响应调用 | respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)；respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidTargetKind, "Invalid targetKind", "invalid target kind")；respondWithPageError(c, err)；c.Status(http.StatusNoContent) |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handleConvert(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req struct {
		Kind    string `json:"targetKind" binding:"required"`
		Version string `json:"version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	if req.Kind != "page" && req.Kind != "section" {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidTargetKind, "Invalid targetKind", "invalid target kind")
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	if err := r.convertPage.Execute(c.Request.Context(), ConvertPageInput{
		UserID: user.ID, ID: id, Version: req.Version, TargetKind: tree.NodeKind(req.Kind),
	}); err != nil {
		respondWithPageError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// handleCopy 从路由取得来源页，目标父标识、标题和路径段来自 JSON。
// 成功使用 201 返回复制得到的页面，来源内容读取和资源处理由复制用例完成。
```

### POST /api/pages/copy/:id

从路由取得来源页，目标父标识、标题和路径段来自 JSON。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleCopy |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | c.Param("id")) |
| 响应调用 | respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)；respondWithPageError(c, err)；r.respondPage(c, http.StatusCreated, out.Page) |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handleCopy(c *gin.Context) {
	sourceID := strings.TrimSpace(c.Param("id"))
	var req struct {
		ParentID *string `json:"targetParentId"`
		Title    string  `json:"title" binding:"required"`
		Slug     string  `json:"slug" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	out, err := r.copyPage.Execute(c.Request.Context(), CopyPageInput{
		UserID: user.ID, SourcePageID: sourceID, TargetParentID: req.ParentID,
		Title: req.Title, Slug: req.Slug,
	})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	r.respondPage(c, http.StatusCreated, out.Page)
}

// handleRefactorPreview 绑定重构方案输入后返回预览，不在处理器中应用正文写入。
// 预览不接收版本字段，具体可执行条件由预览用例检查。
```

### POST /api/pages/:id/refactor/preview

绑定重构方案输入后返回预览，不在处理器中应用正文写入。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleRefactorPreview |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | c.Param("id")) |
| 响应调用 | respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)；respondWithPageError(c, err)；c.JSON(http.StatusOK, out) |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handleRefactorPreview(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req struct {
		Kind        string  `json:"kind" binding:"required"`
		Title       string  `json:"title"`
		Slug        string  `json:"slug"`
		Content     *string `json:"content"`
		NewParentID *string `json:"parentId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	out, err := r.previewRefactor.Execute(c.Request.Context(), RefactorPreviewInput{
		PageID: id, Kind: req.Kind, Title: req.Title, Slug: req.Slug,
		Content: req.Content, NewParentID: req.NewParentID,
	})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// handleRefactorApply 接收版本、操作类型、链接重写选择和可选位置，取得用户后执行。
// 输入 JSON 只负责字段绑定，具体方案计算和保存由应用用例处理。
```

### POST /api/pages/:id/refactor/apply

接收版本、操作类型、链接重写选择和可选位置，取得用户后执行。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | pages / handleRefactorApply |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | c.Param("id")) |
| 响应调用 | respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)；respondWithPageError(c, err)；r.respondPage(c, http.StatusOK, page)；// respondPage 将领域页面转换为 API 格式，解析作者信息并填充标签属性后按指定状态响应。 |
| 源码 | backend/internal/wiki/pages/routes.go |

请求与响应实现：

```go
func (r *Routes) handleRefactorApply(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req struct {
		Version      string  `json:"version" binding:"required"`
		Kind         string  `json:"kind" binding:"required"`
		Title        string  `json:"title"`
		Slug         string  `json:"slug"`
		Content      *string `json:"content"`
		NewParentID  *string `json:"parentId"`
		RewriteLinks bool    `json:"rewriteLinks"`
		Position     *int    `json:"position"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithPageStatusError(c, http.StatusBadRequest, ErrCodePageInvalidRequest, errInvalidRequestUserMsg, errInvalidRequestLogMsg)
		return
	}
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	page, err := r.applyRefactor.Execute(c.Request.Context(), RefactorApplyInput{
		Version: req.Version,
		UserID:  user.ID,
		RefactorPreviewInput: RefactorPreviewInput{
			PageID: id, Kind: req.Kind, Title: req.Title, Slug: req.Slug,
			Content: req.Content, NewParentID: req.NewParentID,
		},
		RewriteLinks: req.RewriteLinks,
		Position:     req.Position,
	})
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	r.respondPage(c, http.StatusOK, page)
}

// respondPage 将领域页面转换为 API 格式，解析作者信息并填充标签属性后按指定状态响应。
```

### GET /api/properties

解析过滤文字和条数，条数不能转为整数时返回 400。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | properties / handleGetPropertyKeys |
| 身份 | 公开读取按实例设置检查 |
| 路由参数与查询 | c.DefaultQuery("q", "")；c.DefaultQuery("limit", "50") |
| 响应调用 | respondWithPropertiesBadRequest(c, ErrCodePropertiesInvalidLimit, "Invalid limit value", "invalid limit value")；respondWithPropertiesError(c, err)；c.JSON(http.StatusOK, out.Keys) |
| 源码 | backend/internal/wiki/properties/routes.go |

请求与响应实现：

```go
func (r *Routes) handleGetPropertyKeys(c *gin.Context) {
	filter := c.DefaultQuery("q", "")
	limitStr := c.DefaultQuery("limit", "50")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		respondWithPropertiesBadRequest(c, ErrCodePropertiesInvalidLimit, "Invalid limit value", "invalid limit value")
		return
	}

	out, err := r.getPropertyKeys.Execute(c.Request.Context(), GetPropertyKeysInput{Filter: filter, Limit: limit})
	if err != nil {
		respondWithPropertiesError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Keys)
}

// handleGetPagesByProperty 原样读取 key 与 value，必填检查及索引查询由用例执行。
```

### GET /api/properties/pages

原样读取 key 与 value，必填检查及索引查询由用例执行。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | properties / handleGetPagesByProperty |
| 身份 | 公开读取按实例设置检查 |
| 路由参数与查询 | c.Query("key")；c.Query("value") |
| 响应调用 | respondWithPropertiesError(c, err)；c.JSON(http.StatusOK, out.Pages) |
| 源码 | backend/internal/wiki/properties/routes.go |

请求与响应实现：

```go
func (r *Routes) handleGetPagesByProperty(c *gin.Context) {
	key := c.Query("key")
	value := c.Query("value")

	out, err := r.getPagesByProperty.Execute(c.Request.Context(), GetPagesByPropertyInput{Key: key, Value: value})
	if err != nil {
		respondWithPropertiesError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Pages)
}
```

### GET /api/restore/progress

从 X-Restore-Token 请求头取得凭据，校验后仅返回当前任务状态。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | restore / handleProgress |
| 身份 | 无需账号会话，仍受限流或专用凭据约束 |
| 路由参数与查询 | c.GetHeader("X-Restore-Token")) |
| 响应调用 | c.Header("Cache-Control", "no-store")；respondWithRestoreStatusError(c, http.StatusUnauthorized, "restore_progress_unauthorized", "Invalid restore progress credential", "invalid restore progress credential")；c.JSON(http.StatusOK, status) |
| 源码 | backend/internal/wiki/restore/routes.go |

请求与响应实现：

```go
func (r *Routes) handleProgress(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}
	status, valid := r.manager.ProgressStatus(c.GetHeader("X-Restore-Token"))
	if !valid {
		respondWithRestoreStatusError(c, http.StatusUnauthorized, "restore_progress_unauthorized", "Invalid restore progress credential", "invalid restore progress credential")
		return
	}
	c.JSON(http.StatusOK, status)
}
```

### POST /api/admin/restore/upload

接收 file 表单项，同步解析上传并将文件暂存到数据目录。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | restore / handleTriggerUpload |
| 身份 | 需要认证；管理员；Cookie 会话专用 |
| 路由参数与查询 | c.FormFile("file") |
| 响应调用 | respondWithRestoreStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeRestoreUploadTooLarge, "Upload exceeds the maximum allowed size", "upload exceeds the maximum allowed size")；respondWithRestoreStatusError(c, http.StatusBadRequest, ErrCodeRestoreUploadInvalid, "Failed to parse the uploaded file", "failed to parse the uploaded file")；respondWithRestoreStatusError(c, http.StatusBadRequest, ErrCodeRestoreMissingFile, "Missing file", "missing file")；respondWithRestoreStatusError(c, http.StatusBadRequest, ErrCodeRestoreFileOpenFailed, "Failed to open uploaded file", "failed to open uploaded file")；respondWithRestoreError(c, err)；c.Header("Cache-Control", "no-store")；c.JSON(http.StatusAccepted, gin.H{"ok": true, "statusToken": statusToken}) |
| 源码 | backend/internal/wiki/restore/routes.go |

请求与响应实现：

```go
func (r *Routes) handleTriggerUpload(c *gin.Context) {
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}

	// 限额覆盖整个请求体，包括表单边界和字段，并非仅限制 ZIP 文件字节。
	maxSize := r.manager.MaxUploadSizeBytes()
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
	if err := c.Request.ParseMultipartForm(uploadParseMaxMemory); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			respondWithRestoreStatusError(c, http.StatusRequestEntityTooLarge, ErrCodeRestoreUploadTooLarge, "Upload exceeds the maximum allowed size", "upload exceeds the maximum allowed size")
			return
		}
		respondWithRestoreStatusError(c, http.StatusBadRequest, ErrCodeRestoreUploadInvalid, "Failed to parse the uploaded file", "failed to parse the uploaded file")
		return
	}
	// 解析成功后登记清理表单临时文件，后续任一返回分支都会执行。
	defer func() {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
	}()

	fh, err := c.FormFile("file")
	if err != nil {
		respondWithRestoreStatusError(c, http.StatusBadRequest, ErrCodeRestoreMissingFile, "Missing file", "missing file")
		return
	}
	file, err := fh.Open()
	if err != nil {
		respondWithRestoreStatusError(c, http.StatusBadRequest, ErrCodeRestoreFileOpenFailed, "Failed to open uploaded file", "failed to open uploaded file")
		return
	}
	// 管理器在返回前读完上传流，因此处理器返回时关闭句柄不影响后台恢复。
	defer func() { _ = file.Close() }()

	statusToken, err := r.manager.TriggerRestoreFromUploadWithStatusToken(file)
	if err != nil {
		respondWithRestoreError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusAccepted, gin.H{"ok": true, "statusToken": statusToken})
}

// handleStatus 为管理员返回当前恢复快照，执行失败通过状态字段表示。
```

### POST /api/admin/restore/:id

将路径中的快照标识交给管理器启动任务，返回 202 与专用进度凭据。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | restore / handleTrigger |
| 身份 | 需要认证；管理员；Cookie 会话专用 |
| 路由参数与查询 | c.Param("id") |
| 响应调用 | respondWithRestoreError(c, err)；c.Header("Cache-Control", "no-store")；c.JSON(http.StatusAccepted, gin.H{"ok": true, "statusToken": statusToken}) |
| 源码 | backend/internal/wiki/restore/routes.go |

请求与响应实现：

```go
func (r *Routes) handleTrigger(c *gin.Context) {
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}
	id := c.Param("id")
	statusToken, err := r.manager.TriggerRestoreWithStatusToken(id)
	if err != nil {
		respondWithRestoreError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusAccepted, gin.H{"ok": true, "statusToken": statusToken})
}

// handleTriggerUpload 接收 file 表单项，同步解析上传并将文件暂存到数据目录。
// 暂存成功后由管理器后台校验归档与执行恢复，返回 202 和任务进度凭据。
// 表单解析、打开文件与暂存失败直接响应错误，归档内容失败通过任务状态报告。
```

### GET /api/admin/restore/status

为管理员返回当前恢复快照，执行失败通过状态字段表示。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | restore / handleStatus |
| 身份 | 需要认证；管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | c.JSON(http.StatusOK, r.manager.Status()) |
| 源码 | backend/internal/wiki/restore/routes.go |

请求与响应实现：

```go
func (r *Routes) handleStatus(c *gin.Context) {
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}
	c.JSON(http.StatusOK, r.manager.Status())
}

// handleSelfRestart 只在任务标记需要介入时调用平台重启实现，其余状态响应 409。
// 成功重启会替换或退出当前进程，不走后续响应分支；调用返回时记录失败原因并响应 500。
```

### POST /api/admin/restore/self-restart

只在任务标记需要介入时调用平台重启实现，其余状态响应 409。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | restore / handleSelfRestart |
| 身份 | 需要认证；管理员；Cookie 会话专用 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithRestoreStatusError(c, http.StatusConflict, ErrCodeRestoreNotIntervenable,；respondWithRestoreStatusError(c, http.StatusInternalServerError, ErrCodeRestoreInternalError, "Failed to restart server", "failed to restart server") |
| 源码 | backend/internal/wiki/restore/routes.go |

请求与响应实现：

```go
func (r *Routes) handleSelfRestart(c *gin.Context) {
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}
	if !r.manager.Status().NeedsIntervention {
		respondWithRestoreStatusError(c, http.StatusConflict, ErrCodeRestoreNotIntervenable,
			"Self-restart is only available after a restore reports it needs intervention",
			"self-restart is only available after a restore reports it needs intervention")
		return
	}
	// 平台实现成功时不返回，返回的错误留在服务日志中供排查。
	err := r.manager.SelfRestart()
	slog.Default().Error("restore: self-restart failed", "error", err)
	respondWithRestoreStatusError(c, http.StatusInternalServerError, ErrCodeRestoreInternalError, "Failed to restart server", "failed to restart server")
}

// handleProgress 从 X-Restore-Token 请求头取得凭据，校验后仅返回当前任务状态。
// 无效或过期凭据响应 401，所有响应禁止缓存，凭据不放入 URL。
```

### POST /api/admin/resync

在启动回调返回后响应 202 和 ok 标记，不等待后台重建结束。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | resync / handleTriggerResync |
| 身份 | 需要认证；管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithResyncError(c, err)；c.JSON(http.StatusAccepted, gin.H{"ok": true}) |
| 源码 | backend/internal/wiki/resync/routes.go |

请求与响应实现：

```go
func (r *Routes) handleTriggerResync(c *gin.Context) {
	if err := r.triggerUC.Execute(c.Request.Context()); err != nil {
		respondWithResyncError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

// handleResyncStatus 返回当前任务快照，成功、失败和执行中均通过字段表示并响应 200。
```

### GET /api/admin/resync/status

返回当前任务快照，成功、失败和执行中均通过字段表示并响应 200。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | resync / handleResyncStatus |
| 身份 | 需要认证；管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | c.JSON(http.StatusOK, out.Status) |
| 源码 | backend/internal/wiki/resync/routes.go |

请求与响应实现：

```go
func (r *Routes) handleResyncStatus(c *gin.Context) {
	out := r.statusUC.Execute(c.Request.Context())
	c.JSON(http.StatusOK, out.Status)
}
```

### GET /api/pages/:id/revisions

去除页面、游标与条数字符串的边缘空白，条数默认五十且限定一至二百。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | revisions / handleListRevisions |
| 身份 | 需要认证 |
| 路由参数与查询 | c.Param("id"))；c.Query("limit")); raw != "" {；c.Query("cursor")), |
| 响应调用 | respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidPageID, errPageIDRequiredUserMsg, errPageIDRequiredLogMsg)；respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidLimit, "Revision list limit is invalid", "revision list limit for page %s is invalid", pageID)；respondWithRevisionError(c, err)；c.JSON(http.StatusOK, gin.H{ |
| 源码 | backend/internal/wiki/revisions/routes.go |

请求与响应实现：

```go
func (r *Routes) handleListRevisions(c *gin.Context) {
	pageID := strings.TrimSpace(c.Param("id"))
	if pageID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidPageID, errPageIDRequiredUserMsg, errPageIDRequiredLogMsg)
		return
	}

	limit := 50
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 || parsed > 200 {
			respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidLimit, "Revision list limit is invalid", "revision list limit for page %s is invalid", pageID)
			return
		}
		limit = parsed
	}

	out, err := r.listRevisions.Execute(c.Request.Context(), ListRevisionsInput{
		PageID: pageID,
		Cursor: strings.TrimSpace(c.Query("cursor")),
		Limit:  limit,
	})
	if err != nil {
		respondWithRevisionError(c, err)
		return
	}

	result := make([]*RevisionResponse, 0, len(out.Revisions))
	for _, rev := range out.Revisions {
		result = append(result, toRevisionResponse(rev, r.userResolver))
	}
	c.JSON(http.StatusOK, gin.H{
		"revisions":  result,
		"nextCursor": out.NextCursor,
	})
}

// handleGetRevision 要求页面与版本标识，快照或其版本记录为空时直接返回 404。
```

### GET /api/pages/:id/revisions/latest

读取单页最新版本，服务返回空记录时作为版本不存在处理。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | revisions / handleGetLatestRevision |
| 身份 | 需要认证 |
| 路由参数与查询 | c.Param("id")) |
| 响应调用 | respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidPageID, errPageIDRequiredUserMsg, errPageIDRequiredLogMsg)；respondWithRevisionError(c, err)；respondWithRevisionStatusError(c, http.StatusNotFound, ErrCodeRevisionNotFound, errRevisionNotFoundUserMsg, "revision for page %s not found", pageID)；c.JSON(http.StatusOK, toRevisionResponse(out.Revision, r.userResolver)) |
| 源码 | backend/internal/wiki/revisions/routes.go |

请求与响应实现：

```go
func (r *Routes) handleGetLatestRevision(c *gin.Context) {
	pageID := strings.TrimSpace(c.Param("id"))
	if pageID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidPageID, errPageIDRequiredUserMsg, errPageIDRequiredLogMsg)
		return
	}

	out, err := r.getLatest.Execute(c.Request.Context(), GetLatestRevisionInput{PageID: pageID})
	if err != nil {
		respondWithRevisionError(c, err)
		return
	}
	if out.Revision == nil {
		respondWithRevisionStatusError(c, http.StatusNotFound, ErrCodeRevisionNotFound, errRevisionNotFoundUserMsg, "revision for page %s not found", pageID)
		return
	}
	c.JSON(http.StatusOK, toRevisionResponse(out.Revision, r.userResolver))
}

// handleCompareRevisions 要求页面、基准和目标版本，任一比较快照缺失时返回 404。
// 差异计算由用例及历史服务完成，路由只转换响应结构。
```

### GET /api/pages/:id/revisions/compare

要求页面、基准和目标版本，任一比较快照缺失时返回 404。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | revisions / handleCompareRevisions |
| 身份 | 需要认证 |
| 路由参数与查询 | c.Param("id"))；c.Query("base"))；c.Query("target")) |
| 响应调用 | respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidPageID, errPageIDRequiredUserMsg, errPageIDRequiredLogMsg)；respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionCompareInvalidRequest, "Revision compare request is invalid", "revision compare request for page %s is invalid", pageID)；respondWithRevisionError(c, err)；respondWithRevisionStatusError(c, http.StatusNotFound, ErrCodeRevisionNotFound, errRevisionNotFoundUserMsg, "revision compare resource for page %s not found", pageID)；c.JSON(http.StatusOK, toComparisonResponse(out.Comparison, r.userResolver)) |
| 源码 | backend/internal/wiki/revisions/routes.go |

请求与响应实现：

```go
func (r *Routes) handleCompareRevisions(c *gin.Context) {
	pageID := strings.TrimSpace(c.Param("id"))
	baseRevisionID := strings.TrimSpace(c.Query("base"))
	targetRevisionID := strings.TrimSpace(c.Query("target"))
	if pageID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidPageID, errPageIDRequiredUserMsg, errPageIDRequiredLogMsg)
		return
	}
	if baseRevisionID == "" || targetRevisionID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionCompareInvalidRequest, "Revision compare request is invalid", "revision compare request for page %s is invalid", pageID)
		return
	}

	out, err := r.compareRevisions.Execute(c.Request.Context(), CompareRevisionsInput{
		PageID: pageID, BaseRevisionID: baseRevisionID, TargetRevisionID: targetRevisionID,
	})
	if err != nil {
		respondWithRevisionError(c, err)
		return
	}
	if out.Comparison == nil || out.Comparison.Base == nil || out.Comparison.Target == nil {
		respondWithRevisionStatusError(c, http.StatusNotFound, ErrCodeRevisionNotFound, errRevisionNotFoundUserMsg, "revision compare resource for page %s not found", pageID)
		return
	}
	c.JSON(http.StatusOK, toComparisonResponse(out.Comparison, r.userResolver))
}

// handleGetRevisionAsset 从历史快照定位资源字节，打开文件并设置不可信内容响应头。
// 类型优先用记录中的 MIME，其次按扩展名查询，再回退二进制流类型。
// ServeContent 使用可定位文件及修改时间处理内容响应，文件关闭错误在当前实现中不报告。
```

### GET /api/pages/:id/revisions/:revisionId/assets/*name

从历史快照定位资源字节，打开文件并设置不可信内容响应头。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | revisions / handleGetRevisionAsset |
| 身份 | 需要认证 |
| 路由参数与查询 | c.Param("id"))；c.Param("revisionId"))；c.Param("name"), "/")) |
| 响应调用 | respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidPageID, errPageIDRequiredUserMsg, errPageIDRequiredLogMsg)；respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidRevisionID, "Revision ID is required", "revision id is required")；respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionPreviewAssetInvalidName, "Revision asset name is invalid", "revision asset name for page %s revision %s is invalid", pageID, revisionID)；respondWithRevisionError(c, err)；respondWithRevisionStatusError(c, http.StatusNotFound, ErrCodeRevisionPreviewAssetNotFound, "Revision asset not found", "revision asset %s for page %s revision %s not found", assetName, pageID, revisionID)；respondWithRevisionStatusError(c, http.StatusInternalServerError, ErrCodeRevisionPreviewAssetBlobUnavailable, "Revision asset blob is unavailable", "revision asset blob %s for page %s revision %s is unavailable", assetName, pageID, revisionID)；respondWithRevisionStatusError(c, http.StatusInternalServerError, ErrCodeRevisionInternalError, "Failed to load revision asset", "failed to load revision asset") |
| 源码 | backend/internal/wiki/revisions/routes.go |

请求与响应实现：

```go
func (r *Routes) handleGetRevisionAsset(c *gin.Context) {
	pageID := strings.TrimSpace(c.Param("id"))
	revisionID := strings.TrimSpace(c.Param("revisionId"))
	assetName := strings.TrimSpace(strings.TrimPrefix(c.Param("name"), "/"))
	if pageID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidPageID, errPageIDRequiredUserMsg, errPageIDRequiredLogMsg)
		return
	}
	if revisionID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidRevisionID, "Revision ID is required", "revision id is required")
		return
	}
	if assetName == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionPreviewAssetInvalidName, "Revision asset name is invalid", "revision asset name for page %s revision %s is invalid", pageID, revisionID)
		return
	}

	out, err := r.getRevisionAsset.Execute(c.Request.Context(), GetRevisionAssetInput{
		PageID: pageID, RevisionID: revisionID, AssetName: assetName,
	})
	if err != nil {
		respondWithRevisionError(c, err)
		return
	}
	if out.Asset == nil {
		respondWithRevisionStatusError(c, http.StatusNotFound, ErrCodeRevisionPreviewAssetNotFound, "Revision asset not found", "revision asset %s for page %s revision %s not found", assetName, pageID, revisionID)
		return
	}

	// 快照索引存在而实际字节文件无法打开时返回内部不可用错误，不当作空文件发送。
	f, err := os.Open(out.Asset.Path)
	if err != nil {
		respondWithRevisionStatusError(c, http.StatusInternalServerError, ErrCodeRevisionPreviewAssetBlobUnavailable, "Revision asset blob is unavailable", "revision asset blob %s for page %s revision %s is unavailable", assetName, pageID, revisionID)
		return
	}
	defer func() { _ = f.Close() }()

	stat, err := f.Stat()
	if err != nil {
		respondWithRevisionStatusError(c, http.StatusInternalServerError, ErrCodeRevisionInternalError, "Failed to load revision asset", "failed to load revision asset")
		return
	}

	contentType := out.Asset.Asset.MIMEType
	if contentType == "" {
		contentType = mime.TypeByExtension(strings.ToLower(path.Ext(assetName)))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	security.SetUntrustedFileHeaders(c.Writer.Header(), assetName)
	c.Writer.Header().Set("Content-Type", contentType)
	http.ServeContent(c.Writer, c.Request, path.Base(assetName), stat.ModTime(), f)
}

// handleRestoreRevision 验证标识后使用当前用户作为恢复作者，成功返回当前页面公开结构。
```

### GET /api/pages/:id/revisions/:revisionId

要求页面与版本标识，快照或其版本记录为空时直接返回 404。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | revisions / handleGetRevision |
| 身份 | 需要认证 |
| 路由参数与查询 | c.Param("id"))；c.Param("revisionId")) |
| 响应调用 | respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidPageID, errPageIDRequiredUserMsg, errPageIDRequiredLogMsg)；respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidRevisionID, "Revision ID is required", "revision id is required")；respondWithRevisionError(c, err)；respondWithRevisionStatusError(c, http.StatusNotFound, ErrCodeRevisionNotFound, errRevisionNotFoundUserMsg, "revision %s for page %s not found", revisionID, pageID)；c.JSON(http.StatusOK, toSnapshotResponse(out.Snapshot, r.userResolver)) |
| 源码 | backend/internal/wiki/revisions/routes.go |

请求与响应实现：

```go
func (r *Routes) handleGetRevision(c *gin.Context) {
	pageID := strings.TrimSpace(c.Param("id"))
	revisionID := strings.TrimSpace(c.Param("revisionId"))
	if pageID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidPageID, errPageIDRequiredUserMsg, errPageIDRequiredLogMsg)
		return
	}
	if revisionID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionInvalidRevisionID, "Revision ID is required", "revision id is required")
		return
	}

	out, err := r.getRevision.Execute(c.Request.Context(), GetRevisionInput{PageID: pageID, RevisionID: revisionID})
	if err != nil {
		respondWithRevisionError(c, err)
		return
	}
	if out.Snapshot == nil || out.Snapshot.Revision == nil {
		respondWithRevisionStatusError(c, http.StatusNotFound, ErrCodeRevisionNotFound, errRevisionNotFoundUserMsg, "revision %s for page %s not found", revisionID, pageID)
		return
	}
	c.JSON(http.StatusOK, toSnapshotResponse(out.Snapshot, r.userResolver))
}

// handleGetLatestRevision 读取单页最新版本，服务返回空记录时作为版本不存在处理。
```

### POST /api/pages/:id/revisions/:revisionId/restore

验证标识后使用当前用户作为恢复作者，成功返回当前页面公开结构。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | revisions / handleRestoreRevision |
| 身份 | 需要认证；编辑者或管理员 |
| 路由参数与查询 | c.Param("id"))；c.Param("revisionId")) |
| 响应调用 | respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionRestoreInvalidPageID, "Failed to restore page", "failed to restore page %s", pageID)；respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionRestoreInvalidRevision, "Restore revision is invalid", "restore revision %s for page %s is invalid", revisionID, pageID)；respondWithRevisionError(c, err)；c.JSON(http.StatusOK, dto.ToAPIPage(out.Page, r.userResolver)) |
| 源码 | backend/internal/wiki/revisions/routes.go |

请求与响应实现：

```go
func (r *Routes) handleRestoreRevision(c *gin.Context) {
	pageID := strings.TrimSpace(c.Param("id"))
	revisionID := strings.TrimSpace(c.Param("revisionId"))
	if pageID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionRestoreInvalidPageID, "Failed to restore page", "failed to restore page %s", pageID)
		return
	}
	if revisionID == "" {
		respondWithRevisionStatusError(c, http.StatusBadRequest, ErrCodeRevisionRestoreInvalidRevision, "Restore revision is invalid", "restore revision %s for page %s is invalid", revisionID, pageID)
		return
	}

	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}

	out, err := r.restoreRevision.Execute(c.Request.Context(), RestoreRevisionInput{
		UserID: user.ID, PageID: pageID, RevisionID: revisionID,
	})
	if err != nil {
		respondWithRevisionError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ToAPIPage(out.Page, r.userResolver))
}
```

### GET /api/search/status

返回共享状态的快照，状态为空时将空值编码到响应。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | search / handleGetIndexingStatus |
| 身份 | 公开读取按实例设置检查 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | c.JSON(http.StatusOK, out.Status) |
| 源码 | backend/internal/wiki/search/routes.go |

请求与响应实现：

```go
func (r *Routes) handleGetIndexingStatus(c *gin.Context) {
	out := r.getIndexingStatus.Execute(c.Request.Context())
	c.JSON(http.StatusOK, out.Status)
}

// splitTags 拆分单个查询值中的逗号标签并去除边缘空白，不去重也不转换大小写。
func splitTags(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		tag := strings.TrimSpace(p)
		if tag != "" {
			result = append(result, tag)
		}
	}
	return result
}

// queryTags 按参数出现顺序合并重复查询参数，每个参数值还可包含多个逗号分隔标签。
// 完全没有该参数时返回 nil，存在但只含空项时返回空切片。
func queryTags(c *gin.Context, key string) []string {
	values := c.QueryArray(key)
	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, splitTags(value)...)
	}
	return result
}
```

### GET /api/search

接受关键词或非空标签列表，缺少两者时返回 400。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | search / handleSearch |
| 身份 | 公开读取按实例设置检查 |
| 路由参数与查询 | c.Query("q")；c.DefaultQuery("offset", "0")；c.DefaultQuery("limit", "20") |
| 响应调用 | respondWithSearchStatusError(c, http.StatusBadRequest, ErrCodeSearchMissingQuery, "Query parameter 'q' is required", "query parameter q is required")；respondWithSearchStatusError(c, http.StatusBadRequest, ErrCodeSearchInvalidOffset, "Invalid offset value", "invalid offset value")；respondWithSearchStatusError(c, http.StatusBadRequest, ErrCodeSearchInvalidLimit, "Invalid limit value", "invalid limit value")；respondWithSearchError(c, err)；c.JSON(http.StatusOK, out.Result) |
| 源码 | backend/internal/wiki/search/routes.go |

请求与响应实现：

```go
func (r *Routes) handleSearch(c *gin.Context) {
	query := c.Query("q")
	tags := queryTags(c, "tags")
	if query == "" && len(tags) == 0 {
		respondWithSearchStatusError(c, http.StatusBadRequest, ErrCodeSearchMissingQuery, "Query parameter 'q' is required", "query parameter q is required")
		return
	}

	// 关键词不在这里去除空白，标签解析会拆分逗号并丢弃全空白项。
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "20")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		respondWithSearchStatusError(c, http.StatusBadRequest, ErrCodeSearchInvalidOffset, "Invalid offset value", "invalid offset value")
		return
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		respondWithSearchStatusError(c, http.StatusBadRequest, ErrCodeSearchInvalidLimit, "Invalid limit value", "invalid limit value")
		return
	}

	out, err := r.search.Execute(c.Request.Context(), SearchInput{Query: query, Tags: tags, Offset: offset, Limit: limit})
	if err != nil {
		respondWithSearchError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Result)
}

// handleGetIndexingStatus 返回共享状态的快照，状态为空时将空值编码到响应。
```

### GET /api/admin/snapshot/status

用管理器是否存在表示功能可用性，而非周期调度的 Enabled 设置。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | snapshot / handleStatus |
| 身份 | 需要认证；管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | c.JSON(http.StatusOK, gin.H{"enabled": false})；c.JSON(http.StatusOK, gin.H{ |
| 源码 | backend/internal/wiki/snapshot/routes.go |

请求与响应实现：

```go
func (r *Routes) handleStatus(c *gin.Context) {
	if r.manager == nil {
		c.JSON(http.StatusOK, gin.H{"enabled": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled":        true,
		"settings":       r.currentSettings(),
		"retentionCount": r.currentSettings().RetentionCount,
		"status":         r.manager.Status(),
	})
}

// currentSettings 优先读取设置存储的副本，没有存储时使用管理器配置，
// 并以路由构造参数覆盖保留数量。调用此方法前需确认管理器存在。
```

### PUT /api/admin/snapshot/settings

只接受一个完整 JSON 值，拒绝未知字段、尾随内容与无效设置。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | snapshot / handleUpdateSettings |
| 身份 | 需要认证；管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot settings"})；c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot settings"})；httpinternal.RespondError(c, http.StatusBadRequest, loc.Code, loc.Message, loc.Template, loc.Args...)；c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})；c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save snapshot settings"})；c.JSON(http.StatusOK, gin.H{"settings": value}) |
| 源码 | backend/internal/wiki/snapshot/routes.go |

请求与响应实现：

```go
func (r *Routes) handleUpdateSettings(c *gin.Context) {
	if r.manager == nil || r.settings == nil {
		r.respondNotEnabled(c)
		return
	}
	var value snapshotSvc.Settings
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot settings"})
		return
	}
	// 第二次解码必须直接到达结尾，避免接受两个相连对象或非法尾部。
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid snapshot settings"})
		return
	}
	if err := snapshotSvc.ValidateSettings(value); err != nil {
		if loc, ok := sharederrors.AsLocalizedError(err); ok {
			httpinternal.RespondError(c, http.StatusBadRequest, loc.Code, loc.Message, loc.Template, loc.Args...)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	if err := r.settings.Put(value); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save snapshot settings"})
		return
	}
	// 通知只是登记重算事件，不取消当前正在生成的归档。
	r.manager.UpdateSettings(value)
	if r.scheduler != nil {
		r.scheduler.UpdateSettings()
	}
	c.JSON(http.StatusOK, gin.H{"settings": value})
}

// respondNotEnabled 在操作所需依赖缺失时返回统一的 503 错误。
```

### GET /api/admin/snapshot

返回管理器读取的元数据列表，按记录的创建时间降序排列。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | snapshot / handleList |
| 身份 | 需要认证；管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithSnapshotError(c, err)；c.JSON(http.StatusOK, gin.H{"snapshots": entries}) |
| 源码 | backend/internal/wiki/snapshot/routes.go |

请求与响应实现：

```go
func (r *Routes) handleList(c *gin.Context) {
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}
	entries, err := r.manager.List()
	if err != nil {
		respondWithSnapshotError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"snapshots": entries})
}

// handleTrigger 尝试向容量为一的手动事件通道排入任务，成功即返回 202。
// 已有待处理事件时返回 409；正在归档不等于通道已满，执行中仍能排队一次。
```

### POST /api/admin/snapshot

尝试向容量为一的手动事件通道排入任务，成功即返回 202。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | snapshot / handleTrigger |
| 身份 | 需要认证；管理员 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithSnapshotError(c, snapshotSvc.ErrAlreadyRunning)；c.JSON(http.StatusAccepted, gin.H{"triggered": true}) |
| 源码 | backend/internal/wiki/snapshot/routes.go |

请求与响应实现：

```go
func (r *Routes) handleTrigger(c *gin.Context) {
	if r.scheduler == nil {
		r.respondNotEnabled(c)
		return
	}
	if !r.scheduler.TriggerNow() {
		respondWithSnapshotError(c, snapshotSvc.ErrAlreadyRunning)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"triggered": true})
}

// handleDownload 校验快照标识并打开 ZIP，通过 ServeContent 发送附件。
// 文件句柄在处理结束时关闭，响应禁止缓存并使用磁盘文件的修改时间。
```

### GET /api/admin/snapshot/:id/download

校验快照标识并打开 ZIP，通过 ServeContent 发送附件。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | snapshot / handleDownload |
| 身份 | 需要认证；管理员；Cookie 会话专用 |
| 路由参数与查询 | c.Param("id") |
| 响应调用 | respondWithSnapshotError(c, err)；respondWithSnapshotStatusError(c, http.StatusInternalServerError, ErrCodeSnapshotInternalError, "Failed to open snapshot", "failed to open snapshot")；respondWithSnapshotStatusError(c, http.StatusInternalServerError, ErrCodeSnapshotInternalError, "Failed to read snapshot", "failed to read snapshot")；c.Header("Content-Disposition", disposition)；c.Header("Cache-Control", "no-store")；c.Header("Pragma", "no-cache") |
| 源码 | backend/internal/wiki/snapshot/routes.go |

请求与响应实现：

```go
func (r *Routes) handleDownload(c *gin.Context) {
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}
	id := c.Param("id")
	zipPath, err := r.manager.SnapshotZipPath(id)
	if err != nil {
		respondWithSnapshotError(c, err)
		return
	}

	f, err := os.Open(zipPath)
	if err != nil {
		respondWithSnapshotStatusError(c, http.StatusInternalServerError, ErrCodeSnapshotInternalError, "Failed to open snapshot", "failed to open snapshot")
		return
	}
	defer func() { _ = f.Close() }()

	stat, err := f.Stat()
	if err != nil {
		respondWithSnapshotStatusError(c, http.StatusInternalServerError, ErrCodeSnapshotInternalError, "Failed to read snapshot", "failed to read snapshot")
		return
	}

	// 只将路径末段作为下载名称，由标准库编码附件头，编码失败时保留附件类型。
	filename := filepath.Base(zipPath)
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
	if disposition == "" {
		disposition = "attachment"
	}
	c.Header("Content-Disposition", disposition)
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Writer.Header().Set("Content-Type", "application/zip")
	http.ServeContent(c.Writer, c.Request, filename, stat.ModTime(), f)
}

// handleDelete 委托管理器删除 ZIP 和元数据。成功后返回 ok，
// 两项删除不构成事务，失败响应不代表此前删除的文件已被恢复。
```

### DELETE /api/admin/snapshot/:id

委托管理器删除 ZIP 和元数据。成功后返回 ok，

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | snapshot / handleDelete |
| 身份 | 需要认证；管理员 |
| 路由参数与查询 | c.Param("id") |
| 响应调用 | respondWithSnapshotError(c, err)；c.JSON(http.StatusOK, gin.H{"ok": true}) |
| 源码 | backend/internal/wiki/snapshot/routes.go |

请求与响应实现：

```go
func (r *Routes) handleDelete(c *gin.Context) {
	if r.manager == nil {
		r.respondNotEnabled(c)
		return
	}
	id := c.Param("id")
	if err := r.manager.Delete(id); err != nil {
		respondWithSnapshotError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
```

### GET /api/tags

解析筛选、已选标签和整数条数，条数范围由用例归一化。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | tags / handleGetTags |
| 身份 | 公开读取按实例设置检查 |
| 路由参数与查询 | c.DefaultQuery("q", "")；c.DefaultQuery("limit", "50") |
| 响应调用 | respondWithTagsBadRequest(c, ErrCodeTagsInvalidLimit, "Invalid limit value", "invalid limit value")；respondWithTagsError(c, err)；c.JSON(http.StatusOK, out.Tags) |
| 源码 | backend/internal/wiki/tags/routes.go |

请求与响应实现：

```go
func (r *Routes) handleGetTags(c *gin.Context) {
	filter := c.DefaultQuery("q", "")
	limitStr := c.DefaultQuery("limit", "50")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		respondWithTagsBadRequest(c, ErrCodeTagsInvalidLimit, "Invalid limit value", "invalid limit value")
		return
	}

	out, err := r.getTags.Execute(c.Request.Context(), GetTagsInput{
		Filter:   filter,
		Selected: queryTags(c, "selected"),
		Limit:    limit,
	})
	if err != nil {
		respondWithTagsError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Tags)
}

// handleGetPagesByTags 合并重复 tags 参数与逗号列表，没有有效条目时返回 400。
// 页面查询和标签大小写规范化由用例执行。
```

### GET /api/tags/pages

合并重复 tags 参数与逗号列表，没有有效条目时返回 400。

| 项目 | 实际约束 |
| --- | --- |
| 模块与处理器 | tags / handleGetPagesByTags |
| 身份 | 公开读取按实例设置检查 |
| 路由参数与查询 | 此处理器未直接读取查询或路由参数 |
| 响应调用 | respondWithTagsBadRequest(c, ErrCodeTagsMissingParam, "Query parameter 'tags' is required", "query parameter tags is required")；respondWithTagsError(c, err)；c.JSON(http.StatusOK, out.Pages) |
| 源码 | backend/internal/wiki/tags/routes.go |

请求与响应实现：

```go
func (r *Routes) handleGetPagesByTags(c *gin.Context) {
	tagList := queryTags(c, "tags")
	if len(tagList) == 0 {
		respondWithTagsBadRequest(c, ErrCodeTagsMissingParam, "Query parameter 'tags' is required", "query parameter tags is required")
		return
	}

	out, err := r.getPagesByTags.Execute(c.Request.Context(), GetPagesByTagsInput{Tags: tagList})
	if err != nil {
		respondWithTagsError(c, err)
		return
	}
	c.JSON(http.StatusOK, out.Pages)
}

// splitTags 拆分单个查询值，去除两端空白并跳过空项，保留大小写及重复条目。
func splitTags(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			result = append(result, t)
		}
	}
	return result
}

// queryTags 顺序展开同名参数的所有值，不将仅第一项作为完整选择。
func queryTags(c *gin.Context, key string) []string {
	values := c.QueryArray(key)
	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, splitTags(value)...)
	}
	return result
}
```

## 动态设置、安装与非 API 资源

实例设置提供 GET 和 PUT /api/admin/settings/preferences，只有偏好服务存在时注册。PUT 要求管理员 Cookie 会话，接受 JSON 局部设置对象，字段由偏好服务校验。另有 public-access 与 toc-display 两个固定写入口。下列代码列出实际绑定与权限：

```go
func (r *Routes) RegisterRoutes(ctx httpinternal.RouterContext) {
	adminGroup := ctx.APIAuthGroup(r.authService).Group("/admin/settings")
	if ctx.Opts.Preferences != nil {
		adminGroup.GET("/preferences", authmw.RequireAdmin(), func(c *gin.Context) { c.JSON(200, ctx.Opts.Preferences.Get()) })
		// JSON 对象作为局部设置交给服务校验并保存，本地化校验错误返回 400。
		adminGroup.PUT("/preferences", authmw.RequireAdmin(), authmw.RequireCookieSession(), func(c *gin.Context) {
			var patch map[string]any
			if err := c.ShouldBindJSON(&patch); err != nil {
				c.JSON(400, gin.H{"error": localization.Message("invalid payload")})
				return
			}
			value, err := ctx.Opts.Preferences.Patch(patch)
			if err != nil {
				if loc, ok := sharederrors.AsLocalizedError(err); ok {
					httpinternal.RespondError(c, http.StatusBadRequest, loc.Code, loc.Message, loc.Template, loc.Args...)
				} else {
					r.log.Error("failed to save preferences", "error", err)
					httpinternal.RespondError(c, http.StatusInternalServerError, ErrCodeInternal, "Failed to save preferences", "failed to save preferences")
				}
				return
			}
			c.JSON(200, value)
		})
	}
	// 公开读取影响整个实例，RequireCookieSession 通过密钥标记拒绝 API 密钥调用。
	// 该守卫不自行读取 Cookie，其他已注入用户身份仍遵循上游鉴权规则。
	adminGroup.PUT("/public-access",
		authmw.RequireAdmin(),
		authmw.RequireCookieSession(),
		r.handleSetPublicAccess,
	)
	adminGroup.PUT("/toc-display",
		authmw.RequireAdmin(),
		r.handleSetTocDisplay,
	)
}

// handleSetPublicAccess 使用布尔指针区分缺少字段与明确关闭，缺失或非法 JSON 返回 400。
// 服务写入成功后重新读取实际值，并将操作者标识写入日志。
```

安装向导由后端独立的安装服务处理，使用安装令牌、来源校验、服务端会话、步骤顺序及版本约束，不使用普通业务 API 假装完成安装。请按实际向导操作，不能绕过数据库检查直接提交完成。

资源入口还包括 /assets 下的附件、/branding/:filename、/favicon.ico 与内置 /defaults 资源。资源读取按实际处理器的路径及内容规则校验，不能把任意磁盘路径当下载地址。

## 错误与兼容

## 安装向导接口

安装服务处于首次配置阶段时使用以下入口，不与正式业务登录混用。POST 要求 X-LeafDoc-Setup: 1。令牌验证后保留 leafdoc_setup Cookie，后续每步携带服务返回的 revision 和 step，版本过期返回 409，不能跳步。会话有效期为 30 分钟，每次有效步骤刷新期限。

| 方法与路径 | 输入 | 返回与操作 |
| --- | --- | --- |
| GET /api/setup/status | 安装 Cookie，可为空 | 未建立会话返回 step=0、revision=0，否则返回当前步骤视图 |
| GET /api/setup/ca | 无账号会话 | 按安装配置提供公开 CA，不提供私钥 |
| POST /api/setup/token | setupToken | 校验实例令牌，创建会话，返回 step=1、revision=1 |
| POST /api/setup/address | revision、step 及连接配置 | 校验数据库地址，成功进入下一步 |
| POST /api/setup/credentials | revision、step 及数据库凭据 | 验证数据库连接身份 |
| POST /api/setup/database | revision、step、数据库选择、需要时 confirmDatabase | 检查结构与操作授权，不能绕过库名确认清空 |
| POST /api/setup/admin | revision、step、admin、email、adminPassword | 校验管理员身份和密码要求 |
| POST /api/setup/back | revision、step | 只返回已完成步骤，重新设置后续检查状态 |
| POST /api/setup | revision、step | 仅 step=5 执行最终数据库准备和安装保存 |

数据库连接字段来自 installation.Config，包含 host、port、database、user、password、tls、ca；账号字段包含 admin、email、adminPassword。未知 JSON 字段、超过 128 KiB 的请求和多个连续 JSON 对象被拒绝。正式安装后这些入口不作为通用数据库管理 API。

## 设置事件与附加资源

| 路径 | 用途与条件 |
| --- | --- |
| GET /api/settings/events | 设置事件流，前端据此同步设置变化，不是普通一次 JSON 查询 |
| GET /custom.css | 配置了对应前端 CSS 时提供，不将任意本地文件开放下载 |
| GET /defaults/* | 程序内置品牌资源，教程图片从此读取 |

400 检查请求格式和字段，401 检查认证，403 检查角色、CSRF 或专用凭据，404 检查 ID 与功能是否注册，409 等冲突应重新读取当前状态再提交，503 检查恢复或功能状态。以实际响应错误代码与消息定位，不无限重试写请求。分页、大小、格式和功能开关按各处理器实际校验执行。
