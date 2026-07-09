.PHONY: run-gateway run-demo compose-build compose-rebuild compose-up compose-down compose-logs compose-ps smoke-test test fmt tidy

run-gateway:
	go run ./cmd/gateway

run-demo:
	go run ./cmd/demo-backend

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

test:
	go test ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy