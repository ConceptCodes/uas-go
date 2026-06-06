package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func skipIfNoDB(t *testing.T) {
	if os.Getenv("MYSQL_TEST_DSN") == "" {
		t.Skip("Skipping: set MYSQL_TEST_DSN to run migration tests (e.g. root:password@tcp(127.0.0.1:3306)/uas_test?multiStatements=true)")
	}
}

func openTestDB(t *testing.T) *sql.DB {
	dsn := os.Getenv("MYSQL_TEST_DSN")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test DB: %v", err)
	}
	return db
}

func newTestMigrator(t *testing.T, db *sql.DB) *migrate.Migrate {
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		t.Fatalf("Failed to create mysql driver: %v", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://../../migrations/sql"),
		"mysql",
		driver,
	)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}
	t.Cleanup(func() {
		m.Close()
	})
	return m
}

func dropAllTables(t *testing.T, db *sql.DB) {
	_, err := db.Exec(`
		DROP TABLE IF EXISTS audit_logs;
		DROP TABLE IF EXISTS sessions;
		DROP TABLE IF EXISTS security_events;
		DROP TABLE IF EXISTS password_histories;
		DROP TABLE IF EXISTS department_configs;
		DROP TABLE IF EXISTS auth_models;
		DROP TABLE IF EXISTS department_roles;
		DROP TABLE IF EXISTS user_models;
		DROP TABLE IF EXISTS department_models;
		DROP TABLE IF EXISTS users;
		DROP TABLE IF EXISTS schema_migrations;
	`)
	if err != nil {
		t.Logf("Warning dropping tables: %v", err)
	}
}

func TestMigrationFullCycle(t *testing.T) {
	skipIfNoDB(t)
	db := openTestDB(t)
	dropAllTables(t, db)
	m := newTestMigrator(t, db)

	// Migrate up to latest (version 3)
	err := m.Up()
	if err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Up migration failed: %v", err)
	}

	// Assert all expected tables exist
	expectedTables := []string{
		"department_models", "user_models", "department_roles",
		"auth_models", "department_configs", "password_histories",
		"security_events", "sessions", "audit_logs",
	}
	for _, table := range expectedTables {
		var exists bool
		err := db.QueryRow(
			"SELECT COUNT(*) > 0 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?",
			table,
		).Scan(&exists)
		if err != nil || !exists {
			t.Errorf("Expected table %s to exist after migration", table)
		}
	}

	// Assert department_id column exists on auth_models
	var hasCol bool
	err = db.QueryRow(
		"SELECT COUNT(*) > 0 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'auth_models' AND column_name = 'department_id'",
	).Scan(&hasCol)
	if err != nil || !hasCol {
		t.Error("Expected auth_models.department_id to exist after 0003 migration")
	}

	// Assert department_id column exists on password_histories
	err = db.QueryRow(
		"SELECT COUNT(*) > 0 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'password_histories' AND column_name = 'department_id'",
	).Scan(&hasCol)
	if err != nil || !hasCol {
		t.Error("Expected password_histories.department_id to exist after 0003 migration")
	}

	// Assert NOT NULL on user_models.email
	var nullable string
	err = db.QueryRow(
		"SELECT is_nullable FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'user_models' AND column_name = 'email'",
	).Scan(&nullable)
	if err != nil || nullable != "NO" {
		t.Errorf("Expected user_models.email to be NOT NULL, got nullable=%s", nullable)
	}

	// Assert composite indexes exist
	expectedIndexes := []string{
		"idx_sessions_department_user",
		"idx_sessions_expires_at",
		"idx_security_events_department_created",
		"idx_auth_models_user_type",
		"idx_password_histories_user_department",
		"idx_department_roles_user",
		"idx_audit_logs_department_timestamp",
	}
	for _, idx := range expectedIndexes {
		var exists bool
		err := db.QueryRow(
			"SELECT COUNT(*) > 0 FROM information_schema.statistics WHERE table_schema = DATABASE() AND index_name = ?",
			idx,
		).Scan(&exists)
		if err != nil || !exists {
			t.Errorf("Expected index %s to exist", idx)
		}
	}

	// Verify version
	version, dirty, err := m.Version()
	if err != nil {
		t.Fatalf("Failed to get version: %v", err)
	}
	if dirty {
		t.Error("Migration should not be dirty after full up")
	}
	if version != 3 {
		t.Errorf("Expected version 3, got %d", version)
	}

	// Migrate down (step backward 3 times to undo all migrations)
	for i := 0; i < 3; i++ {
		err := m.Steps(-1)
		if err != nil {
			t.Fatalf("Step -1 (iteration %d) failed: %v", i, err)
		}
	}

	// Verify no tables remain
	for _, table := range expectedTables {
		var exists bool
		err := db.QueryRow(
			"SELECT COUNT(*) > 0 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?",
			table,
		).Scan(&exists)
		if err == nil && exists {
			t.Errorf("Expected table %s to be dropped after down migration", table)
		}
	}

	// Migrate back up to verify idempotency
	dropAllTables(t, db)

	m2 := newTestMigrator(t, db)
	err = m2.Up()
	if err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Second up migration failed: %v", err)
	}
}
