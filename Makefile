COMPOSE := docker compose -f deployments/docker-compose.yml

KIND_CLUSTER_NAME ?= edgeguard
KUBE_CONTEXT ?= kind-$(KIND_CLUSTER_NAME)
K8S_NAMESPACE ?= edgeguard
K8S_OVERLAY ?= deployments/k8s/overlays/local
K8S_BASE ?= deployments/k8s/base
KIND_CONFIG ?= deployments/k8s/kind/cluster.yaml
LOCAL_IMAGE_TAG ?= local
CLOUD_PROVIDER_KIND_IMAGE ?= registry.k8s.io/cloud-provider-kind/cloud-controller-manager:v0.11.1
CLOUD_PROVIDER_KIND_CONTAINER ?= edgeguard-cloud-provider-kind
K8S_TIMEOUT_SECONDS ?= 300

K8S_LOCAL_IMAGES := \
	edgeguard-gateway:$(LOCAL_IMAGE_TAG) \
	edgeguard-control-plane:$(LOCAL_IMAGE_TAG) \
	edgeguard-auth:$(LOCAL_IMAGE_TAG) \
	edgeguard-analytics-worker:$(LOCAL_IMAGE_TAG) \
	edgeguard-notification-worker:$(LOCAL_IMAGE_TAG) \
	edgeguard-demo-backend:$(LOCAL_IMAGE_TAG) \
	edgeguard-migrate:$(LOCAL_IMAGE_TAG)

.PHONY: run-gateway run-demo run-control-plane run-auth run-analytics run-notification compose-build compose-rebuild compose-up compose-down compose-reset compose-logs compose-ps wait-prometheus wait-tracing wait-grafana smoke-test e2e-test e2e-auth-test e2e-rate-limit-test e2e-analytics-test e2e-jobs-test e2e-observability-test migrate-up migrate-down migrate-status k8s-check-tools k8s-render k8s-cluster-create k8s-cloud-provider-up k8s-images k8s-load-images k8s-validate k8s-deploy k8s-wait k8s-up k8s-down k8s-status k8s-smoke-test k8s-e2e-test k8s-logs test fmt tidy

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

wait-grafana:
	@echo "Waiting for Grafana readiness..."
	@for attempt in $$(seq 1 30); do \
		if curl -fsS http://localhost:3000/api/health >/dev/null 2>&1; then \
			echo "Grafana: ready"; \
			exit 0; \
		fi; \
		sleep 2; \
	done; \
	echo "Grafana did not become ready" >&2; \
	$(COMPOSE) ps -a grafana >&2; \
	$(COMPOSE) logs --no-color --tail=100 grafana >&2; \
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
	@$(MAKE) --no-print-directory wait-grafana
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
	./scripts/e2e/mvp9_observability.sh

e2e-auth-test:
	./scripts/e2e/mvp5_api_key_auth.sh

e2e-rate-limit-test:
	./scripts/e2e/mvp6_rate_limiting.sh

e2e-analytics-test:
	./scripts/e2e/mvp7_analytics.sh

e2e-jobs-test:
	./scripts/e2e/mvp8_background_jobs.sh

e2e-observability-test:
	./scripts/e2e/mvp9_observability.sh


k8s-check-tools:
	@for tool in docker kind kubectl curl jq; do \
		command -v $$tool >/dev/null 2>&1 || { echo "required command is not installed: $$tool" >&2; exit 1; }; \
	done

k8s-render:
	kubectl kustomize $(K8S_OVERLAY)

k8s-cluster-create: k8s-check-tools
	@if kind get clusters 2>/dev/null | grep -Fxq '$(KIND_CLUSTER_NAME)'; then \
		echo "Kind cluster $(KIND_CLUSTER_NAME) already exists"; \
	else \
		kind create cluster --name $(KIND_CLUSTER_NAME) --config $(KIND_CONFIG); \
	fi
	kubectl --context $(KUBE_CONTEXT) wait --for=condition=Ready node --all --timeout=120s
	@$(MAKE) --no-print-directory k8s-cloud-provider-up

