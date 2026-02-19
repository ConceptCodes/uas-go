#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_HOST="${API_HOST:-127.0.0.1}"
API_PORT="${API_PORT:-18080}"
API_BASE="http://${API_HOST}:${API_PORT}/api/v1"

MYSQL_CONTAINER="${MYSQL_CONTAINER:-uas-mysql}"
REDIS_CONTAINER="${REDIS_CONTAINER:-uas-redis}"
MYSQL_IMAGE="${MYSQL_IMAGE:-mysql:8}"
REDIS_IMAGE="${REDIS_IMAGE:-redis:7-alpine}"

MYSQL_DB="${MYSQL_DB:-uas}"
MYSQL_USER="${MYSQL_USER:-uas}"
MYSQL_PASS="${MYSQL_PASS:-uaspass}"

EMAIL="smoke.$(date +%s)@example.com"
PHONE="+1555000$(printf '%04d' "$((RANDOM % 10000))")"
PASSWORD_ONE="S3curePass!234"
PASSWORD_TWO="N3wSecure!567"
TENANT_ID="smoke-tenant-$(date +%s)"
TENANT_NAME="Smoke Tenant"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

wait_for_http() {
  local url="$1"
  for _ in $(seq 1 60); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "Timed out waiting for $url" >&2
  return 1
}

ensure_mysql() {
  if docker ps -a --format '{{.Names}}' | grep -qx "${MYSQL_CONTAINER}"; then
    docker start "${MYSQL_CONTAINER}" >/dev/null
  else
    docker run -d \
      --name "${MYSQL_CONTAINER}" \
      -e MYSQL_ROOT_PASSWORD=rootpass \
      -e MYSQL_DATABASE="${MYSQL_DB}" \
      -e MYSQL_USER="${MYSQL_USER}" \
      -e MYSQL_PASSWORD="${MYSQL_PASS}" \
      -p 3306:3306 \
      "${MYSQL_IMAGE}" >/dev/null
  fi

  for _ in $(seq 1 60); do
    if docker exec "${MYSQL_CONTAINER}" mysqladmin ping -h 127.0.0.1 -u"${MYSQL_USER}" -p"${MYSQL_PASS}" --silent >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "MySQL did not become ready in time" >&2
  return 1
}

ensure_redis() {
  if docker ps -a --format '{{.Names}}' | grep -qx "${REDIS_CONTAINER}"; then
    docker start "${REDIS_CONTAINER}" >/dev/null
  else
    docker run -d --name "${REDIS_CONTAINER}" -p 6379:6379 "${REDIS_IMAGE}" >/dev/null
  fi

  for _ in $(seq 1 60); do
    if docker exec "${REDIS_CONTAINER}" redis-cli ping 2>/dev/null | grep -q PONG; then
      return 0
    fi
    sleep 1
  done
  echo "Redis did not become ready in time" >&2
  return 1
}

require_cmd curl
require_cmd docker
require_cmd go
require_cmd awk
require_cmd sed

ensure_mysql
ensure_redis

# Clear transient rate-limit and lock keys to make repeated smoke runs deterministic.
RATE_KEYS="$(docker exec "${REDIS_CONTAINER}" redis-cli --scan --pattern 'rate_limit:*' | tr -d '\r')"
if [[ -n "${RATE_KEYS}" ]]; then
  while IFS= read -r key; do
    [[ -n "${key}" ]] && docker exec "${REDIS_CONTAINER}" redis-cli DEL "${key}" >/dev/null
  done <<< "${RATE_KEYS}"
fi

LOG_FILE="$(mktemp -t uas-smoke-api)"
HEADER_FILE="$(mktemp -t uas-smoke-headers.XXXXXX)"
BODY_FILE="$(mktemp -t uas-smoke-body.XXXXXX)"

cleanup() {
  if [[ -n "${API_PID:-}" ]] && kill -0 "${API_PID}" >/dev/null 2>&1; then
    kill "${API_PID}" >/dev/null 2>&1 || true
  fi
  rm -f "${HEADER_FILE}" "${BODY_FILE}"
  echo "API logs: ${LOG_FILE}"
}
trap cleanup EXIT

