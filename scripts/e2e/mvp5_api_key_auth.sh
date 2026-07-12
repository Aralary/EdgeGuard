#!/usr/bin/env bash

set -euo pipefail

CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-http://localhost:8082}"
AUTH_URL="${AUTH_URL:-http://localhost:8083}"
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
UPSTREAM_URL="${UPSTREAM_URL:-http://demo-backend:8081}"
E2E_TIMEOUT_SECONDS="${E2E_TIMEOUT_SECONDS:-30}"

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
EMAIL="e2e-${RUN_ID}@example.com"
PASSWORD="password-${RUN_ID}"
PROJECT_NAME="e2e-protected-project-${RUN_ID}"
OTHER_PROJECT_NAME="e2e-other-project-${RUN_ID}"
SERVICE_NAME="e2e-protected-backend-${RUN_ID}"
ROUTE_NAME="e2e-protected-route-${RUN_ID}"
PATH_PREFIX="/e2e-protected-${RUN_ID}"
GATEWAY_ORDERS_URL="${GATEWAY_URL}${PATH_PREFIX}/orders"

response_file="$(mktemp)"
trap 'rm -f "$response_file"' EXIT

json_post() {
	local url="$1"
	local body="$2"
	shift 2
	curl -fsS -X POST "$url" -H "Content-Type: application/json" "$@" -d "$body"
}

echo "Checking service health..."
curl -fsS "${CONTROL_PLANE_URL}/health" >/dev/null
curl -fsS "${AUTH_URL}/health" >/dev/null
curl -fsS "${GATEWAY_URL}/health" >/dev/null

echo "Registering ${EMAIL}..."
json_post \
	"${AUTH_URL}/api/v1/auth/register" \
	"$(jq -nc --arg email "$EMAIL" --arg password "$PASSWORD" '{email: $email, password: $password}')" \
	>/dev/null

echo "Logging in..."
login_response="$({
	json_post \
		"${AUTH_URL}/api/v1/auth/login" \
		"$(jq -nc --arg email "$EMAIL" --arg password "$PASSWORD" '{email: $email, password: $password}')"
})"
ACCESS_TOKEN="$(jq -er '.tokens.access_token | strings | select(length > 0)' <<<"$login_response")"

echo "Creating protected project and route..."
project_response="$({
	json_post \
		"${CONTROL_PLANE_URL}/api/v1/projects" \
		"$(jq -nc --arg name "$PROJECT_NAME" '{name: $name}')"
})"
PROJECT_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$project_response")"

service_response="$({
	json_post \
		"${CONTROL_PLANE_URL}/api/v1/projects/${PROJECT_ID}/services" \
		"$(jq -nc \
			--arg name "$SERVICE_NAME" \
			--arg upstream_url "$UPSTREAM_URL" \
			'{name: $name, upstream_url: $upstream_url}')"
})"
SERVICE_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$service_response")"

route_response="$({
	json_post \
		"${CONTROL_PLANE_URL}/api/v1/services/${SERVICE_ID}/routes" \
		"$(jq -nc \
			--arg name "$ROUTE_NAME" \
			--arg path_prefix "$PATH_PREFIX" \
			'{
				name: $name,
				path_prefix: $path_prefix,
				strip_prefix: true,
				timeout_ms: 3000,
				enabled: true,
				auth_required: true
			}')"
})"
ROUTE_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$route_response")"

snapshot="$({ curl -fsS "${CONTROL_PLANE_URL}/internal/v1/routes"; })"
if ! jq -e \
	--arg project_id "$PROJECT_ID" \
	--arg path_prefix "$PATH_PREFIX" \
	--arg upstream_url "$UPSTREAM_URL" \
	'[
		.[]
		| select(
			.project_id == $project_id
			and .path_prefix == $path_prefix
			and .upstream_url == $upstream_url
			and .auth_required == true
		)
	] | length == 1' <<<"$snapshot" >/dev/null; then
	echo "protected route is missing from the Control Plane snapshot" >&2
	echo "$snapshot" | jq . >&2
	exit 1
fi

echo "Creating API key for protected project..."
api_key_response="$({
	json_post \
		"${AUTH_URL}/api/v1/projects/${PROJECT_ID}/api-keys" \
		"$(jq -nc --arg name "e2e-key-${RUN_ID}" '{name: $name}')" \
		-H "Authorization: Bearer ${ACCESS_TOKEN}"
})"
API_KEY_ID="$(jq -er '.api_key.id | strings | select(length > 0)' <<<"$api_key_response")"
API_KEY="$(jq -er '.value | strings | select(length > 0)' <<<"$api_key_response")"

