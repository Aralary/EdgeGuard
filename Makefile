COMPOSE := docker compose -f deployments/docker-compose.yml

.PHONY: run-gateway run-demo run-control-plane compose-build compose-rebuild compose-up compose-down compose-reset compose-logs compose-ps smoke-test migrate-up migrate-down migrate-status test fmt tidy

run-gateway:
	go run ./cmd/gateway

run-demo:
	go run ./cmd/demo-backend

run-control-plane:
	go run ./cmd/control-plane

compose-build:
	$(COMPOSE) build

compose-rebuild:
	$(COMPOSE) build --no-cache

compose-up:
	$(COMPOSE) up -d --build

compose-down:
	$(COMPOSE) down --remove-orphans

compose-reset:
	$(COMPOSE) down -v --remove-orphans

compose-logs:
	$(COMPOSE) logs -f

compose-ps:
	$(COMPOSE) ps -a

smoke-test:
	curl -i http://localhost:8080/health
	curl -i http://localhost:8080/api/v1/orders
	curl -i http://localhost:8080/api/v1/orders/ord_1
	curl -i http://localhost:8082/health

migrate-up:
	$(COMPOSE) run --rm migrate up

migrate-down:
	$(COMPOSE) run --rm migrate down

migrate-status:
	$(COMPOSE) run --rm migrate status

test:
	go test ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy
