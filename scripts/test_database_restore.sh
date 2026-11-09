#!/usr/bin/env bash
set -euo pipefail

# This script is intended only for the temporary databases created by GitHub Actions.
if [[ "${GITHUB_ACTIONS:-}" != "true" ]]; then
  echo "The database recovery test may only run in GitHub Actions." >&2
  exit 2
fi

test -n "${GITHUB_RUN_ID:-}"
restored="rustdesk_restore_${GITHUB_RUN_ID}"

export MYSQL_PWD=test-only-password
mysql_base=(mysql --host=127.0.0.1 --port=3306 --user=root --batch --skip-column-names)
mysqldump --host=127.0.0.1 --port=3306 --user=root \
  --single-transaction --no-tablespaces --set-gtid-purged=OFF \
  rustdesk_upgrade > /tmp/rustdesk-mysql-backup.sql
test -s /tmp/rustdesk-mysql-backup.sql
"${mysql_base[@]}" -e "CREATE DATABASE \`${restored}\`;"
"${mysql_base[@]}" "${restored}" < /tmp/rustdesk-mysql-backup.sql
mysql_policy=$("${mysql_base[@]}" -e \
  "SELECT COUNT(*) FROM \`${restored}\`.device_policies WHERE name='Legacy policy' AND access_mode='view' AND trust_duration_days=30;")
mysql_audit=$("${mysql_base[@]}" -e \
  "SELECT COUNT(*) FROM \`${restored}\`.audit_conns WHERE peer_id='target' AND from_peer='source' AND conn_id=42;")
mysql_build=$("${mysql_base[@]}" -e \
  "SELECT COUNT(*) FROM \`${restored}\`.client_build_tasks WHERE status='pending' AND filename='test-client.exe';")
test "$mysql_policy" = "1"
test "$mysql_audit" = "1"
test "$mysql_build" = "1"
echo "MySQL 8.4 logical backup and recovery checks passed."

export PGPASSWORD=test-only-password
pg_base=(psql --host=127.0.0.1 --port=5432 --username=rustdesk --no-psqlrc --tuples-only --no-align)
pg_dump --host=127.0.0.1 --port=5432 --username=rustdesk \
  --no-owner --no-acl --format=custom \
  --file=/tmp/rustdesk-postgres-backup.dump rustdesk_upgrade
test -s /tmp/rustdesk-postgres-backup.dump
createdb --host=127.0.0.1 --port=5432 --username=rustdesk "$restored"
pg_restore --host=127.0.0.1 --port=5432 --username=rustdesk \
  --no-owner --no-acl --exit-on-error \
  --dbname="$restored" /tmp/rustdesk-postgres-backup.dump
pg_policy=$("${pg_base[@]}" --dbname="$restored" -c \
  "SELECT COUNT(*) FROM device_policies WHERE name='Legacy policy' AND access_mode='view' AND trust_duration_days=30;")
pg_audit=$("${pg_base[@]}" --dbname="$restored" -c \
  "SELECT COUNT(*) FROM audit_conns WHERE peer_id='target' AND from_peer='source' AND conn_id=42;")
pg_build=$("${pg_base[@]}" --dbname="$restored" -c \
  "SELECT COUNT(*) FROM client_build_tasks WHERE status='pending' AND filename='test-client.exe';")
test "$pg_policy" = "1"
test "$pg_audit" = "1"
test "$pg_build" = "1"
echo "PostgreSQL 16 logical backup and recovery checks passed."
