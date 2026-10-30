-- LeafDoc MySQL 8.0 完整结构和初始化默认值。
-- 生成来源：node scripts/container.mjs schema-write，核验：node scripts/container.mjs schema。
-- 在选定的空业务数据库执行，不包含 CREATE DATABASE、DROP 或服务器连接凭据。
-- 未声明 DEFAULT 的非空业务字段必须由应用提供，不以空字符串冒充业务默认值。
-- NULL 表示明确的未发生或未解析状态，时间戳精度和字段长度均以以下 DDL 为准。
-- 全部逻辑关系由应用维护，当前无数据库外键、视图、存储过程、触发器或事件。
-- schema_migrations 未显式声明字符集，继承目标数据库设置；其字段均为数字和时间，不存储文本。
-- content_records 的 settings 记录键为下方五个 JSON 文件名，payload 是配置对象。
-- page_nodes 的键为页面 ID，payload 包含 node 元信息、parentId 和 raw 原始 Markdown，子节点与父引用不重复序列化。
-- revisions 的键为 页面ID:版本ID，revision_content 的键为 页面ID:正文SHA256，revision_manifests 的键为清单SHA256。
-- 历史附件内容保存在实例目录中，数据库保存历史清单关联，不将文件系统内容虚构为 SQL 表。
-- instance_state 保存教程初始化状态，complete 为布尔值；初始化完成后不会在每次启动时重复建立教程。
SET NAMES utf8mb4;
SET time_zone = '+00:00';

-- content_records：业务文档和设置。namespace 与 record_key 组成主键，payload 为 UTF-8 JSON 或正文二进制。settings 使用稳定设置键。page_nodes 使用页面 ID，包含正文、父节点、排序和元信息。revisions、revision_content、revision_manifests 保存历史和索引。instance_state 保存教程初始化标记。
--   namespace：业务命名空间。完整定义见 DDL。
--   record_key：命名空间内唯一记录键。完整定义见 DDL。
--   payload：完整业务记录的 JSON 或二进制正文。完整定义见 DDL。
CREATE TABLE IF NOT EXISTS content_records (namespace VARCHAR(64) NOT NULL, record_key VARCHAR(255) NOT NULL, payload LONGBLOB NOT NULL, PRIMARY KEY(namespace,record_key)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

-- auth_state：认证代次。会话验证使用 generation 使恢复前的认证状态失效。
--   id：记录唯一标识。完整定义见 DDL。
--   generation：随机认证代次标识。完整定义见 DDL。
CREATE TABLE IF NOT EXISTS auth_state (
        id TINYINT PRIMARY KEY,
        generation VARCHAR(64) COLLATE utf8mb4_bin NOT NULL
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- schema_migrations：已应用的结构版本及应用时间。
--   version：结构迁移版本号。完整定义见 DDL。
--   applied_at：迁移应用时间，数据库当前 UTC 时间。完整定义见 DDL。
CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)) ENGINE=InnoDB;

INSERT INTO schema_migrations(version) VALUES(1) ON DUPLICATE KEY UPDATE version=schema_migrations.version;

