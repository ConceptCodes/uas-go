#!/usr/bin/env bash
set -euo pipefail

MYSQL_HOST="${DB_HOST:-127.0.0.1}"
MYSQL_PORT="${DB_PORT:-3306}"
MYSQL_USER="${DB_USER:-uas}"
MYSQL_PASS="${DB_PASS:-uaspass}"
MYSQL_DB="${DB_NAME:-uas}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"
TIMESTAMP="$(date +"%Y%m%d_%H%M%S")"
OUTPUT_FILE="${1:-${BACKUP_DIR}/${MYSQL_DB}_${TIMESTAMP}.sql.gz}"

mkdir -p "${BACKUP_DIR}"

if ! command -v mysqldump >/dev/null 2>&1; then
  echo "mysqldump is required but not installed" >&2
  exit 1
fi

echo "Creating backup: ${OUTPUT_FILE}"
MYSQL_PWD="${MYSQL_PASS}" mysqldump \
  --host="${MYSQL_HOST}" \
  --port="${MYSQL_PORT}" \
  --user="${MYSQL_USER}" \
  --single-transaction \
  --quick \
  --routines \
  --triggers \
  "${MYSQL_DB}" | gzip > "${OUTPUT_FILE}"

echo "Backup completed: ${OUTPUT_FILE}"
