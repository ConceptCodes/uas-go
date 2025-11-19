package migrations

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"uas/config"
	"uas/pkg/logger"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog"
)

type MigrationManager struct {
	migrate *migrate.Migrate
	logger  *zerolog.Logger
	db      *sql.DB
}

type MigrationInfo struct {
	Version      uint
	Dirty        bool
	Current      string
	Target       string
	DatabaseName string
}

type MigrationFile struct {
	Version     uint
	Name        string
	Direction   string // "up" or "down"
	Content     string
	CreatedAt   time.Time
	Description string
}

// NewMigrationManager creates a new migration manager instance
func NewMigrationManager() (*MigrationManager, error) {
	log := logger.New()

	// Create migrations directory if it doesn't exist
	migrationsDir := "migrations/sql"
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		log.Error().Err(err).Msg("Failed to create migrations directory")
		return nil, fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// Build database connection string
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true",
		config.AppConfig.DbUser,
		config.AppConfig.DbPass,
		config.AppConfig.DbHost,
		config.AppConfig.DbPort,
		config.AppConfig.DbName,
	)

	// Add TLS parameters if enabled
	if config.AppConfig.EnableMySQLTLS {
		dsn += "&tls=true"
	}

	// Open database connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Error().Err(err).Msg("Failed to open database connection")
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		log.Error().Err(err).Msg("Failed to ping database")
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create database driver instance
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create database driver")
		return nil, fmt.Errorf("failed to create database driver: %w", err)
	}

	// Create migration instance
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsDir),
		"mysql",
		driver,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create migration instance")
		return nil, fmt.Errorf("failed to create migration instance: %w", err)
	}

	return &MigrationManager{
		migrate: m,
		logger:  log,
		db:      db,
	}, nil
}

