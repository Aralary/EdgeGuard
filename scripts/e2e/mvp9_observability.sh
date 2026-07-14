#!/usr/bin/env bash

set -euo pipefail

CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-http://localhost:8082}"
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
TEMPO_URL="${TEMPO_URL:-http://localhost:3200}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
GRAFANA_USER="${GRAFANA_USER:-admin}"
GRAFANA_PASSWORD="${GRAFANA_PASSWORD:-admin}"
UPSTREAM_URL="${UPSTREAM_URL:-http://demo-backend:8081}"
COMPOSE_FILE="${COMPOSE_FILE:-deployments/docker-compose.yml}"
E2E_TIMEOUT_SECONDS="${E2E_TIMEOUT_SECONDS:-120}"

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "required command is not installed: $1" >&2
		exit 1
	fi
}

require_positive_integer() {
	local name="$1"
	local value="$2"
	if ! [[ "$value" =~ ^[1-9][0-9]*$ ]]; then
		echo "${name} must be a positive integer" >&2
		exit 1
	fi
}

json_post() {
	local url="$1"
	local body="$2"
	curl -fsS -X POST "$url" -H "Content-Type: application/json" -d "$body"
}

trace_id_from_headers() {
	awk '
		BEGIN { IGNORECASE=1 }
		/^X-Trace-ID:/ {
			gsub("\r", "", $2)
			print $2
			exit
		}
	' "$1"
}

prometheus_value() {
	local query="$1"
	curl -fsS --get "${PROMETHEUS_URL}/api/v1/query" \
		--data-urlencode "query=${query}" |
		jq -r '.data.result[0].value[1] // "0"'
}

number_greater_than() {
	local current="$1"
	local baseline="$2"
	awk -v current="$current" -v baseline="$baseline" 'BEGIN { exit !(current > baseline) }'
}

trace_contains_services() {
	local file="$1"
	shift
	local service
	for service in "$@"; do
		if ! jq -e --arg service "$service" \
			'[.. | strings | select(. == $service)] | length > 0' \
			"$file" >/dev/null; then
			return 1
		fi
	done
}

wait_for_trace() {
	local trace_id="$1"
	shift
	local trace_file="$1"
	shift
	local deadline=$((SECONDS + E2E_TIMEOUT_SECONDS))
	local status=""

	while ((SECONDS < deadline)); do
		status="$(curl -sS -o "$trace_file" -w '%{http_code}' \
			"${TEMPO_URL}/api/traces/${trace_id}" || true)"
		if [[ "$status" == "200" ]] && trace_contains_services "$trace_file" "$@"; then
			return 0
		fi
		sleep 1
	done

	return 1
}

wait_for_job_status() {
	local job_id="$1"
	local expected="$2"
	local deadline=$((SECONDS + E2E_TIMEOUT_SECONDS))
	local response=""
	local status=""

	while ((SECONDS < deadline)); do
		response="$(curl -fsS "${CONTROL_PLANE_URL}/api/v1/jobs/${job_id}" 2>/dev/null || true)"
		status="$(jq -r '.status // empty' <<<"$response" 2>/dev/null || true)"
		if [[ "$status" == "$expected" ]]; then
			return 0
		fi
		if [[ "$status" == "failed" || "$status" == "publish_failed" ]]; then
			echo "job ${job_id} reached terminal status ${status}: ${response}" >&2
			return 1
		fi
		sleep 1
	done

	echo "job ${job_id} did not reach status ${expected}; last response: ${response}" >&2
	return 1
}

dump_diagnostics() {
	echo "--- Prometheus targets ---" >&2
	curl -fsS "${PROMETHEUS_URL}/api/v1/targets" | jq '.data.activeTargets[] | {job: .labels.job, health: .health, lastError: .lastError}' >&2 || true

	echo "--- Grafana health ---" >&2
	curl -fsS "${GRAFANA_URL}/api/health" | jq . >&2 || true

	echo "--- observability container logs ---" >&2
	docker compose -f "$COMPOSE_FILE" logs --no-color --tail=100 \
		prometheus tempo otel-collector grafana >&2 || true
}

