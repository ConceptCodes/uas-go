# UAS Deployment Operations Hardening

This runbook focuses on production readiness for alerts, dashboards, backups, rollback, and migration safety.

## 1. Monitoring and Alerts

### Enable Metrics in API
The API now exposes Prometheus metrics at `/metrics` when `ENABLE_METRICS=true`.

### Prometheus Alert Rules
- File: `deploy/monitoring/prometheus-alerts.yml`
- Includes:
  - `UASInstanceDown`
  - `UASHigh5xxRate`
  - `UASHighP95Latency`
  - `UASRateLimitSpike`
  - `UASNoLoginActivity`

### Dashboard
- File: `deploy/monitoring/grafana-dashboard.json`
- Panels:
  - API request rate
  - API 5xx ratio
  - p95 latency
  - rate-limit error spikes
  - login attempts
  - registration outcomes

## 2. Backup Strategy

### Backup Frequency
- Full MySQL backup at least daily.
- Keep 7 daily backups minimum for rollback safety.
- For critical environments, add hourly binlog shipping.

### Backup Command
```bash
make backup-db
```

Optional target path:
```bash
bash scripts/db_backup.sh backups/uas_YYYYMMDD_HHMMSS.sql.gz
```

### Restore Command
```bash
make restore-db FILE=backups/uas_YYYYMMDD_HHMMSS.sql.gz
```

### Backup Verification
Run weekly restore drills in a staging database:
1. Restore latest backup into clean staging DB.
2. Run `make migrate-status`.
3. Run `make test-smoke-auth` against restored data in isolated env.

## 3. Rollback Plan

### App Rollback
1. Keep previous release artifact/image tagged and deployable.
2. Roll back traffic to previous release if alerts fire after deploy.
3. Validate:
   - `/api/v1/health/alive`
   - `/api/v1/health/status`
   - key auth path using smoke test

### DB Rollback Options
- Preferred: forward-fix migrations when possible.
- If immediate rollback is required:
  1. Stop write traffic.
  2. Roll back app release.
  3. If needed, execute one migration step down:
     ```bash
     make migrate-down
     ```
  4. If schema/data divergence occurred, restore from backup:
     ```bash
     make restore-db FILE=backups/<snapshot>.sql.gz
     ```

## 4. Migration Strategy

### Pre-Deploy (Required)
1. Validate migration files:
   ```bash
   make migrate-validate
   ```
2. Check current migration state:
   ```bash
   make migrate-status
   ```
3. Take backup immediately before migration:
   ```bash
   make backup-db
   ```
4. Apply migrations:
   ```bash
   make migrate-up
   ```
5. Run smoke test:
   ```bash
   make test-smoke-auth
   ```

### Deployment Sequence
1. Deploy API binary/image.
2. Confirm readiness endpoint is healthy.
3. Watch dashboard + alerts for 30 minutes.

### Post-Deploy
- Confirm alert silence for critical rules.
- Confirm expected login/registration traffic in dashboard.
- Confirm no unexpected `5xx` or sustained latency increases.

## 5. Ownership and Escalation

- Primary owner: backend/on-call engineer.
- Escalate immediately when:
  - `UASInstanceDown` fires > 2m.
  - `UASHigh5xxRate` > 10m.
  - `UASHighP95Latency` > 15m.
- Attach during incident:
  - current deploy version
  - last migration version
  - latest backup file used/available