echo "Creating API key for another project..."
other_project_response="$({
	json_post \
		"${CONTROL_PLANE_URL}/api/v1/projects" \
		"$(jq -nc --arg name "$OTHER_PROJECT_NAME" '{name: $name}')"
})"
OTHER_PROJECT_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$other_project_response")"

other_key_response="$({
	json_post \
		"${AUTH_URL}/api/v1/projects/${OTHER_PROJECT_ID}/api-keys" \
		"$(jq -nc --arg name "e2e-other-key-${RUN_ID}" '{name: $name}')" \
		-H "Authorization: Bearer ${ACCESS_TOKEN}"
})"
OTHER_API_KEY="$(jq -er '.value | strings | select(length > 0)' <<<"$other_key_response")"

echo "Waiting for Gateway to apply protected route..."
deadline=$((SECONDS + E2E_TIMEOUT_SECONDS))
last_status=""
while ((SECONDS < deadline)); do
	last_status="$(curl -sS -o /dev/null -w '%{http_code}' "$GATEWAY_ORDERS_URL" || true)"
	if [[ "$last_status" == "401" ]]; then
		break
	fi
	sleep 1
done

if [[ "$last_status" != "401" ]]; then
	echo "Gateway did not apply protected route within ${E2E_TIMEOUT_SECONDS}s" >&2
	echo "last HTTP status: ${last_status:-request failed}" >&2
	exit 1
fi

invalid_status="$(curl -sS -o /dev/null -w '%{http_code}' \
	-H "X-API-Key: invalid" \
	"$GATEWAY_ORDERS_URL")"
if [[ "$invalid_status" != "401" ]]; then
	echo "invalid API key returned HTTP ${invalid_status}, want 401" >&2
	exit 1
fi

wrong_project_status="$(curl -sS -o /dev/null -w '%{http_code}' \
	-H "X-API-Key: ${OTHER_API_KEY}" \
	"$GATEWAY_ORDERS_URL")"
if [[ "$wrong_project_status" != "403" ]]; then
	echo "API key from another project returned HTTP ${wrong_project_status}, want 403" >&2
	exit 1
fi

valid_status="$(curl -sS -o "$response_file" -w '%{http_code}' \
	-H "X-API-Key: ${API_KEY}" \
	"$GATEWAY_ORDERS_URL")"
if [[ "$valid_status" != "200" ]]; then
	echo "valid API key returned HTTP ${valid_status}, want 200" >&2
	cat "$response_file" >&2
	exit 1
fi
if ! jq -e 'type == "array" and any(.[]; .id == "ord_1")' "$response_file" >/dev/null; then
	echo "Gateway response does not contain ord_1" >&2
	cat "$response_file" >&2
	exit 1
fi

keys_response="$({
	curl -fsS \
		"${AUTH_URL}/api/v1/projects/${PROJECT_ID}/api-keys" \
		-H "Authorization: Bearer ${ACCESS_TOKEN}"
})"
if ! jq -e --arg id "$API_KEY_ID" \
	'any(.[]; .id == $id and .last_used_at != null)' <<<"$keys_response" >/dev/null; then
	echo "API key last_used_at was not updated" >&2
	echo "$keys_response" | jq . >&2
	exit 1
fi

echo "Revoking API key..."
curl -fsS -X DELETE \
	"${AUTH_URL}/api/v1/projects/${PROJECT_ID}/api-keys/${API_KEY_ID}" \
	-H "Authorization: Bearer ${ACCESS_TOKEN}" \
	-o /dev/null

revoked_status="$(curl -sS -o /dev/null -w '%{http_code}' \
	-H "X-API-Key: ${API_KEY}" \
	"$GATEWAY_ORDERS_URL")"
if [[ "$revoked_status" != "401" ]]; then
	echo "revoked API key returned HTTP ${revoked_status}, want 401" >&2
	exit 1
fi

echo "MVP5 API key E2E test passed"
echo "user_email=${EMAIL}"
echo "project_id=${PROJECT_ID}"
echo "service_id=${SERVICE_ID}"
echo "route_id=${ROUTE_ID}"
echo "api_key_id=${API_KEY_ID}"
echo "gateway_url=${GATEWAY_ORDERS_URL}"
