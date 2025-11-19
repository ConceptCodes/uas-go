package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"uas/internal/migrations"
)

func main() {
	var (
		command = flag.String("cmd", "", "Migration command: up, down, create, status, force, drop, validate")
		version = flag.Uint("version", 0, "Target version for migrate/force commands")
		name    = flag.String("name", "", "Migration name for create command")
		help    = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help || *command == "" {
		showHelp()
		return
	}

	// Initialize migration manager
	migrationManager, err := migrations.NewMigrationManager()
	if err != nil {
		log.Fatalf("Failed to initialize migration manager: %v", err)
	}
	defer migrationManager.Close()

	// Execute command
	switch *command {
	case "up":
		if err := migrationManager.Up(); err != nil {
			log.Fatalf("Migration up failed: %v", err)
		}
		fmt.Println("Migration up completed successfully")

	case "down":
		if err := migrationManager.Down(); err != nil {
			log.Fatalf("Migration down failed: %v", err)
		}
		fmt.Println("Migration down completed successfully")

	case "create":
		if *name == "" {
			log.Fatal("Migration name is required for create command")
		}
		filename, err := migrationManager.CreateMigration(*name)
		if err != nil {
			log.Fatalf("Failed to create migration: %v", err)
		}
		fmt.Printf("Migration created successfully: %s\n", filename)

	case "status":
		status, err := migrationManager.GetStatus()
		if err != nil {
			log.Fatalf("Failed to get migration status: %v", err)
		}
		printStatus(status)

	case "force":
		if *version == 0 {
			log.Fatal("Version is required for force command")
		}
		if err := migrationManager.ForceVersion(*version); err != nil {
			log.Fatalf("Failed to force version: %v", err)
		}
		fmt.Printf("Forced migration version to %d\n", *version)

	case "drop":
		fmt.Println("WARNING: This will drop all database tables!")
		fmt.Print("Are you sure? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			fmt.Println("Operation cancelled")
			return
		}
		if err := migrationManager.Drop(); err != nil {
			log.Fatalf("Failed to drop database: %v", err)
		}
		fmt.Println("Database dropped successfully")

	case "validate":
		if err := migrationManager.ValidateMigrations(); err != nil {
			log.Fatalf("Migration validation failed: %v", err)
		}
		fmt.Println("All migrations are valid")

	case "list":
		files, err := migrationManager.ListMigrations()
		if err != nil {
			log.Fatalf("Failed to list migrations: %v", err)
		}
		printMigrations(files)

	default:
		fmt.Printf("Unknown command: %s\n", *command)
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println("UAS Migration Tool")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  go run cmd/migrate/main.go -cmd <command> [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  up       - Run all pending migrations")
	fmt.Println("  down     - Rollback the last migration")
	fmt.Println("  create   - Create a new migration pair (requires -name)")
	fmt.Println("  status   - Show current migration status")
	fmt.Println("  force    - Force migration to specific version (requires -version)")
	fmt.Println("  drop     - Drop all database tables (DANGEROUS)")
	fmt.Println("  validate - Validate migration files")
	fmt.Println("  list     - List all migration files")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -name string    - Migration name (for create command)")
	fmt.Println("  -version uint   - Target version (for force command)")
	fmt.Println("  -help           - Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  go run cmd/migrate/main.go -cmd up")
	fmt.Println("  go run cmd/migrate/main.go -cmd create -name add_user_roles")
	fmt.Println("  go run cmd/migrate/main.go -cmd down")
	fmt.Println("  go run cmd/migrate/main.go -cmd status")
	fmt.Println("  go run cmd/migrate/main.go -cmd force -version 5")
}

func printStatus(status *migrations.MigrationInfo) {
	fmt.Println("Migration Status:")
	fmt.Println("==================")
	fmt.Printf("Database: %s\n", status.DatabaseName)
	fmt.Printf("Current Version: %s\n", status.Current)
	fmt.Printf("Target Version: %s\n", status.Target)
	fmt.Printf("Dirty: %t\n", status.Dirty)

	if status.Dirty {
		fmt.Println("")
		fmt.Println("WARNING: Database is in a dirty state!")
		fmt.Println("Some migrations may have failed partially.")
		fmt.Println("Use 'force' command to fix the version if needed.")
	}
}

func printMigrations(files []migrations.MigrationFile) {
	fmt.Println("Migration Files:")
	fmt.Println("================")

	for _, file := range files {
		status := "Pending"
		if file.Direction == "down" {
			status = "Rollback"
		}

		fmt.Printf("Version: %04d | File: %s | Status: %s | Description: %s\n",
			file.Version, file.Name, status, file.Description)
	}
}