(
  cd "${ROOT_DIR}"
  HOST="${API_HOST}" \
  PORT="${API_PORT}" \
  DB_HOST=127.0.0.1 \
  DB_PORT=3306 \
  DB_USER="${MYSQL_USER}" \
  DB_PASS="${MYSQL_PASS}" \
  DB_NAME="${MYSQL_DB}" \
  REDIS_HOST=127.0.0.1 \
  REDIS_PORT=6379 \
  REDIS_PASSWORD="" \
  REDIS_DB=0 \
  EMAIL_PROVIDER=mock \
  EMAIL_FROM=noreply@example.com \
  RESEND_API_KEY=mock \
  RESEND_EMAIL_DOMAIN=example.com \
  COOKIE_SECURE=false \
  ACCESS_JWT_SECRET=12345678901234567890123456789012 \
  REFRESH_JWT_SECRET=12345678901234567890123456789012 \
  COOKIE_BLOCK_KEY=12345678901234567890123456789012 \
  COOKIE_HASH_KEY=12345678901234567890123456789012 \
  ENCRYPTION_KEY=12345678901234567890123456789012 \
  PASSWORD_RESET_BASE_URL="http://${API_HOST}:${API_PORT}/api/v1/users/credentials/reset-password" \
  go run main.go >"${LOG_FILE}" 2>&1
) &
API_PID=$!

wait_for_http "${API_BASE}/health/alive"
echo "Health check passed"

STATUS=$(curl -sS -D "${HEADER_FILE}" -o "${BODY_FILE}" -w "%{http_code}" \
  -X POST "${API_BASE}/tenants" \
  -H "Content-Type: application/json" \
  -d "{\"departmentName\":\"${TENANT_NAME}\",\"departmentId\":\"${TENANT_ID}\"}")
if [[ "${STATUS}" != "200" ]]; then
  echo "Tenant onboarding failed: ${STATUS}" >&2
  cat "${BODY_FILE}" >&2
  exit 1
fi

TENANT_AUTH="$(awk 'tolower($1)=="authorization:" {sub(/\r$/, "", $0); sub(/^[^:]+:[[:space:]]*/, "", $0); print $0; exit}' "${HEADER_FILE}")"
if [[ -z "${TENANT_AUTH}" ]]; then
  echo "Missing tenant authorization header" >&2
  cat "${HEADER_FILE}" >&2
  exit 1
fi
echo "Tenant onboarding passed"

STATUS=$(curl -sS -o "${BODY_FILE}" -w "%{http_code}" \
  -X POST "${API_BASE}/users/credentials/register" \
  -H "Content-Type: application/json" \
  -H "Authorization: ${TENANT_AUTH}" \
  -d "{\"name\":\"Smoke User\",\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD_ONE}\",\"phoneNumber\":\"${PHONE}\"}")
if [[ "${STATUS}" != "200" ]]; then
  echo "Register failed: ${STATUS}" >&2
  cat "${BODY_FILE}" >&2
  exit 1
fi
echo "Registration passed"

OTP="$(docker exec "${REDIS_CONTAINER}" redis-cli GET "otp:${EMAIL}" | tr -d '\r')"
if [[ -z "${OTP}" || "${OTP}" == "(nil)" ]]; then
  OTP="$(awk 'match($0, /"Otp":"[0-9]{6}"/) {otp=substr($0, RSTART+7, 6)} END {print otp}' "${LOG_FILE}")"
fi
if [[ -z "${OTP}" ]]; then
  echo "OTP not found in Redis or logs for ${EMAIL}" >&2
  exit 1
fi

STATUS=$(curl -sS -o "${BODY_FILE}" -w "%{http_code}" \
  -X POST "${API_BASE}/users/credentials/verify-email" \
  -H "Content-Type: application/json" \
  -H "Authorization: ${TENANT_AUTH}" \
  -d "{\"email\":\"${EMAIL}\",\"otp\":\"${OTP}\"}")
if [[ "${STATUS}" != "200" ]]; then
  echo "Verify email failed: ${STATUS}" >&2
  cat "${BODY_FILE}" >&2
  exit 1
fi
echo "Email verification passed"

STATUS=$(curl -sS -D "${HEADER_FILE}" -o "${BODY_FILE}" -w "%{http_code}" \
  -X POST "${API_BASE}/users/credentials/login" \
  -H "Content-Type: application/json" \
  -H "Authorization: ${TENANT_AUTH}" \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD_ONE}\"}")
