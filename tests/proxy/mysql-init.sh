#!/bin/sh
# Only the two generated schemas inside the private, temporary MySQL container are created.
set -eu
for database_name in "$E2E_DATABASE" "$E2E_AUTOCREATE_DATABASE"; do
  case "$database_name" in leafdoc_proxy_test_*) ;; *) echo "Invalid isolated database name" >&2; exit 1 ;; esac
  case "$database_name" in *[!a-z0-9_]*) echo "Unsafe database name" >&2; exit 1 ;; esac
  MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --protocol=socket -uroot -e "CREATE DATABASE \`$database_name\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
done
