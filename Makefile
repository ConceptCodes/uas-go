run:
	@docker-compose -f docker-compose.yml up --build

clean:
	@go clean

# Migration commands
migrate: migrate-up

migrate-up:
	@go run cmd/migrate/main.go -cmd up

migrate-down:
	@go run cmd/migrate/main.go -cmd down

migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=migration_name"; exit 1; fi
	@go run cmd/migrate/main.go -cmd create -name $(NAME)

migrate-status:
	@go run cmd/migrate/main.go -cmd status

migrate-force:
	@if [ -z "$(VERSION)" ]; then echo "Usage: make migrate-force VERSION=version_number"; exit 1; fi
	@go run cmd/migrate/main.go -cmd force -version $(VERSION)

migrate-drop:
	@go run cmd/migrate/main.go -cmd drop

migrate-validate:
	@go run cmd/migrate/main.go -cmd validate

migrate-list:
	@go run cmd/migrate/main.go -cmd list

# Development commands
dev:
	@go run cmd/api/main.go

build:
	@go build -o bin/uas cmd/api/main.go

test:
	@go test ./... -count=1

test-race:
	@go test ./... -race -count=1

test-cover:
	@go test ./... -coverprofile=coverage.out -count=1
	@go tool cover -func=coverage.out | tail -1

test-cover-html:
	@go test ./... -coverprofile=coverage.out -count=1
	@go tool cover -html=coverage.out -o coverage.html

test-verbose:
	@go test ./... -v -count=1

test-short:
	@go test ./... -short -count=1

test-smoke-auth:
	@bash scripts/auth_smoke.sh

vet:
	@go vet ./...

lint:
	@golangci-lint run

fmt:
	@go fmt ./...

backup-db:
	@bash scripts/db_backup.sh

restore-db:
	@if [ -z "$(FILE)" ]; then echo "Usage: make restore-db FILE=backups/your_file.sql.gz"; exit 1; fi
	@bash scripts/db_restore.sh $(FILE)

lint:
	@golangci-lint run

fmt:
	@go fmt ./...

mod-tidy:
	@go mod tidy

mod-download:
	@go mod download

.PHONY: run clean migrate migrate-up migrate-down migrate-create migrate-status migrate-force migrate-drop migrate-validate migrate-list dev build test test-smoke-auth backup-db restore-db lint fmt mod-tidy mod-download