if [[ "${STATUS}" != "200" ]]; then
  echo "Login failed: ${STATUS}" >&2
  cat "${BODY_FILE}" >&2
  exit 1
fi
REFRESH_TOKEN="$(awk 'tolower($1)=="x-jwt-token:" {sub(/\r$/, "", $0); sub(/^[^:]+:[[:space:]]*/, "", $0); print $0; exit}' "${HEADER_FILE}")"
if [[ -z "${REFRESH_TOKEN}" ]]; then
  echo "Refresh token header missing from login response" >&2
  cat "${HEADER_FILE}" >&2
  exit 1
fi
echo "Login passed"

STATUS=$(curl -sS -D "${HEADER_FILE}" -o "${BODY_FILE}" -w "%{http_code}" \
  -X POST "${API_BASE}/users/refresh-token" \
  -H "Authorization: ${TENANT_AUTH}" \
  -H "x-jwt-token: ${REFRESH_TOKEN}")
if [[ "${STATUS}" != "200" ]]; then
  echo "Refresh token failed: ${STATUS}" >&2
  cat "${BODY_FILE}" >&2
  exit 1
fi
echo "Refresh token passed"

STATUS=$(curl -sS -o "${BODY_FILE}" -w "%{http_code}" \
  -X POST "${API_BASE}/users/credentials/forgot-password" \
  -H "Content-Type: application/json" \
  -H "Authorization: ${TENANT_AUTH}" \
  -d "{\"email\":\"${EMAIL}\"}")
if [[ "${STATUS}" != "200" ]]; then
  echo "Forgot password failed: ${STATUS}" >&2
  cat "${BODY_FILE}" >&2
  exit 1
fi
echo "Forgot-password passed"

RESET_TOKEN="$(docker exec "${MYSQL_CONTAINER}" mysql -u"${MYSQL_USER}" -p"${MYSQL_PASS}" -D "${MYSQL_DB}" -Nse \
  "SELECT a.token FROM auth_models a JOIN user_models u ON a.user_id = u.id WHERE u.email='${EMAIL}' AND a.type='reset-password' ORDER BY a.created_at DESC LIMIT 1;" | tr -d '\r')"
if [[ -z "${RESET_TOKEN}" ]]; then
  echo "Reset token not found in database" >&2
  exit 1
fi

STATUS=$(curl -sS -o "${BODY_FILE}" -w "%{http_code}" \
  -X POST "${API_BASE}/users/credentials/reset-password?token=${RESET_TOKEN}" \
  -H "Content-Type: application/json" \
  -H "Authorization: ${TENANT_AUTH}" \
  -d "{\"password\":\"${PASSWORD_TWO}\"}")
if [[ "${STATUS}" != "200" ]]; then
  echo "Reset password failed: ${STATUS}" >&2
  cat "${BODY_FILE}" >&2
  exit 1
fi
echo "Password reset passed"

STATUS=$(curl -sS -D "${HEADER_FILE}" -o "${BODY_FILE}" -w "%{http_code}" \
  -X POST "${API_BASE}/users/credentials/login" \
  -H "Content-Type: application/json" \
  -H "Authorization: ${TENANT_AUTH}" \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD_TWO}\"}")
if [[ "${STATUS}" == "429" ]]; then
  RETRY_AFTER="$(awk 'tolower($1)=="ratelimit-retryafter:" {gsub(/\r/, "", $2); print $2; exit}' "${HEADER_FILE}")"
  if [[ -z "${RETRY_AFTER}" ]]; then
    RETRY_AFTER=12
  fi
  sleep "$((RETRY_AFTER + 1))"
  STATUS=$(curl -sS -D "${HEADER_FILE}" -o "${BODY_FILE}" -w "%{http_code}" \
    -X POST "${API_BASE}/users/credentials/login" \
    -H "Content-Type: application/json" \
    -H "Authorization: ${TENANT_AUTH}" \
    -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD_TWO}\"}")
fi
if [[ "${STATUS}" != "200" ]]; then
  echo "Login with new password failed: ${STATUS}" >&2
  cat "${BODY_FILE}" >&2
  exit 1
fi
echo "Login with new password passed"

echo "Auth smoke test completed successfully"
