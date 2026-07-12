#!/usr/bin/env bash

set -euo pipefail

CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-http://localhost:8082}"
AUTH_URL="${AUTH_URL:-http://localhost:8083}"
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
UPSTREAM_URL="${UPSTREAM_URL:-http://demo-backend:8081}"
E2E_TIMEOUT_SECONDS="${E2E_TIMEOUT_SECONDS:-40}"
RATE_LIMIT_WINDOW_SECONDS="${RATE_LIMIT_WINDOW_SECONDS:-5}"

PUBLIC_RATE_LIMIT_REQUESTS=3
PROTECTED_RATE_LIMIT_REQUESTS=2

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

require_command curl
require_command jq
require_positive_integer E2E_TIMEOUT_SECONDS "$E2E_TIMEOUT_SECONDS"
require_positive_integer RATE_LIMIT_WINDOW_SECONDS "$RATE_LIMIT_WINDOW_SECONDS"

RUN_ID="$(date +%s)-$$-${RANDOM}"
EMAIL="e2e-rate-limit-${RUN_ID}@example.com"
PASSWORD="password-${RUN_ID}"
PROJECT_NAME="e2e-rate-limit-project-${RUN_ID}"
SERVICE_NAME="e2e-rate-limit-backend-${RUN_ID}"
PUBLIC_ROUTE_NAME="e2e-public-rate-limit-${RUN_ID}"
PROTECTED_ROUTE_NAME="e2e-protected-rate-limit-${RUN_ID}"
PUBLIC_PATH_PREFIX="/e2e-rate-public-${RUN_ID}"
PROTECTED_PATH_PREFIX="/e2e-rate-protected-${RUN_ID}"
PUBLIC_URL="${GATEWAY_URL}${PUBLIC_PATH_PREFIX}/orders"
PROTECTED_URL="${GATEWAY_URL}${PROTECTED_PATH_PREFIX}/orders"

headers_file="$(mktemp)"
body_file="$(mktemp)"
trap 'rm -f "$headers_file" "$body_file"' EXIT

json_post() {
	local url="$1"
	local body="$2"
	shift 2

	curl -fsS -X POST "$url" \
		-H "Content-Type: application/json" \
		"$@" \
		-d "$body"
}

request() {
	local url="$1"
	shift

	: >"$headers_file"
	: >"$body_file"

	curl -sS \
		-D "$headers_file" \
		-o "$body_file" \
		-w '%{http_code}' \
		"$@" \
		"$url"
}

header_value() {
	local header_name="$1"

	awk -v expected="${header_name,,}:" '
		tolower($1) == expected {
			value = $2
			gsub("\\r", "", value)
		}
		END {
			print value
		}
	' "$headers_file"
}

assert_status() {
	local actual="$1"
	local expected="$2"
	local description="$3"

	if [[ "$actual" != "$expected" ]]; then
		echo "${description}: got HTTP ${actual}, want ${expected}" >&2
		if [[ -s "$body_file" ]]; then
			echo "response body:" >&2
			cat "$body_file" >&2
			echo >&2
		fi
		exit 1
	fi
}

assert_rate_limit_headers() {
	local expected_limit="$1"
	local expected_remaining="$2"

	local actual_limit
	local actual_remaining
	local reset_at

	actual_limit="$(header_value X-RateLimit-Limit)"
	actual_remaining="$(header_value X-RateLimit-Remaining)"
	reset_at="$(header_value X-RateLimit-Reset)"

	if [[ "$actual_limit" != "$expected_limit" ]]; then
		echo "X-RateLimit-Limit=${actual_limit:-missing}, want ${expected_limit}" >&2
		exit 1
	fi
	if [[ "$actual_remaining" != "$expected_remaining" ]]; then
		echo "X-RateLimit-Remaining=${actual_remaining:-missing}, want ${expected_remaining}" >&2
		exit 1
	fi
	if ! [[ "$reset_at" =~ ^[1-9][0-9]*$ ]]; then
		echo "X-RateLimit-Reset is missing or invalid: ${reset_at:-missing}" >&2
		exit 1
	fi
}

wait_until_next_window() {
	local reset_at="$1"
	local now
	local wait_seconds

	now="$(date +%s)"
	wait_seconds=$((reset_at - now + 1))
	if ((wait_seconds > 0)); then
		sleep "$wait_seconds"
	fi
}