require_command curl
require_command docker
require_command jq
require_positive_integer E2E_TIMEOUT_SECONDS "$E2E_TIMEOUT_SECONDS"

RUN_ID="$(date +%s)-$$-${RANDOM}"
PROJECT_NAME="e2e-observability-project-${RUN_ID}"
SERVICE_NAME="e2e-observability-backend-${RUN_ID}"
ROUTE_NAME="e2e-observability-route-${RUN_ID}"
PATH_PREFIX="/e2e-observability-${RUN_ID}"
GATEWAY_ORDERS_URL="${GATEWAY_URL}${PATH_PREFIX}/orders"
ORDER_ID="observability-order-${RUN_ID}"

headers_file="$(mktemp)"
response_file="$(mktemp)"
trace_file="$(mktemp)"
trap 'rm -f "$headers_file" "$response_file" "$trace_file"' EXIT

fail_with_diagnostics() {
	local message="$1"
	echo "$message" >&2
	dump_diagnostics
	exit 1
}

echo "Checking observability backends..."
curl -fsS "${PROMETHEUS_URL}/-/ready" >/dev/null
curl -fsS "${TEMPO_URL}/ready" >/dev/null
curl -fsS "${GRAFANA_URL}/api/health" >/dev/null

unhealthy_targets="$(curl -fsS "${PROMETHEUS_URL}/api/v1/targets" |
	jq '[.data.activeTargets[] | select(.labels.job | startswith("edgeguard-")) | select(.health != "up")] | length')"
if [[ "$unhealthy_targets" != "0" ]]; then
	fail_with_diagnostics "Prometheus has ${unhealthy_targets} unhealthy EdgeGuard targets"
fi

echo "Checking provisioned Grafana resources..."
curl -fsS -u "${GRAFANA_USER}:${GRAFANA_PASSWORD}" \
	"${GRAFANA_URL}/api/datasources/uid/prometheus" |
	jq -e '.uid == "prometheus" and .type == "prometheus"' >/dev/null
curl -fsS -u "${GRAFANA_USER}:${GRAFANA_PASSWORD}" \
	"${GRAFANA_URL}/api/datasources/uid/tempo" |
	jq -e '.uid == "tempo" and .type == "tempo"' >/dev/null
curl -fsS -u "${GRAFANA_USER}:${GRAFANA_PASSWORD}" \
	"${GRAFANA_URL}/api/dashboards/uid/edgeguard-overview" |
	jq -e '.dashboard.uid == "edgeguard-overview" and (.dashboard.panels | length) >= 8' >/dev/null

echo "Creating a route for HTTP metrics and tracing..."
project_response="$(json_post \
	"${CONTROL_PLANE_URL}/api/v1/projects" \
	"$(jq -nc --arg name "$PROJECT_NAME" '{name: $name}')")"
PROJECT_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$project_response")"

service_response="$(json_post \
	"${CONTROL_PLANE_URL}/api/v1/projects/${PROJECT_ID}/services" \
	"$(jq -nc --arg name "$SERVICE_NAME" --arg upstream_url "$UPSTREAM_URL" \
		'{name: $name, upstream_url: $upstream_url}')")"
SERVICE_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$service_response")"

route_response="$(json_post \
	"${CONTROL_PLANE_URL}/api/v1/services/${SERVICE_ID}/routes" \
	"$(jq -nc --arg name "$ROUTE_NAME" --arg path_prefix "$PATH_PREFIX" \
		'{
			name: $name,
			path_prefix: $path_prefix,
			strip_prefix: true,
			timeout_ms: 3000,
			enabled: true,
			auth_required: false,
			rate_limit_enabled: false
		}')")"
ROUTE_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$route_response")"

baseline="$(prometheus_value 'sum(edgeguard_http_requests_total{service="gateway",status_code="200"})')"

