# Database Migrations

## Prerequisites

- MySQL 8.0+
- `MYSQL_TEST_DSN` env var for tests (e.g. `root:password@tcp(127.0.0.1:3306)/uas_test?multiStatements=true`)

## Commands

```bash
# Apply all pending migrations
go run cmd/migrate/main.go -cmd up

# Rollback one version
go run cmd/migrate/main.go -cmd down

# Rollback to specific version
go run cmd/migrate/main.go -cmd force -version <N> && go run cmd/migrate/main.go -cmd up

# Create new migration pair
go run cmd/migrate/main.go -cmd create -name <description>

# Check status
go run cmd/migrate/main.go -cmd status

# List migration files
go run cmd/migrate/main.go -cmd list

# Validate migration files
go run cmd/migrate/main.go -cmd validate

# Run migration tests (requires MySQL at $MYSQL_TEST_DSN)
MYSQL_TEST_DSN="root:password@tcp(127.0.0.1:3306)/uas_test?multiStatements=true" go test ./internal/migrations/ -run TestMigrationFullCycle -v
```

## Migration Rollback Runbook

### 0001 — create_users_table (Legacy)
- **Up**: Creates legacy `users` table.
- **Down**: `DROP TABLE users;`
- **Rollback risk**: Low — this table is legacy and not used by current code.
- **Data loss**: Destroys legacy user records. Ensure data migrated first.

### 0002 — align_auth_schema
- **Up**: Creates 9 tables matching GORM models. Migrates data from legacy `users` → `user_models`.
- **Down**: `DROP TABLE` for all 9 tables.
- **Rollback risk**: High — drops all current data. Only roll back in dev/staging.
- **Data loss**: Complete. Take a DB snapshot before rolling back.
- **Procedure**:
  1. `go run cmd/migrate/main.go -cmd down` (one step)
  2. Verify with `go run cmd/migrate/main.go -cmd status`
  3. If dirty, `go run cmd/migrate/main.go -cmd force -version 1`

### 0003 — baseline_v1
- **Up**: Adds `department_id` columns to `auth_models`/`password_histories`, NOT NULL constraints, composite indexes.
- **Down**: Reverses all changes — drops columns and indexes, restores NULLability.
- **Rollback risk**: Low to Moderate. Columns are `NULL`-compatible so existing data is safe.
- **Data loss**: None for data. Indexes must be rebuilt on re-apply.
- **Procedure**:
  1. `go run cmd/migrate/main.go -cmd down`
  2. Verify columns removed: `SHOW CREATE TABLE auth_models;`
  3. If migration is dirty: `go run cmd/migrate/main.go -cmd force -version 2` then re-apply `up`

## Migration File Format

Files are named `{version}_{description}.{direction}.sql` in `migrations/sql/`:
- `0001_create_users_table.up.sql`
- `0001_create_users_table.down.sql`

Each version MUST have both up and down files. Version numbers are sequential integers starting from 1.