k8s-cloud-provider-up: k8s-check-tools
	@docker_endpoint="$${DOCKER_HOST:-}"; \
	if [ -z "$$docker_endpoint" ]; then \
		docker_endpoint="$$(docker context inspect --format '{{(index .Endpoints "docker").Host}}' 2>/dev/null || true)"; \
	fi; \
	case "$$docker_endpoint" in \
		unix://*) docker_socket="$${docker_endpoint#unix://}" ;; \
		*) \
			echo "cloud-provider-kind requires a local Unix Docker socket, got: $${docker_endpoint:-<empty>}" >&2; \
			exit 1; \
			;; \
	esac; \
	if [ ! -S "$$docker_socket" ]; then \
		echo "Docker socket does not exist or is not a socket: $$docker_socket" >&2; \
		echo "Current Docker endpoint: $$docker_endpoint" >&2; \
		exit 1; \
	fi; \
	echo "Cloud Provider KIND Docker socket: $$docker_socket"; \
	docker rm -f $(CLOUD_PROVIDER_KIND_CONTAINER) >/dev/null 2>&1 || true; \
	security_opt=""; \
	if command -v getenforce >/dev/null 2>&1 && [ "$$(getenforce 2>/dev/null || true)" != "Disabled" ]; then \
		security_opt="--security-opt label=disable"; \
	fi; \
	docker run -d \
		--name $(CLOUD_PROVIDER_KIND_CONTAINER) \
		--restart unless-stopped \
		--network kind \
		$$security_opt \
		-e KIND_EXPERIMENTAL_PROVIDER=docker \
		-e DOCKER_HOST=unix:///var/run/docker.sock \
		-v "$$docker_socket:/var/run/docker.sock" \
		$(CLOUD_PROVIDER_KIND_IMAGE) >/dev/null; \
	for attempt in $$(seq 1 30); do \
		status="$$(docker inspect -f '{{.State.Status}}' $(CLOUD_PROVIDER_KIND_CONTAINER) 2>/dev/null || true)"; \
		if [ "$$status" = "running" ] \
			&& docker exec $(CLOUD_PROVIDER_KIND_CONTAINER) docker info >/dev/null 2>&1 \
			&& kubectl --context $(KUBE_CONTEXT) get ingressclass cloud-provider-kind >/dev/null 2>&1; then \
			echo "Cloud Provider KIND: ready"; \
			exit 0; \
		fi; \
		if [ "$$status" = "exited" ] || [ "$$status" = "dead" ]; then \
			break; \
		fi; \
		sleep 2; \
	done; \
	echo "Cloud Provider KIND did not become ready" >&2; \
	docker inspect \
		--format 'status={{.State.Status}} restarting={{.State.Restarting}} restarts={{.RestartCount}} error={{.State.Error}}' \
		$(CLOUD_PROVIDER_KIND_CONTAINER) >&2 || true; \
	docker logs --tail=200 $(CLOUD_PROVIDER_KIND_CONTAINER) >&2 || true; \
	exit 1

k8s-images: k8s-check-tools
	docker build -f deployments/docker/gateway.Dockerfile -t edgeguard-gateway:$(LOCAL_IMAGE_TAG) .
	docker build -f deployments/docker/control-plane.Dockerfile -t edgeguard-control-plane:$(LOCAL_IMAGE_TAG) .
	docker build -f deployments/docker/auth.Dockerfile -t edgeguard-auth:$(LOCAL_IMAGE_TAG) .
	docker build -f deployments/docker/analytics-worker.Dockerfile -t edgeguard-analytics-worker:$(LOCAL_IMAGE_TAG) .
	docker build -f deployments/docker/notification-worker.Dockerfile -t edgeguard-notification-worker:$(LOCAL_IMAGE_TAG) .
	docker build -f deployments/docker/demo-backend.Dockerfile -t edgeguard-demo-backend:$(LOCAL_IMAGE_TAG) .
	docker build -f deployments/docker/migrations.Dockerfile -t edgeguard-migrate:$(LOCAL_IMAGE_TAG) .

k8s-load-images: k8s-check-tools
	@kind get clusters 2>/dev/null | grep -Fxq '$(KIND_CLUSTER_NAME)' || { \
		echo "Kind cluster $(KIND_CLUSTER_NAME) does not exist" >&2; \
		exit 1; \
	}
	kind load docker-image --name $(KIND_CLUSTER_NAME) $(K8S_LOCAL_IMAGES)

