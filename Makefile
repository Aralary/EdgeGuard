POSTGRES_DSN ?= postgres://edgeguard:edgeguard@localhost:5432/edgeguard?sslmode=disable

.PHONY: run-gateway run-demo run-control-plane compose-build compose-rebuild compose-up compose-down compose-logs compose-ps smoke-test migrate-up migrate-down migrate-status test fmt tidy

run-gateway:
	go run ./cmd/gateway

run-demo:
	go run ./cmd/demo-backend

run-control-plane:
	go run ./cmd/control-plane

compose-build:
	docker compose -f deployments/docker-compose.yml build

compose-rebuild:
	docker compose -f deployments/docker-compose.yml build --no-cache

compose-up:
	docker compose -f deployments/docker-compose.yml up -d --build

compose-down:
	docker compose -f deployments/docker-compose.yml down --remove-orphans

compose-logs:
	docker compose -f deployments/docker-compose.yml logs -f

compose-ps:
	docker compose -f deployments/docker-compose.yml ps


smoke-test:
	curl -i http://localhost:8080/health
	curl -i http://localhost:8080/api/v1/orders
	curl -i http://localhost:8080/api/v1/orders/ord_1
	curl -i http://localhost:8082/health

migrate-up:
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations postgres "$(POSTGRES_DSN)" up

migrate-down:
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations postgres "$(POSTGRES_DSN)" down

migrate-status:
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations postgres "$(POSTGRES_DSN)" status

test:
	go test ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy