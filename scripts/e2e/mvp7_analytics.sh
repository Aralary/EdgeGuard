#!/usr/bin/env bash

set -euo pipefail

CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-http://localhost:8082}"
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
ANALYTICS_URL="${ANALYTICS_URL:-http://localhost:8084}"
UPSTREAM_URL="${UPSTREAM_URL:-http://demo-backend:8081}"
E2E_TIMEOUT_SECONDS="${E2E_TIMEOUT_SECONDS:-40}"
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

require_command curl
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
while ((SECONDS < deadline)); do
	status="$(curl -sS -o "$response_file" -w '%{http_code}' "$GATEWAY_ORDERS_URL" || true)"
	if [[ "$status" == "200" ]] && jq -e 'type == "array" and length > 0' "$response_file" >/dev/null 2>&1; then
		break
	fi
	sleep 1
done
if [[ "${status:-}" != "200" ]]; then
	echo "Gateway did not apply analytics route within ${E2E_TIMEOUT_SECONDS}s" >&2
	exit 1
fi

echo "Waiting for the baseline event to be aggregated..."
baseline=0
deadline=$((SECONDS + E2E_TIMEOUT_SECONDS))
while ((SECONDS < deadline)); do
	summary="$(analytics_summary || true)"
	baseline="$(jq -r '.request_count // 0' <<<"${summary:-{}}" 2>/dev/null || echo 0)"
	if [[ "$baseline" =~ ^[0-9]+$ ]] && ((baseline >= 1)); then
		break
	fi
	sleep 1
done
if ! [[ "$baseline" =~ ^[0-9]+$ ]] || ((baseline < 1)); then
	echo "Analytics worker did not aggregate the baseline request" >&2
	exit 1
fi

echo "Sending ${REQUESTS_TO_SEND} requests through Gateway..."
for ((i = 1; i <= REQUESTS_TO_SEND; i++)); do
	curl -fsS "$GATEWAY_ORDERS_URL" >/dev/null
done

target=$((baseline + REQUESTS_TO_SEND))
echo "Waiting for analytics summary to reach ${target} requests..."
deadline=$((SECONDS + E2E_TIMEOUT_SECONDS))
current="$baseline"
while ((SECONDS < deadline)); do
	summary="$(analytics_summary || true)"
	current="$(jq -r '.request_count // 0' <<<"${summary:-{}}" 2>/dev/null || echo 0)"
	if [[ "$current" =~ ^[0-9]+$ ]] && ((current >= target)); then
		break
	fi
	sleep 1
done
if ! [[ "$current" =~ ^[0-9]+$ ]] || ((current < target)); then
	echo "analytics request_count=${current:-invalid}, want at least ${target}" >&2
	echo "last summary: ${summary:-missing}" >&2
	exit 1
fi

if ! jq -e --arg route "$ROUTE_NAME" --argjson target "$target" '
	.route_name == $route
	and .method == "GET"
	and .request_count >= $target
	and .status_2xx_count >= $target
	and .status_4xx_count == 0
' <<<"$summary" >/dev/null; then
	echo "unexpected analytics summary" >&2
	echo "$summary" | jq . >&2
	exit 1
fi

hourly="$(curl -fsS --get \
	"${ANALYTICS_URL}/api/v1/projects/${PROJECT_ID}/analytics/hourly" \
	--data-urlencode "route_name=${ROUTE_NAME}" \
	--data-urlencode "method=GET" \
	--data-urlencode "limit=10")"

if ! jq -e --arg route "$ROUTE_NAME" --argjson target "$target" '
	.items | length >= 1
	and ([.[] | select(.route_name == $route and .method == "GET") | .request_count] | add) >= $target
' <<<"$hourly" >/dev/null; then
	echo "unexpected hourly analytics response" >&2
	echo "$hourly" | jq . >&2
	exit 1
fi

echo "MVP7 analytics E2E test passed"
echo "project_id=${PROJECT_ID}"
echo "route_id=${ROUTE_ID}"
echo "summary_request_count=${current}"
