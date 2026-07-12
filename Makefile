COMPOSE := docker compose -f deployments/docker-compose.yml

.PHONY: run-gateway run-demo run-control-plane run-auth compose-build compose-rebuild compose-up compose-down compose-reset compose-logs compose-ps smoke-test e2e-test e2e-auth-test e2e-rate-limit-test migrate-up migrate-down migrate-status test fmt tidy

run-gateway:
	go run ./cmd/gateway

run-demo:
	go run ./cmd/demo-backend

run-control-plane:
	go run ./cmd/control-plane

run-auth:
	go run ./cmd/auth

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
	curl -fsS http://localhost:8080/health
	@echo
	curl -fsS http://localhost:8082/health
	@echo
	curl -fsS http://localhost:8083/health
	@echo
	curl -fsS http://localhost:8082/internal/v1/routes
	@echo
	@$(COMPOSE) exec -T redis redis-cli ping | grep -q PONG
	@echo "Redis: PONG"

e2e-test:
	./scripts/e2e/mvp4_dynamic_routes.sh
	./scripts/e2e/mvp5_api_key_auth.sh
	./scripts/e2e/mvp6_rate_limiting.sh

e2e-auth-test:
	./scripts/e2e/mvp5_api_key_auth.sh

e2e-rate-limit-test:
	./scripts/e2e/mvp6_rate_limiting.sh

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
