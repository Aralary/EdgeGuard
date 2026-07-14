COMPOSE := docker compose -f deployments/docker-compose.yml

.PHONY: run-gateway run-demo run-control-plane run-auth run-analytics run-notification compose-build compose-rebuild compose-up compose-down compose-reset compose-logs compose-ps wait-prometheus wait-tracing smoke-test e2e-test e2e-auth-test e2e-rate-limit-test e2e-analytics-test e2e-jobs-test migrate-up migrate-down migrate-status test fmt tidy

run-gateway:
	go run ./cmd/gateway

run-demo:
	go run ./cmd/demo-backend

run-control-plane:
	go run ./cmd/control-plane

run-auth:
	go run ./cmd/auth

run-analytics:
	go run ./cmd/analytics-worker

run-notification:
	go run ./cmd/notification-worker

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

wait-prometheus:
	@echo "Waiting for Prometheus readiness..."
	@for attempt in $$(seq 1 30); do \
		if curl -fsS http://localhost:9090/-/ready >/dev/null 2>&1; then \
			echo "Prometheus: ready"; \
			exit 0; \
		fi; \
		sleep 2; \
	done; \
	echo "Prometheus did not become ready" >&2; \
	$(COMPOSE) ps -a prometheus >&2; \
	$(COMPOSE) logs --no-color --tail=100 prometheus >&2; \
	exit 1

wait-tracing:
	@echo "Waiting for OpenTelemetry Collector and Tempo..."
	@for attempt in $$(seq 1 30); do \
		if curl -fsS http://localhost:13133/ >/dev/null 2>&1 && curl -fsS http://localhost:3200/ready >/dev/null 2>&1; then \
			echo "Tracing backends: ready"; \
			exit 0; \
		fi; \
		sleep 2; \
	done; \
	echo "Tracing backends did not become ready" >&2; \
	$(COMPOSE) ps -a otel-collector tempo >&2; \
	$(COMPOSE) logs --no-color --tail=100 otel-collector tempo >&2; \
	exit 1

smoke-test:
	curl -fsS http://localhost:8080/health
	@echo
	curl -fsS http://localhost:8082/health
	@echo
	curl -fsS http://localhost:8083/health
	@echo
	curl -fsS http://localhost:8084/health
	@echo
	curl -fsS http://localhost:8085/health
	@echo
	curl -fsS http://localhost:8080/ready
	@echo
	curl -fsS http://localhost:8082/ready
	@echo
	curl -fsS http://localhost:8083/ready
	@echo
	curl -fsS http://localhost:8084/ready
	@echo
	curl -fsS http://localhost:8085/ready
	@echo
	@$(MAKE) --no-print-directory wait-prometheus
	@echo
	@$(MAKE) --no-print-directory wait-tracing
	@echo
	curl -fsS http://localhost:8082/internal/v1/routes
	@echo
	@$(COMPOSE) exec -T redis redis-cli ping | grep -q PONG
	@echo "Redis: PONG"
	@$(COMPOSE) exec -T kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list | grep -q '^edgeguard.gateway.access.v1$$'
	@echo "Kafka topic: edgeguard.gateway.access.v1"
	@$(COMPOSE) exec -T rabbitmq rabbitmq-diagnostics -q ping
	@echo "RabbitMQ: Ping succeeded"
	@$(COMPOSE) exec -T rabbitmq rabbitmqctl -q list_queues -p edgeguard name | grep -q '^edgeguard.jobs.main.v1$$'
	@echo "RabbitMQ queue: edgeguard.jobs.main.v1"
	@$(COMPOSE) exec -T rabbitmq rabbitmqctl -q list_queues -p edgeguard name | grep -q '^edgeguard.jobs.dead.v1$$'
	@echo "RabbitMQ DLQ: edgeguard.jobs.dead.v1"

e2e-test:
	./scripts/e2e/mvp4_dynamic_routes.sh
	./scripts/e2e/mvp5_api_key_auth.sh
	./scripts/e2e/mvp6_rate_limiting.sh
	./scripts/e2e/mvp7_analytics.sh
	./scripts/e2e/mvp8_background_jobs.sh

e2e-auth-test:
	./scripts/e2e/mvp5_api_key_auth.sh

e2e-rate-limit-test:
	./scripts/e2e/mvp6_rate_limiting.sh

e2e-analytics-test:
	./scripts/e2e/mvp7_analytics.sh

e2e-jobs-test:
	./scripts/e2e/mvp8_background_jobs.sh

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