k8s-validate: k8s-check-tools
	# Server-side dry-run needs the target namespace to exist.
	kubectl --context $(KUBE_CONTEXT) apply -f $(K8S_BASE)/namespace.yaml
	# Completed Jobs are recreated on every deployment and may contain immutable fields.
	@kubectl --context $(KUBE_CONTEXT) -n $(K8S_NAMESPACE) delete \
		job/edgeguard-migrations job/kafka-init \
		--ignore-not-found >/dev/null
	kubectl --context $(KUBE_CONTEXT) apply --dry-run=server -k $(K8S_OVERLAY)

k8s-deploy: k8s-check-tools
	# Jobs are one-shot workloads and are recreated on every deployment.
	@kubectl --context $(KUBE_CONTEXT) -n $(K8S_NAMESPACE) delete \
		job/edgeguard-migrations job/kafka-init \
		--ignore-not-found >/dev/null
	kubectl --context $(KUBE_CONTEXT) apply -k $(K8S_OVERLAY)
	# Local images reuse the :local tag; force Pods to consume the newly loaded image.
	kubectl --context $(KUBE_CONTEXT) -n $(K8S_NAMESPACE) rollout restart \
		deployment/gateway \
		deployment/control-plane \
		deployment/auth \
		deployment/analytics-worker \
		deployment/notification-worker \
		deployment/demo-backend

k8s-wait:
	KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) \
	KUBE_CONTEXT=$(KUBE_CONTEXT) \
	K8S_NAMESPACE=$(K8S_NAMESPACE) \
	K8S_OVERLAY=$(K8S_OVERLAY) \
	K8S_TIMEOUT_SECONDS=$(K8S_TIMEOUT_SECONDS) \
	./scripts/k8s/wait-ready.sh

k8s-up:
	@$(MAKE) --no-print-directory k8s-cluster-create
	@$(MAKE) --no-print-directory k8s-images
	@$(MAKE) --no-print-directory k8s-load-images
	@$(MAKE) --no-print-directory k8s-validate
	@$(MAKE) --no-print-directory k8s-deploy
	@$(MAKE) --no-print-directory k8s-wait
	@$(MAKE) --no-print-directory k8s-smoke-test

k8s-down: k8s-check-tools
	@docker rm -f $(CLOUD_PROVIDER_KIND_CONTAINER) >/dev/null 2>&1 || true
	@kind delete cluster --name $(KIND_CLUSTER_NAME)

k8s-status:
	@echo "Context: $(KUBE_CONTEXT)"
	kubectl --context $(KUBE_CONTEXT) get nodes -o wide
	kubectl --context $(KUBE_CONTEXT) get all -n $(K8S_NAMESPACE)
	kubectl --context $(KUBE_CONTEXT) get ingress,pvc -n $(K8S_NAMESPACE)
	@docker ps --filter name=$(CLOUD_PROVIDER_KIND_CONTAINER) --format 'Cloud provider: {{.Names}} {{.Status}}'

k8s-smoke-test:
	KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) \
	KUBE_CONTEXT=$(KUBE_CONTEXT) \
	K8S_NAMESPACE=$(K8S_NAMESPACE) \
	K8S_OVERLAY=$(K8S_OVERLAY) \
	K8S_TIMEOUT_SECONDS=$(K8S_TIMEOUT_SECONDS) \
	./scripts/k8s/smoke-test.sh

k8s-e2e-test:
	KIND_CLUSTER_NAME=$(KIND_CLUSTER_NAME) \
	KUBE_CONTEXT=$(KUBE_CONTEXT) \
	K8S_NAMESPACE=$(K8S_NAMESPACE) \
	K8S_OVERLAY=$(K8S_OVERLAY) \
	K8S_TIMEOUT_SECONDS=$(K8S_TIMEOUT_SECONDS) \
	./scripts/k8s/e2e-test.sh

k8s-logs:
	kubectl --context $(KUBE_CONTEXT) logs -n $(K8S_NAMESPACE) \
		-l app.kubernetes.io/part-of=edgeguard \
		--all-containers=true --prefix=true --tail=200

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