echo "Waiting for Gateway to apply the route and capturing a trace..."
deadline=$((SECONDS + E2E_TIMEOUT_SECONDS))
http_status=""
TRACE_ID=""
while ((SECONDS < deadline)); do
	: >"$headers_file"
	http_status="$(curl -sS -D "$headers_file" -o "$response_file" -w '%{http_code}' \
		-H "X-Request-ID: mvp9-${RUN_ID}-http" \
		"$GATEWAY_ORDERS_URL" || true)"
	TRACE_ID="$(trace_id_from_headers "$headers_file")"
	if [[ "$http_status" == "200" ]] \
		&& [[ "$TRACE_ID" =~ ^[0-9a-f]{32}$ ]] \
		&& jq -e 'type == "array" and length > 0' "$response_file" >/dev/null 2>&1; then
		break
	fi
	sleep 1
done
if [[ "$http_status" != "200" || ! "$TRACE_ID" =~ ^[0-9a-f]{32}$ ]]; then
	fail_with_diagnostics "Gateway route did not produce a valid HTTP trace"
fi

echo "Waiting for Prometheus to scrape the request metric..."
deadline=$((SECONDS + E2E_TIMEOUT_SECONDS))
current="$baseline"
while ((SECONDS < deadline)); do
	current="$(prometheus_value 'sum(edgeguard_http_requests_total{service="gateway",status_code="200"})' 2>/dev/null || echo 0)"
	if number_greater_than "$current" "$baseline"; then
		break
	fi
	sleep 1
done
if ! number_greater_than "$current" "$baseline"; then
	fail_with_diagnostics "Gateway request metric did not increase: baseline=${baseline}, current=${current}"
fi

echo "Waiting for the HTTP/Kafka distributed trace..."
if ! wait_for_trace "$TRACE_ID" "$trace_file" gateway demo-backend analytics; then
	echo "last trace response:" >&2
	cat "$trace_file" >&2 || true
	fail_with_diagnostics "trace ${TRACE_ID} did not contain gateway, demo-backend and analytics services"
fi

echo "Submitting a RabbitMQ webhook job and checking async trace propagation..."
: >"$headers_file"
job_http_status="$(curl -sS -D "$headers_file" -o "$response_file" -w '%{http_code}' \
	-X POST "${CONTROL_PLANE_URL}/api/v1/jobs/webhooks" \
	-H "Content-Type: application/json" \
	-d "$(jq -nc \
		--arg url 'http://demo-backend:8081/orders' \
		--arg id "$ORDER_ID" \
		'{
			url: $url,
			method: "POST",
			headers: {"Content-Type": "application/json"},
			body: {id: $id, status: "created", amount: 1000},
			max_attempts: 3
		}')")"
JOB_TRACE_ID="$(trace_id_from_headers "$headers_file")"
if [[ "$job_http_status" != "202" || ! "$JOB_TRACE_ID" =~ ^[0-9a-f]{32}$ ]]; then
	fail_with_diagnostics "job submission did not return HTTP 202 and a valid trace ID"
fi
JOB_ID="$(jq -er '.id | strings | select(length > 0)' "$response_file")"

if ! wait_for_job_status "$JOB_ID" succeeded; then
	fail_with_diagnostics "webhook job did not succeed"
fi

if ! wait_for_trace "$JOB_TRACE_ID" "$trace_file" control-plane notification-worker demo-backend; then
	echo "last job trace response:" >&2
	cat "$trace_file" >&2 || true
	fail_with_diagnostics "trace ${JOB_TRACE_ID} did not contain control-plane, notification-worker and demo-backend services"
fi

echo "MVP9 observability E2E test passed"
echo "project_id=${PROJECT_ID}"
echo "route_id=${ROUTE_ID}"
echo "http_trace_id=${TRACE_ID}"
echo "job_id=${JOB_ID}"
echo "job_trace_id=${JOB_TRACE_ID}"
echo "grafana_dashboard=${GRAFANA_URL}/d/edgeguard-overview/edgeguard-overview"
