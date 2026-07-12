#!/usr/bin/env bash

set -euo pipefail

CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-http://localhost:8082}"
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
E2E_TIMEOUT_SECONDS="${E2E_TIMEOUT_SECONDS:-30}"
UPSTREAM_URL="${UPSTREAM_URL:-http://demo-backend:8081}"

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "required command is not installed: $1" >&2
		exit 1
	fi
}

require_command curl
require_command jq

if ! [[ "$E2E_TIMEOUT_SECONDS" =~ ^[1-9][0-9]*$ ]]; then
	echo "E2E_TIMEOUT_SECONDS must be a positive integer" >&2
	exit 1
fi

RUN_ID="$(date +%s)-$$-${RANDOM}"
PROJECT_NAME="e2e-project-${RUN_ID}"
SERVICE_NAME="e2e-demo-backend-${RUN_ID}"
ROUTE_NAME="e2e-demo-route-${RUN_ID}"
PATH_PREFIX="/e2e-${RUN_ID}"
GATEWAY_ORDERS_URL="${GATEWAY_URL}${PATH_PREFIX}/orders"

response_file="$(mktemp)"
trap 'rm -f "$response_file"' EXIT

echo "Checking service health..."
curl -fsS "${CONTROL_PLANE_URL}/health" >/dev/null
curl -fsS "${GATEWAY_URL}/health" >/dev/null

status_before="$(curl -sS -o /dev/null -w '%{http_code}' "$GATEWAY_ORDERS_URL")"
if [[ "$status_before" != "404" ]]; then
	echo "expected an unknown route to return HTTP 404, got ${status_before}" >&2
	exit 1
fi

echo "Creating project ${PROJECT_NAME}..."
project_response="$({
	curl -fsS -X POST "${CONTROL_PLANE_URL}/api/v1/projects" \
		-H "Content-Type: application/json" \
		-d "$(jq -nc --arg name "$PROJECT_NAME" '{name: $name}')"
})"
PROJECT_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$project_response")"

echo "Creating service ${SERVICE_NAME}..."
service_response="$({
	curl -fsS -X POST "${CONTROL_PLANE_URL}/api/v1/projects/${PROJECT_ID}/services" \
		-H "Content-Type: application/json" \
		-d "$(jq -nc \
			--arg name "$SERVICE_NAME" \
			--arg upstream_url "$UPSTREAM_URL" \
			'{name: $name, upstream_url: $upstream_url}')"
})"
SERVICE_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$service_response")"

echo "Creating dynamic route ${PATH_PREFIX}..."
route_response="$({
	curl -fsS -X POST "${CONTROL_PLANE_URL}/api/v1/services/${SERVICE_ID}/routes" \
		-H "Content-Type: application/json" \
		-d "$(jq -nc \
			--arg name "$ROUTE_NAME" \
			--arg path_prefix "$PATH_PREFIX" \
			'{
				name: $name,
				path_prefix: $path_prefix,
				strip_prefix: true,
				timeout_ms: 3000,
				enabled: true
			}')"
})"
ROUTE_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$route_response")"

snapshot="$({ curl -fsS "${CONTROL_PLANE_URL}/internal/v1/routes"; })"
if ! jq -e \
	--arg path_prefix "$PATH_PREFIX" \
	--arg upstream_url "$UPSTREAM_URL" \
	'[
		.[]
		| select(
			.path_prefix == $path_prefix
			and .upstream_url == $upstream_url
			and .strip_prefix == true
			and .timeout_ms == 3000
		)
	] | length == 1' <<<"$snapshot" >/dev/null; then
	echo "created route is missing from the Control Plane snapshot" >&2
	echo "$snapshot" | jq . >&2
	exit 1
fi

echo "Waiting for Gateway to load the new snapshot..."
deadline=$((SECONDS + E2E_TIMEOUT_SECONDS))
last_status=""

while ((SECONDS < deadline)); do
	last_status="$(curl -sS -o "$response_file" -w '%{http_code}' "$GATEWAY_ORDERS_URL" || true)"

	if [[ "$last_status" == "200" ]] && jq -e \
		'type == "array" and any(.[]; .id == "ord_1")' \
		"$response_file" >/dev/null 2>&1; then
		echo "MVP4 E2E test passed"
		echo "project_id=${PROJECT_ID}"
		echo "service_id=${SERVICE_ID}"
		echo "route_id=${ROUTE_ID}"
		echo "gateway_url=${GATEWAY_ORDERS_URL}"
		exit 0
	fi

	sleep 1
done

echo "Gateway did not apply the route within ${E2E_TIMEOUT_SECONDS}s" >&2
echo "last HTTP status: ${last_status:-request failed}" >&2
if [[ -s "$response_file" ]]; then
	echo "last response body:" >&2
	cat "$response_file" >&2
	echo >&2
fi

echo "Control Plane snapshot:" >&2
echo "$snapshot" | jq . >&2
echo "Inspect Gateway logs with: make compose-logs" >&2
exit 1
