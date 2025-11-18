# Security Configuration Guide

This document explains the security-related configuration options in the UAS authentication microservice.

## Cookie Security Settings

### `COOKIE_DOMAIN`
**Type:** String  
**Default:** `` (empty)  
**Description:** Sets the domain for which the cookie is valid. Leave empty for the current domain only.

**Examples:**
- `.example.com` - Cookie valid for all subdomains
- `api.example.com` - Cookie valid for specific subdomain
- `` (empty) - Cookie valid only for the exact domain

**Security Implications:**
- Narrower domain reduces cookie leakage to other domains
- Leave empty unless you specifically need cross-subdomain authentication

---

### `COOKIE_SECURE`
**Type:** Boolean  
**Default:** `true`  
**Description:** Restricts cookie transmission to HTTPS connections only.

**Recommended:**
- `true` - Production environments (HTTPS required)
- `false` - Development with HTTP only (NOT for production)

**Security Implications:**
- `true` prevents cookie transmission over unencrypted connections
- `false` allows cookies over HTTP (vulnerable to man-in-the-middle attacks)
- Always use `true` in production

---

### `COOKIE_SAMESITE`
**Type:** String  
**Default:** `Strict`  
**Allowed Values:** `Strict`, `Lax`

**Description:** Prevents Cross-Site Request Forgery (CSRF) attacks by controlling when cookies are sent with cross-site requests.

**Options:**
- `Strict` - Cookie only sent in same-site requests (most secure)
- `Lax` - Cookie sent with safe cross-site requests (top-level navigation)

**Recommended:**
- `Strict` - Default for most applications
- `Lax` - If you need cookies in cross-site navigation

**Security Implications:**
- `Strict` provides maximum CSRF protection
- `Lax` allows some legitimate cross-site scenarios
- Prevents unauthorized form submissions from other sites

---

## Failed Login Protection

### `MAX_FAILED_ATTEMPTS`
**Type:** Integer  
**Default:** `5`  
**Unit:** Number of failed attempts  
**Description:** Number of consecutive failed login attempts before account lockout.

**Recommended:** `5-10`
- `3-5` - Strict (may lock legitimate users)
- `5-10` - Balanced
- `10+` - Lenient (allows more brute force attempts)

**Security Implications:**
- Lower values increase protection but may inconvenience users
- Higher values allow more brute force attempts
- Combined with progressive delays for effectiveness

---

### `ACCOUNT_LOCK_MINUTES`
**Type:** Integer  
**Default:** `15`  
**Unit:** Minutes  
**Description:** Duration for which an account is locked after exceeding max failed attempts.

**Recommended:** `15-30`
- `5-10` - Short (users can retry quickly)
- `15-30` - Balanced (slows down attackers)
- `60+` - Long (stronger protection, may frustrate users)

**Security Implications:**
- Longer lockout periods provide stronger brute force protection
- Shorter periods reduce user friction
- Automatic unlock prevents permanent lockout

**Lockout Behavior:**
1. User exceeds `MAX_FAILED_ATTEMPTS` failed logins
2. Account locked for `ACCOUNT_LOCK_MINUTES`
3. Failed login counter resets after lock duration
4. Successful login clears counter immediately

---

### Progressive Delay System

After failed login attempt, the following delays are applied before allowing next attempt:

1. **1st failure:** 0 seconds (immediate)
2. **2nd failure:** 2 seconds
3. **3rd failure:** 5 seconds
4. **4th failure:** 15 seconds
5. **5th+ failures:** 60 seconds + Account Lockout

This system significantly slows down brute force attacks without requiring configuration.

---

## Rate Limiting Settings

### `LOGIN_RATE_LIMIT_MINUTES`
**Type:** Integer  
**Default:** `1`  
**Unit:** Rate limit window in minutes  
**Description:** Time window for login rate limiting (5 requests per this window).

**Effective Limit:** 5 login requests per `LOGIN_RATE_LIMIT_MINUTES` per IP address

**Recommended:** `1-5`
- `1` - Strict (5 attempts per minute)
- `5` - Lenient (5 attempts per 5 minutes)

**Examples:**
- `LOGIN_RATE_LIMIT_MINUTES=1`: Allow 5 login attempts per minute per IP
- `LOGIN_RATE_LIMIT_MINUTES=5`: Allow 5 login attempts per 5 minutes per IP

**Security Implications:**
- Protects against distributed login brute force
- Per-IP limiting prevents account enumeration
- Combined with account lockout for defense-in-depth

---

