#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <backup_file.sql|backup_file.sql.gz>" >&2
  exit 1
fi

INPUT_FILE="$1"
MYSQL_HOST="${DB_HOST:-127.0.0.1}"
MYSQL_PORT="${DB_PORT:-3306}"
MYSQL_USER="${DB_USER:-uas}"
MYSQL_PASS="${DB_PASS:-uaspass}"
MYSQL_DB="${DB_NAME:-uas}"

if [[ ! -f "${INPUT_FILE}" ]]; then
  echo "Backup file not found: ${INPUT_FILE}" >&2
  exit 1
fi

if ! command -v mysql >/dev/null 2>&1; then
  echo "mysql client is required but not installed" >&2
  exit 1
fi

echo "Restoring ${INPUT_FILE} into ${MYSQL_DB} on ${MYSQL_HOST}:${MYSQL_PORT}"
if [[ "${INPUT_FILE}" == *.gz ]]; then
  gzip -dc "${INPUT_FILE}" | MYSQL_PWD="${MYSQL_PASS}" mysql \
    --host="${MYSQL_HOST}" \
    --port="${MYSQL_PORT}" \
    --user="${MYSQL_USER}" \
    "${MYSQL_DB}"
else
  MYSQL_PWD="${MYSQL_PASS}" mysql \
    --host="${MYSQL_HOST}" \
    --port="${MYSQL_PORT}" \
    --user="${MYSQL_USER}" \
    "${MYSQL_DB}" < "${INPUT_FILE}"
fi

echo "Restore completed"
