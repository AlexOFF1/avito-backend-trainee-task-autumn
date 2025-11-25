build:
	go build -o main ./cmd/main.go

test:
	go test ./...

lint:
	golangci-lint run

migrate:
	goose -dir migrations postgres "${DB_CONN}" up