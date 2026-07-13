#!/usr/bin/env bash

set -euo pipefail

CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-http://localhost:8082}"
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
ANALYTICS_URL="${ANALYTICS_URL:-http://localhost:8084}"
UPSTREAM_URL="${UPSTREAM_URL:-http://demo-backend:8081}"
COMPOSE_FILE="${COMPOSE_FILE:-deployments/docker-compose.yml}"
POSTGRES_USER="${POSTGRES_USER:-edgeguard}"
POSTGRES_DB="${POSTGRES_DB:-edgeguard}"
E2E_TIMEOUT_SECONDS="${E2E_TIMEOUT_SECONDS:-120}"
REQUESTS_TO_SEND="${REQUESTS_TO_SEND:-3}"

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

analytics_summary() {
	curl -fsS --get \
		"${ANALYTICS_URL}/api/v1/projects/${PROJECT_ID}/analytics/summary" \
		--data-urlencode "route_name=${ROUTE_NAME}" \
		--data-urlencode "method=GET"
}

raw_event_count() {
	docker compose -f "$COMPOSE_FILE" exec -T postgres \
		psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "
SELECT COUNT(*)
FROM gateway_access_events
WHERE project_id = '${PROJECT_ID}'::uuid
  AND route_name = '${ROUTE_NAME}';
" | tr -d '[:space:]'
}

dump_diagnostics() {
	echo "--- analytics summary ---" >&2
	analytics_summary 2>/dev/null | jq . >&2 || true

	echo "--- raw events ---" >&2
	docker compose -f "$COMPOSE_FILE" exec -T postgres \
		psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "
SELECT request_id, occurred_at, status_code, duration_ms
FROM gateway_access_events
WHERE project_id = '${PROJECT_ID}'::uuid
  AND route_name = '${ROUTE_NAME}'
ORDER BY occurred_at;
" >&2 || true

	echo "--- hourly aggregates ---" >&2
	docker compose -f "$COMPOSE_FILE" exec -T postgres \
		psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "
SELECT bucket_start, method, request_count, status_2xx_count,
       status_4xx_count, status_5xx_count
FROM gateway_route_stats_hourly
WHERE project_id = '${PROJECT_ID}'::uuid
  AND route_name = '${ROUTE_NAME}'
ORDER BY bucket_start;
" >&2 || true

	echo "--- analytics worker logs ---" >&2
	docker compose -f "$COMPOSE_FILE" \
		logs --no-color --tail=100 analytics-worker >&2 || true

	echo "--- analytics consumer group ---" >&2
	docker compose -f "$COMPOSE_FILE" exec -T kafka \
		/opt/kafka/bin/kafka-consumer-groups.sh \
		--bootstrap-server localhost:9092 \
		--describe \
		--group edgeguard-analytics-v1 >&2 || true
}

require_command curl
require_command docker
require_command jq
require_positive_integer E2E_TIMEOUT_SECONDS "$E2E_TIMEOUT_SECONDS"
require_positive_integer REQUESTS_TO_SEND "$REQUESTS_TO_SEND"

RUN_ID="$(date +%s)-$$-${RANDOM}"
PROJECT_NAME="e2e-analytics-project-${RUN_ID}"
SERVICE_NAME="e2e-analytics-backend-${RUN_ID}"
ROUTE_NAME="e2e-analytics-route-${RUN_ID}"
PATH_PREFIX="/e2e-analytics-${RUN_ID}"
GATEWAY_ORDERS_URL="${GATEWAY_URL}${PATH_PREFIX}/orders"

response_file="$(mktemp)"
trap 'rm -f "$response_file"' EXIT

fail_with_diagnostics() {
	local message="$1"
	echo "$message" >&2
	dump_diagnostics
	exit 1
}

echo "Checking service health..."
curl -fsS "${CONTROL_PLANE_URL}/health" >/dev/null
curl -fsS "${GATEWAY_URL}/health" >/dev/null
curl -fsS "${ANALYTICS_URL}/health" >/dev/null