-- users：用户账号。用户名和邮箱分别唯一。password 仅保存 bcrypt 散列，role 使用应用角色定义。
--   id：记录唯一标识。完整定义见 DDL。
--   username：唯一登录用户名。完整定义见 DDL。
--   password：bcrypt 密码散列。完整定义见 DDL。
--   email：唯一账号邮箱。完整定义见 DDL。
--   role：权限角色 admin、editor 或 viewer。完整定义见 DDL。
--   created_at：记录创建时间，日期列使用 UTC，BIGINT 列使用 Unix 秒。完整定义见 DDL。
--   must_set_password：是否必须先设置密码。完整定义见 DDL。
--   must_change_password：是否必须先更改初始密码。完整定义见 DDL。
CREATE TABLE IF NOT EXISTS users (
        id VARCHAR(64) COLLATE utf8mb4_bin PRIMARY KEY,
        username VARCHAR(191) NOT NULL UNIQUE,
        password VARCHAR(255) NOT NULL,
        email VARCHAR(191) NOT NULL UNIQUE,
        role VARCHAR(32) NOT NULL,
        created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
        must_set_password BOOLEAN NOT NULL DEFAULT FALSE,
        must_change_password BOOLEAN NOT NULL DEFAULT FALSE
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- sessions：访问和刷新会话。user_id 在应用层关联 users.id，时间采用 Unix 秒，revoked_at 为空表示未撤销。
--   id：记录唯一标识。完整定义见 DDL。
--   user_id：所属用户 ID，应用层关联 users.id。完整定义见 DDL。
--   token_type：访问或刷新令牌类型。完整定义见 DDL。
--   created_at：记录创建时间，日期列使用 UTC，BIGINT 列使用 Unix 秒。完整定义见 DDL。
--   expires_at：失效时刻，Unix 秒；仅允许 NULL 的 API 密钥字段以 NULL 表示不设到期时间。完整定义见 DDL。
--   revoked_at：撤销时刻，Unix 秒，NULL 表示未撤销。完整定义见 DDL。
CREATE TABLE IF NOT EXISTS sessions (
        id VARCHAR(128) COLLATE utf8mb4_bin PRIMARY KEY,
        user_id VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
        token_type VARCHAR(32) NOT NULL,
        created_at BIGINT NOT NULL,
        expires_at BIGINT NOT NULL,
        revoked_at BIGINT NULL,
        INDEX sessions_user_type(user_id, token_type),
        INDEX sessions_expiry(expires_at)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- api_keys：API 密钥元数据及散列。user_id、created_by 关联 users.id，prefix 唯一，密钥明文不存储。
--   id：记录唯一标识。完整定义见 DDL。
--   name：用户填写的显示名称。完整定义见 DDL。
--   user_id：所属用户 ID，应用层关联 users.id。完整定义见 DDL。
--   prefix：公开密钥前缀，唯一。完整定义见 DDL。
--   key_hash：API 密钥散列。完整定义见 DDL。
--   role：权限角色 admin、editor 或 viewer。完整定义见 DDL。
--   expires_at：失效时刻，Unix 秒；仅允许 NULL 的 API 密钥字段以 NULL 表示不设到期时间。完整定义见 DDL。
--   created_by：创建者用户 ID，应用层关联 users.id。完整定义见 DDL。
--   created_at：记录创建时间，日期列使用 UTC，BIGINT 列使用 Unix 秒。完整定义见 DDL。
--   last_used_at：最后使用时刻，Unix 秒，NULL 表示未使用。完整定义见 DDL。
--   revoked_at：撤销时刻，Unix 秒，NULL 表示未撤销。完整定义见 DDL。
CREATE TABLE IF NOT EXISTS api_keys (
        id VARCHAR(64) COLLATE utf8mb4_bin PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        user_id VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
        prefix VARCHAR(128) COLLATE utf8mb4_bin NOT NULL UNIQUE,
        key_hash VARCHAR(128) COLLATE utf8mb4_bin NOT NULL,
        role VARCHAR(32) NOT NULL,
        expires_at BIGINT NULL,
        created_by VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
        created_at BIGINT NOT NULL,
        last_used_at BIGINT NULL,
        revoked_at BIGINT NULL,
        INDEX api_keys_owner(user_id)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- email_tokens：邮件操作令牌散列。user_id 关联 users.id，时间采用 Unix 秒，consumed_at 为空表示未消费。
--   id：记录唯一标识。完整定义见 DDL。
--   token_hash：邮件令牌散列。完整定义见 DDL。
--   user_id：所属用户 ID，应用层关联 users.id。完整定义见 DDL。
--   purpose：邮件令牌用途。完整定义见 DDL。
--   created_at：记录创建时间，日期列使用 UTC，BIGINT 列使用 Unix 秒。完整定义见 DDL。
--   expires_at：失效时刻，Unix 秒；仅允许 NULL 的 API 密钥字段以 NULL 表示不设到期时间。完整定义见 DDL。
--   consumed_at：消费时刻，Unix 秒，NULL 表示未消费。完整定义见 DDL。
CREATE TABLE IF NOT EXISTS email_tokens (
        id VARCHAR(64) COLLATE utf8mb4_bin PRIMARY KEY,
        token_hash VARCHAR(128) COLLATE utf8mb4_bin NOT NULL,
        user_id VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
        purpose VARCHAR(32) NOT NULL,
        created_at BIGINT NOT NULL,
        expires_at BIGINT NOT NULL,
        consumed_at BIGINT NULL,
        INDEX email_tokens_owner(user_id),
        INDEX email_tokens_expiry(expires_at)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- favorites：用户收藏，user_id 关联 users.id，page_id 关联 page_nodes 记录，复合主键防止重复收藏。
--   user_id：所属用户 ID，应用层关联 users.id。完整定义见 DDL。
--   page_id：页面 ID，应用层关联 page_nodes 记录。完整定义见 DDL。
--   created_at：记录创建时间，日期列使用 UTC，BIGINT 列使用 Unix 秒。完整定义见 DDL。
CREATE TABLE IF NOT EXISTS favorites (
        user_id VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
        page_id VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
        created_at DATETIME(6) NOT NULL,
        PRIMARY KEY(user_id, page_id)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- links：页面出入链接派生索引。from_page_id 和 to_page_id 关联页面 ID，未解析目标为 NULL，to_path 保存目标路径。
--   from_page_id：来源页面 ID。完整定义见 DDL。
--   to_page_id：目标页面 ID，NULL 表示目标尚未解析。完整定义见 DDL。
--   to_path：目标页面路径。完整定义见 DDL。
--   from_title：来源页面标题。完整定义见 DDL。
--   state：链接状态枚举，0 为已解析，1 为链接失效，2 为目标不唯一。完整定义见 DDL。
CREATE TABLE IF NOT EXISTS links (
        from_page_id VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
        to_page_id VARCHAR(64) COLLATE utf8mb4_bin NULL,
        to_path VARCHAR(512) COLLATE utf8mb4_bin NOT NULL,
        from_title TEXT NULL,
        state INTEGER NOT NULL DEFAULT 0,
        PRIMARY KEY(from_page_id, to_path),
        INDEX links_target(to_page_id),
        INDEX links_path(to_path),
        INDEX links_state(state)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- page_tags：页面标签，page_id 关联页面 ID，复合主键防止同一页面重复标签。
--   page_id：页面 ID，应用层关联 page_nodes 记录。完整定义见 DDL。
--   tag：标签文本。完整定义见 DDL。
CREATE TABLE IF NOT EXISTS page_tags (
        page_id VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
        tag VARCHAR(255) NOT NULL,
        PRIMARY KEY(page_id, tag),
        INDEX page_tags_tag(tag)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- page_meta：页面摘要，page_id 关联页面 ID。
--   page_id：页面 ID，应用层关联 page_nodes 记录。完整定义见 DDL。
--   excerpt：派生页面摘要。完整定义见 DDL。
CREATE TABLE IF NOT EXISTS page_meta (
        page_id VARCHAR(64) COLLATE utf8mb4_bin PRIMARY KEY,
        excerpt TEXT NOT NULL
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- page_properties：页面属性，page_id 关联页面 ID，property_key 保持精确大小写，value_type 默认为 text。
--   page_id：页面 ID，应用层关联 page_nodes 记录。完整定义见 DDL。
--   property_key：属性名称。完整定义见 DDL。
--   property_value：属性值的文本表示。完整定义见 DDL。
--   value_type：属性类型，默认 text。完整定义见 DDL。
CREATE TABLE IF NOT EXISTS page_properties (
        page_id VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
        property_key VARCHAR(191) COLLATE utf8mb4_bin NOT NULL,
        property_value TEXT COLLATE utf8mb4_bin NOT NULL,
        value_type VARCHAR(32) NOT NULL DEFAULT 'text',
        PRIMARY KEY(page_id, property_key),
        INDEX property_keys(property_key)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- search_documents：可重建的页面全文搜索索引，pageID 关联页面 ID，ngram 解析器支持中文搜索。
--   pageID：页面 ID，与 page_nodes 记录关联。完整定义见 DDL。
--   path：页面访问路径。完整定义见 DDL。
--   filepath：页面在应用存储层的逻辑路径。完整定义见 DDL。
--   kind：页面或章节类型。完整定义见 DDL。
--   title：页面标题。完整定义见 DDL。
--   headings：派生标题检索文本。完整定义见 DDL。
--   content：派生正文检索文本。完整定义见 DDL。
CREATE TABLE IF NOT EXISTS search_documents (
        pageID VARCHAR(64) COLLATE utf8mb4_bin PRIMARY KEY,
        path TEXT NOT NULL,
        filepath TEXT NOT NULL,
        kind VARCHAR(32) NOT NULL,
        title TEXT NOT NULL,
        headings LONGTEXT NOT NULL,
        content LONGTEXT NOT NULL,
        FULLTEXT INDEX search_text(title, headings, content) WITH PARSER ngram
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 设置仅在缺失时插入，不覆盖管理员已保存的设置。
INSERT INTO content_records(namespace, record_key, payload) VALUES('settings', 'preferences.json', '{"dateFormat":"iso","language":"zh","timeFormat":"24h"}') ON DUPLICATE KEY UPDATE record_key=content_records.record_key;
INSERT INTO content_records(namespace, record_key, payload) VALUES('settings', 'public-access.json', '{"enabled":true}') ON DUPLICATE KEY UPDATE record_key=content_records.record_key;
INSERT INTO content_records(namespace, record_key, payload) VALUES('settings', 'toc-display.json', '{"alwaysShow":false,"mode":"never"}') ON DUPLICATE KEY UPDATE record_key=content_records.record_key;
INSERT INTO content_records(namespace, record_key, payload) VALUES('settings', 'snapshot-settings.json', '{"enabled":true,"runOnStartup":true,"mode":"daily","interval":"24h0m0s","hour":0,"minute":0,"timeZone":"Asia/Shanghai","retentionCount":10}') ON DUPLICATE KEY UPDATE record_key=content_records.record_key;
INSERT INTO content_records(namespace, record_key, payload) VALUES('settings', 'branding.json', '{"siteName":"LeafDoc","logoFile":"","faviconFile":""}') ON DUPLICATE KEY UPDATE record_key=content_records.record_key;

-- 初始账号仅在尚无管理员时建立，ID 为 SQL 引导记录标识，密码仅存储 bcrypt。
INSERT INTO users(id, username, password, email, role, must_set_password, must_change_password) SELECT UUID(), 'Nian', '$2a$10$zDwHEZt59LBmBoTihZJ6HemXPHHl5qwlX4fmnQ8bfSMD2W6gzYWCG', 'Nian@LeafDoc.doc', 'admin', FALSE, FALSE WHERE NOT EXISTS (SELECT 1 FROM users WHERE role='admin');
-- 教程、随机认证代次、页面 ID 和运行密钥由应用在首次启动时生成，不使用固定伪造值。
