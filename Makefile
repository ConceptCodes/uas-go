run:
	@docker-compose -f docker-compose.yml up --build

clean:
	@go clean

# Migration commands
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
	@go test ./...

lint:
	@golangci-lint run

fmt:
	@go fmt ./...

mod-tidy:
	@go mod tidy

mod-download:
	@go mod download

.PHONY: run clean migrate-up migrate-down migrate-create migrate-status migrate-force migrate-drop migrate-validate migrate-list dev build test lint fmt mod-tidy mod-download