echo "Creating analytics test project and route..."
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

echo "Waiting for Gateway to apply the route..."
deadline=$((SECONDS + E2E_TIMEOUT_SECONDS))
ready_request_id=""
attempt=0
status=""
while ((SECONDS < deadline)); do
	attempt=$((attempt + 1))
	request_id="mvp7-${RUN_ID}-ready-${attempt}"
	status="$(curl -sS -o "$response_file" -w '%{http_code}' \
		-H "X-Request-ID: ${request_id}" \
		"$GATEWAY_ORDERS_URL" || true)"
	if [[ "$status" == "200" ]] && jq -e 'type == "array" and length > 0' "$response_file" >/dev/null 2>&1; then
		ready_request_id="$request_id"
		break
	fi
	sleep 1
done
if [[ "$status" != "200" || -z "$ready_request_id" ]]; then
	fail_with_diagnostics "Gateway did not apply analytics route within ${E2E_TIMEOUT_SECONDS}s"
fi

echo "Sending ${REQUESTS_TO_SEND} requests through Gateway..."
for ((i = 1; i <= REQUESTS_TO_SEND; i++)); do
	curl -fsS \
		-H "X-Request-ID: mvp7-${RUN_ID}-request-${i}" \
		"$GATEWAY_ORDERS_URL" >/dev/null
done

expected_count=$((REQUESTS_TO_SEND + 1))
echo "Waiting for ${expected_count} raw analytics events..."
deadline=$((SECONDS + E2E_TIMEOUT_SECONDS))
raw_count=0
while ((SECONDS < deadline)); do
	raw_count="$(raw_event_count 2>/dev/null || echo 0)"
	if [[ "$raw_count" =~ ^[0-9]+$ ]] && ((raw_count >= expected_count)); then
		break
	fi
	sleep 1
done
if ! [[ "$raw_count" =~ ^[0-9]+$ ]] || ((raw_count < expected_count)); then
	fail_with_diagnostics "raw event count=${raw_count:-invalid}, want at least ${expected_count}"
fi

echo "Waiting for analytics summary to reach ${expected_count} requests..."
deadline=$((SECONDS + E2E_TIMEOUT_SECONDS))
current=0
summary=""
while ((SECONDS < deadline)); do
	summary="$(analytics_summary 2>/dev/null || true)"
	if [[ -n "$summary" ]]; then
		current="$(jq -r '.request_count // 0' <<<"$summary" 2>/dev/null || echo 0)"
	else
		current=0
	fi
	if [[ "$current" =~ ^[0-9]+$ ]] && ((current >= expected_count)); then
		break
	fi
	sleep 1
done
if ! [[ "$current" =~ ^[0-9]+$ ]] || ((current < expected_count)); then
	fail_with_diagnostics "analytics request_count=${current:-invalid}, want at least ${expected_count}"
fi

if ! jq -e --arg route "$ROUTE_NAME" --argjson target "$expected_count" '
	.route_name == $route
	and .method == "GET"
	and .request_count >= $target
	and .status_2xx_count >= $target
	and .status_4xx_count == 0
' <<<"$summary" >/dev/null; then
	fail_with_diagnostics "unexpected analytics summary"
fi

hourly="$(curl -fsS --get \
	"${ANALYTICS_URL}/api/v1/projects/${PROJECT_ID}/analytics/hourly" \
	--data-urlencode "route_name=${ROUTE_NAME}" \
	--data-urlencode "method=GET" \
	--data-urlencode "limit=10")"

if ! jq -e --arg route "$ROUTE_NAME" --argjson target "$expected_count" '
	.items | length >= 1
	and ([.[] | select(.route_name == $route and .method == "GET") | .request_count] | add) >= $target
' <<<"$hourly" >/dev/null; then
	fail_with_diagnostics "unexpected hourly analytics response"
fi

echo "MVP7 analytics E2E test passed"
echo "project_id=${PROJECT_ID}"
echo "route_id=${ROUTE_ID}"
echo "summary_request_count=${current}"