wait_for_rate_limited_route() {
	local url="$1"
	shift

	local deadline=$((SECONDS + E2E_TIMEOUT_SECONDS))
	local status=""
	local reset_at=""

	while ((SECONDS < deadline)); do
		status="$(request "$url" "$@" || true)"
		if [[ "$status" == "200" || "$status" == "429" ]]; then
			reset_at="$(header_value X-RateLimit-Reset)"
			if [[ "$reset_at" =~ ^[1-9][0-9]*$ ]]; then
				wait_until_next_window "$reset_at"
				return 0
			fi
		fi
		sleep 1
	done

	echo "Gateway did not apply rate-limited route within ${E2E_TIMEOUT_SECONDS}s" >&2
	echo "last HTTP status: ${status:-request failed}" >&2
	exit 1
}

echo "Checking service health..."
curl -fsS "${CONTROL_PLANE_URL}/health" >/dev/null
curl -fsS "${AUTH_URL}/health" >/dev/null
curl -fsS "${GATEWAY_URL}/health" >/dev/null

echo "Registering and logging in ${EMAIL}..."
json_post \
	"${AUTH_URL}/api/v1/auth/register" \
	"$(jq -nc --arg email "$EMAIL" --arg password "$PASSWORD" '{email: $email, password: $password}')" \
	>/dev/null

login_response="$({
	json_post \
		"${AUTH_URL}/api/v1/auth/login" \
		"$(jq -nc --arg email "$EMAIL" --arg password "$PASSWORD" '{email: $email, password: $password}')"
})"
ACCESS_TOKEN="$(jq -er '.tokens.access_token | strings | select(length > 0)' <<<"$login_response")"

echo "Creating project and upstream service..."
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

echo "Creating public rate-limited route..."
public_route_response="$({
	json_post \
		"${CONTROL_PLANE_URL}/api/v1/services/${SERVICE_ID}/routes" \
		"$(jq -nc \
			--arg name "$PUBLIC_ROUTE_NAME" \
			--arg path_prefix "$PUBLIC_PATH_PREFIX" \
			--argjson requests "$PUBLIC_RATE_LIMIT_REQUESTS" \
			--argjson window "$RATE_LIMIT_WINDOW_SECONDS" \
			'{
				name: $name,
				path_prefix: $path_prefix,
				strip_prefix: true,
				timeout_ms: 3000,
				enabled: true,
				auth_required: false,
				rate_limit_enabled: true,
				rate_limit_requests: $requests,
				rate_limit_window_seconds: $window
			}')"
})"
PUBLIC_ROUTE_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$public_route_response")"

echo "Creating protected rate-limited route..."
protected_route_response="$({
	json_post \
		"${CONTROL_PLANE_URL}/api/v1/services/${SERVICE_ID}/routes" \
		"$(jq -nc \
			--arg name "$PROTECTED_ROUTE_NAME" \
			--arg path_prefix "$PROTECTED_PATH_PREFIX" \
			--argjson requests "$PROTECTED_RATE_LIMIT_REQUESTS" \
			--argjson window "$RATE_LIMIT_WINDOW_SECONDS" \
			'{
				name: $name,
				path_prefix: $path_prefix,
				strip_prefix: true,
				timeout_ms: 3000,
				enabled: true,
				auth_required: true,
				rate_limit_enabled: true,
				rate_limit_requests: $requests,
				rate_limit_window_seconds: $window
			}')"
})"
PROTECTED_ROUTE_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$protected_route_response")"

snapshot="$(curl -fsS "${CONTROL_PLANE_URL}/internal/v1/routes")"
if ! jq -e \
	--arg public_path "$PUBLIC_PATH_PREFIX" \
	--arg protected_path "$PROTECTED_PATH_PREFIX" \
	--argjson public_limit "$PUBLIC_RATE_LIMIT_REQUESTS" \
	--argjson protected_limit "$PROTECTED_RATE_LIMIT_REQUESTS" \
	--argjson window "$RATE_LIMIT_WINDOW_SECONDS" \
	'(
		[.[] | select(
			.path_prefix == $public_path
			and .auth_required == false
			and .rate_limit_enabled == true
			and .rate_limit_requests == $public_limit
			and .rate_limit_window_seconds == $window
		)] | length == 1
	) and (
		[.[] | select(
			.path_prefix == $protected_path
			and .auth_required == true
			and .rate_limit_enabled == true
			and .rate_limit_requests == $protected_limit
			and .rate_limit_window_seconds == $window
		)] | length == 1
	)' <<<"$snapshot" >/dev/null; then
	echo "rate limit policies are missing from the Control Plane snapshot" >&2
	echo "$snapshot" | jq . >&2
	exit 1
