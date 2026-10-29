package database

import (
	"database/sql"
	"fmt"
)

// relationSchema owns the tables shared by authentication and derived indexes.
// Binary IDs and paths retain exact identity; human labels use Unicode comparison.
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

// EnsureStoreSchema creates relational stores without creating local database files.
func EnsureStoreSchema(db *sql.DB) error {
	for _, statement := range relationSchema {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("initialize MySQL relation schema: %w", err)
		}
	}
	return nil
}