### `OTP_RATE_LIMIT_MINUTES`
**Type:** Integer  
**Default:** `5`  
**Unit:** Rate limit window in minutes  
**Description:** Time window for OTP operations rate limiting.

**Effective Limits:**
- **OTP Send:** 3 requests per `OTP_RATE_LIMIT_MINUTES` per phone number
- **OTP Verify:** 5 requests per `OTP_RATE_LIMIT_MINUTES` per phone number

**Recommended:** `5-10`
- `5` - Balanced (protects against abuse while allowing retries)
- `10` - Lenient (allows more retry attempts)

**Examples:**
- `OTP_RATE_LIMIT_MINUTES=5`: 
  - 3 OTP sends per 5 minutes
  - 5 OTP verify attempts per 5 minutes per phone

**Security Implications:**
- Prevents SMS spam/abuse
- Protects against OTP brute force
- Limits toll fraud potential

---

## Rate Limiting Configuration Summary

| Endpoint | Limit | Identifier | Window |
|----------|-------|------------|--------|
| Register | 5 req/min | Per IP | 1 minute |
| Login | 5 req/min | Per IP | 1 minute |
| OTP Send | 3 req | Per phone | `LOGIN_RATE_LIMIT_MINUTES` |
| OTP Verify | 5 req | Per phone | `LOGIN_RATE_LIMIT_MINUTES` |

---

## Environment-Specific Recommendations

### Development
```bash
ENV=development
COOKIE_SECURE=false
COOKIE_DOMAIN=localhost
MAX_FAILED_ATTEMPTS=10
ACCOUNT_LOCK_MINUTES=1
LOGIN_RATE_LIMIT_MINUTES=10
OTP_RATE_LIMIT_MINUTES=10
```

### Staging
```bash
ENV=staging
COOKIE_SECURE=true
COOKIE_DOMAIN=.staging.example.com
MAX_FAILED_ATTEMPTS=7
ACCOUNT_LOCK_MINUTES=10
LOGIN_RATE_LIMIT_MINUTES=5
OTP_RATE_LIMIT_MINUTES=5
```

### Production
```bash
ENV=prod
COOKIE_SECURE=true
COOKIE_DOMAIN=.example.com
MAX_FAILED_ATTEMPTS=5
ACCOUNT_LOCK_MINUTES=15
LOGIN_RATE_LIMIT_MINUTES=1
OTP_RATE_LIMIT_MINUTES=5
```

---

## Security Best Practices

### 1. Secrets Management
- Generate strong JWT secrets using: `openssl rand -base64 32`
- Use environment variables, not hardcoded values
- Rotate secrets periodically
- Consider using HashiCorp Vault for secrets in production

### 2. Cookie Configuration
- Always set `COOKIE_SECURE=true` in production
- Use `COOKIE_SAMESITE=Strict` unless you have specific cross-site requirements
- Set `COOKIE_DOMAIN` to the minimum necessary scope

### 3. Brute Force Protection
- Combine account lockout with progressive delays
- Monitor failed login attempts in logs
- Consider implementing IP-level blocking for repeated attacks

### 4. Rate Limiting
- Adjust limits based on expected legitimate traffic
- Monitor rate limit headers in responses
- Use stricter limits during security incidents

### 5. Monitoring & Alerts
- Log all failed login attempts
- Alert on repeated lockouts
- Monitor rate limit violations
- Track unusual access patterns

---

## Response Headers

When rate limits are enforced, the following headers are included in responses:

```
RateLimit-Remaining: 3
RateLimit-RetryAfter: 45
```

**RateLimit-Remaining:** Number of requests remaining in current window  
**RateLimit-RetryAfter:** Seconds to wait before next request is allowed

---

## Troubleshooting

### "Account temporarily locked due to too many failed login attempts"
- Wait for `ACCOUNT_LOCK_MINUTES` duration
- Or contact administrator to unlock account
- Failed attempt counter resets after lock duration

### "Too Many Requests" (429 status)
- Rate limit exceeded for this endpoint
- Check `RateLimit-RetryAfter` header
- Wait before retrying
- Review `LOGIN_RATE_LIMIT_MINUTES` and `OTP_RATE_LIMIT_MINUTES`

### Progressive delays between login attempts
- Intentional security feature
- Delays increase with each failed attempt
- Designed to slow brute force attacks
- Resets after successful login

---

## Related Documentation

- [Security Configuration](./SECURITY_CONFIG.md)
- [Phase 1: Critical Security Fixes](./docs/PHASE_1_SECURITY.md)
- [Phase 2: High Priority Security Fixes](./docs/PHASE_2_SECURITY.md)