fi

echo "Creating two API keys for counter isolation test..."
api_key_a_response="$({
	json_post \
		"${AUTH_URL}/api/v1/projects/${PROJECT_ID}/api-keys" \
		"$(jq -nc --arg name "e2e-rate-key-a-${RUN_ID}" '{name: $name}')" \
		-H "Authorization: Bearer ${ACCESS_TOKEN}"
})"
API_KEY_A="$(jq -er '.value | strings | select(length > 0)' <<<"$api_key_a_response")"

api_key_b_response="$({
	json_post \
		"${AUTH_URL}/api/v1/projects/${PROJECT_ID}/api-keys" \
		"$(jq -nc --arg name "e2e-rate-key-b-${RUN_ID}" '{name: $name}')" \
		-H "Authorization: Bearer ${ACCESS_TOKEN}"
})"
API_KEY_B="$(jq -er '.value | strings | select(length > 0)' <<<"$api_key_b_response")"

echo "Waiting for Gateway to apply public rate-limited route..."
wait_for_rate_limited_route "$PUBLIC_URL"

echo "Checking public route limit by client IP..."
for request_number in 1 2 3; do
	status="$(request "$PUBLIC_URL")"
	assert_status "$status" "200" "public request ${request_number}"
	assert_rate_limit_headers \
		"$PUBLIC_RATE_LIMIT_REQUESTS" \
		"$((PUBLIC_RATE_LIMIT_REQUESTS - request_number))"
done

status="$(request "$PUBLIC_URL")"
assert_status "$status" "429" "public request above limit"
assert_rate_limit_headers "$PUBLIC_RATE_LIMIT_REQUESTS" "0"
retry_after="$(header_value Retry-After)"
if ! [[ "$retry_after" =~ ^[1-9][0-9]*$ ]]; then
	echo "Retry-After is missing or invalid: ${retry_after:-missing}" >&2
	exit 1
fi

reset_at="$(header_value X-RateLimit-Reset)"
wait_until_next_window "$reset_at"
status="$(request "$PUBLIC_URL")"
assert_status "$status" "200" "public request after window reset"
assert_rate_limit_headers "$PUBLIC_RATE_LIMIT_REQUESTS" "$((PUBLIC_RATE_LIMIT_REQUESTS - 1))"

echo "Waiting for Gateway to apply protected rate-limited route..."
wait_for_rate_limited_route "$PROTECTED_URL" -H "X-API-Key: ${API_KEY_A}"

echo "Checking protected route limit by API key..."
for request_number in 1 2; do
	status="$(request "$PROTECTED_URL" -H "X-API-Key: ${API_KEY_A}")"
	assert_status "$status" "200" "protected request ${request_number} for key A"
	assert_rate_limit_headers \
		"$PROTECTED_RATE_LIMIT_REQUESTS" \
		"$((PROTECTED_RATE_LIMIT_REQUESTS - request_number))"
done

status="$(request "$PROTECTED_URL" -H "X-API-Key: ${API_KEY_A}")"
assert_status "$status" "429" "protected request above limit for key A"
assert_rate_limit_headers "$PROTECTED_RATE_LIMIT_REQUESTS" "0"

status="$(request "$PROTECTED_URL" -H "X-API-Key: ${API_KEY_B}")"
assert_status "$status" "200" "first protected request for independent key B"
assert_rate_limit_headers "$PROTECTED_RATE_LIMIT_REQUESTS" "$((PROTECTED_RATE_LIMIT_REQUESTS - 1))"

echo "MVP6 rate limiting E2E test passed"
echo "project_id=${PROJECT_ID}"
echo "service_id=${SERVICE_ID}"
echo "public_route_id=${PUBLIC_ROUTE_ID}"
echo "protected_route_id=${PROTECTED_ROUTE_ID}"
echo "public_gateway_url=${PUBLIC_URL}"
echo "protected_gateway_url=${PROTECTED_URL}"
