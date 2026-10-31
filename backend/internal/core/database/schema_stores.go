// 本文件集中定义关系表和内容记录表，应用初始化与 SQL 导出共用这些语句。
package database

import (
	"database/sql"
	"fmt"
)

// relationSchema 按执行次序保存认证、收藏、链接、标签、属性和搜索表的定义。
// 显式标记 utf8mb4_bin 的标识字段按该规则比较，其他字符串沿用各表默认比较规则。
// CREATE TABLE IF NOT EXISTS 只保证表存在，不负责把已存在表变更为这里的列结构。
var relationSchema = []string{
	`CREATE TABLE IF NOT EXISTS auth_state (
        id TINYINT PRIMARY KEY,
        generation VARCHAR(64) COLLATE utf8mb4_bin NOT NULL
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)) ENGINE=InnoDB`,
	`INSERT INTO schema_migrations(version) VALUES(1) ON DUPLICATE KEY UPDATE version=schema_migrations.version`,
	`CREATE TABLE IF NOT EXISTS users (
        id VARCHAR(64) COLLATE utf8mb4_bin PRIMARY KEY,
        username VARCHAR(191) NOT NULL UNIQUE,
        password VARCHAR(255) NOT NULL,
        email VARCHAR(191) NOT NULL UNIQUE,
        role VARCHAR(32) NOT NULL,
        created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
        must_set_password BOOLEAN NOT NULL DEFAULT FALSE,
        must_change_password BOOLEAN NOT NULL DEFAULT FALSE
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS sessions (
        id VARCHAR(128) COLLATE utf8mb4_bin PRIMARY KEY,
        user_id VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
        token_type VARCHAR(32) NOT NULL,
        created_at BIGINT NOT NULL,
        expires_at BIGINT NOT NULL,
        revoked_at BIGINT NULL,
        INDEX sessions_user_type(user_id, token_type),
        INDEX sessions_expiry(expires_at)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS api_keys (
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
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS email_tokens (
        id VARCHAR(64) COLLATE utf8mb4_bin PRIMARY KEY,
        token_hash VARCHAR(128) COLLATE utf8mb4_bin NOT NULL,
        user_id VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
        purpose VARCHAR(32) NOT NULL,
        created_at BIGINT NOT NULL,
        expires_at BIGINT NOT NULL,
        consumed_at BIGINT NULL,
        INDEX email_tokens_owner(user_id),
        INDEX email_tokens_expiry(expires_at)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS favorites (
        user_id VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
        page_id VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
        created_at DATETIME(6) NOT NULL,
        PRIMARY KEY(user_id, page_id)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS links (
        from_page_id VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
        to_page_id VARCHAR(64) COLLATE utf8mb4_bin NULL,
        to_path VARCHAR(512) COLLATE utf8mb4_bin NOT NULL,
        from_title TEXT NULL,
        state INTEGER NOT NULL DEFAULT 0,
        PRIMARY KEY(from_page_id, to_path),
        INDEX links_target(to_page_id),
        INDEX links_path(to_path),
        INDEX links_state(state)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS page_tags (
        page_id VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
        tag VARCHAR(255) NOT NULL,
        PRIMARY KEY(page_id, tag),
        INDEX page_tags_tag(tag)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS page_meta (
        page_id VARCHAR(64) COLLATE utf8mb4_bin PRIMARY KEY,
        excerpt TEXT NOT NULL
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS page_properties (
        page_id VARCHAR(64) COLLATE utf8mb4_bin NOT NULL,
        property_key VARCHAR(191) COLLATE utf8mb4_bin NOT NULL,
        property_value TEXT COLLATE utf8mb4_bin NOT NULL,
        value_type VARCHAR(32) NOT NULL DEFAULT 'text',
        PRIMARY KEY(page_id, property_key),
        INDEX property_keys(property_key)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS search_documents (
        pageID VARCHAR(64) COLLATE utf8mb4_bin PRIMARY KEY,
        path TEXT NOT NULL,
        filepath TEXT NOT NULL,
        kind VARCHAR(32) NOT NULL,
        title TEXT NOT NULL,
        headings LONGTEXT NOT NULL,
        content LONGTEXT NOT NULL,
        FULLTEXT INDEX search_text(title, headings, content) WITH PARSER ngram
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
}

// EnsureStoreSchema 逐条执行关系结构语句，首条失败立即返回带上下文的错误。
// 这些语句未包在单一事务中，失败前已经成功执行的建表结果会保留在 MySQL 中。
func EnsureStoreSchema(db *sql.DB) error {
	for _, statement := range relationSchema {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("initialize MySQL relation schema: %w", err)
		}
	}
	return nil
}

// contentSchema 定义通用字节载荷表，与 relationSchema 共同作为可执行结构的权威定义。
const contentSchema = `CREATE TABLE IF NOT EXISTS content_records (namespace VARCHAR(64) NOT NULL, record_key VARCHAR(255) NOT NULL, payload LONGBLOB NOT NULL, PRIMARY KEY(namespace,record_key)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin`

// SchemaStatements 返回包含内容表及全部关系语句的新切片，供导出和一致性测试使用。
// 修改返回切片的元素不会改写 relationSchema 中的元素。
func SchemaStatements() []string {
	return append([]string{contentSchema}, relationSchema...)
}
