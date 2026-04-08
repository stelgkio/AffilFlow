.PHONY: run build test migrate-up swagger fmt vet check

run:
	go run ./cmd

build:
	go build -o bin/affilflow ./cmd

# Format before commits (uses gofmt; install goimports for import grouping: go install golang.org/x/tools/cmd/goimports@latest)
fmt:
	go fmt ./...

vet:
	go vet ./...

# fmt + vet + test — run before pushing
check: fmt vet
	go test ./...

swagger:
	go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/main.go -o apidocs --parseInternal --parseDependency

test:
	go test ./...

# Requires golang-migrate CLI: brew install golang-migrate
migrate-up:
	migrate -path migrations -database "$$DATABASE_URL" up