// Up runs all pending migrations
func (mm *MigrationManager) Up() error {
	mm.logger.Info().Msg("Running database migrations...")

	err := mm.migrate.Up()
	if err != nil && err != migrate.ErrNoChange {
		mm.logger.Error().Err(err).Msg("Failed to run migrations")
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if err == migrate.ErrNoChange {
		mm.logger.Info().Msg("No pending migrations to run")
	} else {
		mm.logger.Info().Msg("Migrations completed successfully")
	}

	return nil
}

// Down rolls back the most recent migration
func (mm *MigrationManager) Down() error {
	mm.logger.Info().Msg("Rolling back last migration...")

	err := mm.migrate.Steps(-1)
	if err != nil && err != migrate.ErrNoChange {
		mm.logger.Error().Err(err).Msg("Failed to rollback migration")
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	if err == migrate.ErrNoChange {
		mm.logger.Info().Msg("No migrations to rollback")
	} else {
		mm.logger.Info().Msg("Migration rollback completed successfully")
	}

	return nil
}

// MigrateToVersion migrates to a specific version
func (mm *MigrationManager) MigrateToVersion(version uint) error {
	mm.logger.Info().Uint("version", version).Msg("Migrating to specific version...")

	err := mm.migrate.Migrate(version)
	if err != nil && err != migrate.ErrNoChange {
		mm.logger.Error().Err(err).Uint("version", version).Msg("Failed to migrate to version")
		return fmt.Errorf("failed to migrate to version %d: %w", version, err)
	}

	if err == migrate.ErrNoChange {
		mm.logger.Info().Uint("version", version).Msg("Already at target version")
	} else {
		mm.logger.Info().Uint("version", version).Msg("Migration to version completed successfully")
	}

	return nil
}

// GetStatus returns the current migration status
func (mm *MigrationManager) GetStatus() (*MigrationInfo, error) {
	version, dirty, err := mm.migrate.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return nil, fmt.Errorf("failed to get migration version: %w", err)
	}

	info := &MigrationInfo{
		Version:      version,
		Dirty:        dirty,
		DatabaseName: config.AppConfig.DbName,
	}

	if err == migrate.ErrNilVersion {
		info.Current = "No migrations applied"
		info.Target = "Latest"
	} else {
		info.Current = fmt.Sprintf("%d", version)
		info.Target = "Latest"
	}

	return info, nil
}

// ListMigrations returns a list of all migration files
func (mm *MigrationManager) ListMigrations() ([]MigrationFile, error) {
	migrationsDir := "migrations/sql"

	var files []MigrationFile

	err := filepath.WalkDir(migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		filename := d.Name()
		if !strings.HasSuffix(filename, ".sql") {
			return nil
		}

		// Parse filename to extract version and direction
		parts := strings.Split(strings.TrimSuffix(filename, ".sql"), "_")
		if len(parts) < 2 {
			return nil
		}

		var version uint
		if _, err := fmt.Sscanf(parts[0], "%d", &version); err != nil {
			return nil
		}

		direction := "up"
		if strings.Contains(filename, "down") {
			direction = "down"
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Get file info
		fileInfo, err := d.Info()
		if err != nil {
			return err
		}

		// Extract description from filename
		description := strings.Join(parts[1:], " ")
		if direction == "down" {
			description = strings.TrimSuffix(description, " down")
		} else {
			description = strings.TrimSuffix(description, " up")
		}

		files = append(files, MigrationFile{
			Version:     version,
			Name:        filename,
			Direction:   direction,
			Content:     string(content),
			CreatedAt:   fileInfo.ModTime(),
			Description: description,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list migration files: %w", err)
	}

	// Sort by version
	sort.Slice(files, func(i, j int) bool {
		return files[i].Version < files[j].Version
	})

	return files, nil
}

// ForceVersion sets the migration version without running migrations
func (mm *MigrationManager) ForceVersion(version uint) error {
	mm.logger.Info().Uint("version", version).Msg("Forcing migration version...")

	if err := mm.migrate.Force(int(version)); err != nil {
		mm.logger.Error().Err(err).Uint("version", version).Msg("Failed to force migration version")
		return fmt.Errorf("failed to force migration version %d: %w", version, err)
	}

	mm.logger.Info().Uint("version", version).Msg("Migration version forced successfully")
	return nil
}

// Drop drops the entire database schema
func (mm *MigrationManager) Drop() error {
	mm.logger.Warn().Msg("Dropping all database tables...")

	if err := mm.migrate.Drop(); err != nil {
		mm.logger.Error().Err(err).Msg("Failed to drop database")
		return fmt.Errorf("failed to drop database: %w", err)
	}

	mm.logger.Info().Msg("Database dropped successfully")
	return nil
}

// Close closes the migration manager and database connection
func (mm *MigrationManager) Close() error {
	var sourceErr, databaseErr error

	if mm.migrate != nil {
		sourceErr, databaseErr = mm.migrate.Close()
	}

	if mm.db != nil {
		if err := mm.db.Close(); err != nil {
			databaseErr = err
		}
	}

	if sourceErr != nil {
		return fmt.Errorf("source error: %w", sourceErr)
	}

	if databaseErr != nil {
		return fmt.Errorf("database error: %w", databaseErr)
	}

	return nil
}

// CreateMigration creates a new migration file pair (up and down)
func (mm *MigrationManager) CreateMigration(name string) (string, error) {
	migrationsDir := "migrations/sql"

	// Get the next version number
	files, err := mm.ListMigrations()
	if err != nil {
		return "", fmt.Errorf("failed to list existing migrations: %w", err)
	}

	var nextVersion uint = 1
	if len(files) > 0 {
		// Find the highest version
		for _, file := range files {
			if file.Version > nextVersion {
				nextVersion = file.Version
			}
		}
		nextVersion++
	}

	// Create up migration file
	upFileName := fmt.Sprintf("%04d_%s.up.sql", nextVersion, name)
	upFilePath := filepath.Join(migrationsDir, upFileName)
	upContent := fmt.Sprintf(`-- Migration: %s
-- Version: %d
-- Created: %s
-- Description: %s

-- Add your UP migration SQL here

`, name, nextVersion, time.Now().Format("2006-01-02 15:04:05"), name)

	if err := os.WriteFile(upFilePath, []byte(upContent), 0644); err != nil {
		return "", fmt.Errorf("failed to create up migration file: %w", err)
	}

	// Create down migration file
	downFileName := fmt.Sprintf("%04d_%s.down.sql", nextVersion, name)
	downFilePath := filepath.Join(migrationsDir, downFileName)
	downContent := fmt.Sprintf(`-- Migration: %s
-- Version: %d
-- Created: %s
-- Description: %s (Rollback)

-- Add your DOWN migration SQL here

`, name, nextVersion, time.Now().Format("2006-01-02 15:04:05"), name)

	if err := os.WriteFile(downFilePath, []byte(downContent), 0644); err != nil {
		// Clean up the up file if down file creation fails
		os.Remove(upFilePath)
		return "", fmt.Errorf("failed to create down migration file: %w", err)
	}

	mm.logger.Info().
		Str("name", name).
		Uint("version", nextVersion).
		Str("up_file", upFileName).
		Str("down_file", downFileName).
		Msg("Migration files created successfully")

	return upFileName, nil
}

// ValidateMigrations checks if migration files are valid
func (mm *MigrationManager) ValidateMigrations() error {
	files, err := mm.ListMigrations()
	if err != nil {
		return fmt.Errorf("failed to list migrations for validation: %w", err)
	}

	// Group migrations by version
	versionMap := make(map[uint][]MigrationFile)
	for _, file := range files {
		versionMap[file.Version] = append(versionMap[file.Version], file)
	}

	// Validate each version
	for version, versionFiles := range versionMap {
		if len(versionFiles) != 2 {
			return fmt.Errorf("version %d should have exactly 2 files (up and down), found %d", version, len(versionFiles))
		}

		var hasUp, hasDown bool
		for _, file := range versionFiles {
			if file.Direction == "up" {
				hasUp = true
			} else if file.Direction == "down" {
				hasDown = true
			}
		}

		if !hasUp {
			return fmt.Errorf("version %d is missing up migration", version)
		}
		if !hasDown {
			return fmt.Errorf("version %d is missing down migration", version)
		}

		// Validate SQL syntax (basic check)
		for _, file := range versionFiles {
			if strings.TrimSpace(file.Content) == "" {
				return fmt.Errorf("migration file %s is empty", file.Name)
			}
		}
	}

	mm.logger.Info().Msg("All migration files are valid")
	return nil
}
