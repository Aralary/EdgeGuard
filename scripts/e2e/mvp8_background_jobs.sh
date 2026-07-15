#!/usr/bin/env bash

set -euo pipefail

CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-http://localhost:8082}"
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
COMPOSE_FILE="${COMPOSE_FILE:-deployments/docker-compose.yml}"
E2E_RUNTIME="${E2E_RUNTIME:-compose}"
KUBE_CONTEXT="${KUBE_CONTEXT:-kind-edgeguard}"
K8S_NAMESPACE="${K8S_NAMESPACE:-edgeguard}"
POSTGRES_USER="${POSTGRES_USER:-edgeguard}"
POSTGRES_DB="${POSTGRES_DB:-edgeguard}"
E2E_TIMEOUT_SECONDS="${E2E_TIMEOUT_SECONDS:-120}"

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "required command is not installed: $1" >&2
		exit 1
	fi
}

json_post() {
	local url="$1"
	local body="$2"
	curl -fsS -X POST "$url" -H "Content-Type: application/json" -d "$body"
}

postgres_exec() {
	if [[ "$E2E_RUNTIME" == "kubernetes" ]]; then
		kubectl --context "$KUBE_CONTEXT" -n "$K8S_NAMESPACE" exec -i postgres-0 -- "$@"
	else
		docker compose -f "$COMPOSE_FILE" exec -T postgres "$@"
	fi
}

rabbitmq_exec() {
	if [[ "$E2E_RUNTIME" == "kubernetes" ]]; then
		kubectl --context "$KUBE_CONTEXT" -n "$K8S_NAMESPACE" exec -i rabbitmq-0 -- "$@"
	else
		docker compose -f "$COMPOSE_FILE" exec -T rabbitmq "$@"
	fi
}

notification_exec() {
	if [[ "$E2E_RUNTIME" == "kubernetes" ]]; then
		kubectl --context "$KUBE_CONTEXT" -n "$K8S_NAMESPACE" exec -i deployment/notification-worker -- "$@"
	else
		docker compose -f "$COMPOSE_FILE" exec -T notification-worker "$@"
	fi
}

notification_logs() {
	if [[ "$E2E_RUNTIME" == "kubernetes" ]]; then
		kubectl --context "$KUBE_CONTEXT" -n "$K8S_NAMESPACE" logs \
			deployment/notification-worker --all-containers=true --tail=200
	else
		docker compose -f "$COMPOSE_FILE" logs --no-color --tail=200 notification-worker
	fi
}

job_status() {
	local job_id="$1"
	curl -fsS "${CONTROL_PLANE_URL}/api/v1/jobs/${job_id}"
}

wait_for_job_status() {
	local job_id="$1"
	local expected_status="$2"
	local deadline=$((SECONDS + E2E_TIMEOUT_SECONDS))
	local response=""
	local current=""

	while ((SECONDS < deadline)); do
		response="$(job_status "$job_id" 2>/dev/null || true)"
		current="$(jq -r '.status // empty' <<<"$response" 2>/dev/null || true)"
		if [[ "$current" == "$expected_status" ]]; then
			printf '%s' "$response"
			return 0
		fi
		if [[ "$current" == "failed" && "$expected_status" != "failed" ]]; then
			echo "job ${job_id} failed while waiting for ${expected_status}: ${response}" >&2
			return 1
		fi
		sleep 1
	done

	echo "job ${job_id} did not reach status ${expected_status}; last response: ${response}" >&2
	return 1
}

queue_ready_count() {
	local queue="$1"
	rabbitmq_exec rabbitmqctl -q list_queues -p edgeguard name messages_ready | \
		awk -v target="$queue" '$1 == target {print $2}' | tr -d '[:space:]'
}

dump_diagnostics() {
	echo "--- background jobs ---" >&2
	postgres_exec psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "
SELECT id, type, status, current_attempt, max_attempts, last_error,
       result_message, output_path, affected_rows, updated_at
FROM background_jobs
ORDER BY created_at DESC
LIMIT 20;
" >&2 || true

	echo "--- RabbitMQ queues ---" >&2
	rabbitmq_exec rabbitmqctl -q list_queues -p edgeguard \
		name messages_ready messages_unacknowledged consumers >&2 || true

	echo "--- notification worker logs ---" >&2
	notification_logs >&2 || true
}

fail_with_diagnostics() {
	echo "$1" >&2
	exit 1
}

require_command curl
require_command jq
if [[ "$E2E_RUNTIME" == "kubernetes" ]]; then
	require_command kubectl
elif [[ "$E2E_RUNTIME" == "compose" ]]; then
	require_command docker
else
	echo "E2E_RUNTIME must be compose or kubernetes" >&2
	exit 1
fi

if ! [[ "$E2E_TIMEOUT_SECONDS" =~ ^[1-9][0-9]*$ ]]; then
	echo "E2E_TIMEOUT_SECONDS must be a positive integer" >&2
	exit 1
fi

RUN_ID="$(date +%s)-$$-${RANDOM}"
ORDER_ID="mvp8-order-${RUN_ID}"
PROJECT_NAME="mvp8-project-${RUN_ID}"
SERVICE_NAME="mvp8-demo-backend-${RUN_ID}"
ROUTE_NAME="mvp8-verification-route-${RUN_ID}"
PATH_PREFIX="/mvp8-${RUN_ID}"
GATEWAY_ORDERS_URL="${GATEWAY_URL}${PATH_PREFIX}/orders"

trap 'status=$?; if ((status != 0)); then dump_diagnostics; fi' EXIT

echo "Checking service health..."
curl -fsS "${CONTROL_PLANE_URL}/health" >/dev/null
curl -fsS "${GATEWAY_URL}/health" >/dev/null
rabbitmq_exec rabbitmq-diagnostics -q ping >/dev/null

echo "Checking unknown job response..."
unknown_status="$(curl -sS -o /dev/null -w '%{http_code}' \
	"${CONTROL_PLANE_URL}/api/v1/jobs/missing-${RUN_ID}")"
if [[ "$unknown_status" != "404" ]]; then
	fail_with_diagnostics "unknown job returned HTTP ${unknown_status}, want 404"
fi

echo "Creating project and route for webhook verification..."
project_response="$(json_post \
	"${CONTROL_PLANE_URL}/api/v1/projects" \
	"$(jq -nc --arg name "$PROJECT_NAME" '{name: $name}')")"
PROJECT_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$project_response")"

service_response="$(json_post \
	"${CONTROL_PLANE_URL}/api/v1/projects/${PROJECT_ID}/services" \
	"$(jq -nc \
		--arg name "$SERVICE_NAME" \
		--arg upstream_url 'http://demo-backend:8081' \
		'{name: $name, upstream_url: $upstream_url}')")"
SERVICE_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$service_response")"

route_response="$(json_post \
	"${CONTROL_PLANE_URL}/api/v1/services/${SERVICE_ID}/routes" \
	"$(jq -nc \
		--arg name "$ROUTE_NAME" \
		--arg path_prefix "$PATH_PREFIX" \
		'{
			name: $name,
			path_prefix: $path_prefix,
			strip_prefix: true,
			timeout_ms: 3000,
			enabled: true
		}')")"
ROUTE_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$route_response")"

echo "Waiting for Gateway to apply the MVP8 verification route..."
deadline=$((SECONDS + E2E_TIMEOUT_SECONDS))
route_status=""
while ((SECONDS < deadline)); do
	route_status="$(curl -sS -o /dev/null -w '%{http_code}' "$GATEWAY_ORDERS_URL" || true)"
	if [[ "$route_status" == "200" ]]; then
		break
	fi
	sleep 1
done
if [[ "$route_status" != "200" ]]; then
	fail_with_diagnostics "Gateway did not apply MVP8 route ${PATH_PREFIX}; last HTTP status: ${route_status:-request failed}"
fi

echo "Submitting successful webhook job..."
webhook_response="$(json_post \
	"${CONTROL_PLANE_URL}/api/v1/jobs/webhooks" \
	"$(jq -nc \
		--arg url 'http://demo-backend:8081/orders' \
		--arg id "$ORDER_ID" \
		'{
			url: $url,
			method: "POST",
			headers: {"Content-Type": "application/json"},
			body: {id: $id, status: "created", amount: 1500},
			max_attempts: 3
		}')")"
WEBHOOK_JOB_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$webhook_response")"
webhook_status="$(wait_for_job_status "$WEBHOOK_JOB_ID" succeeded)" || fail_with_diagnostics "webhook job failed"

if ! jq -e '
	.type == "jobs.webhook.deliver"
	and .status == "succeeded"
	and .current_attempt == 1
	and .result_message == "webhook delivered"
	and .last_error == null
' <<<"$webhook_status" >/dev/null; then
	fail_with_diagnostics "unexpected successful webhook status: ${webhook_status}"
fi

order_response="$(curl -fsS "${GATEWAY_ORDERS_URL}/${ORDER_ID}")"
if ! jq -e --arg id "$ORDER_ID" '.id == $id and .amount == 1500' <<<"$order_response" >/dev/null; then
	fail_with_diagnostics "webhook did not create expected order: ${order_response}"
fi

echo "Submitting permanently failing webhook job..."
dead_before="$(queue_ready_count edgeguard.jobs.dead.v1)"
dead_before="${dead_before:-0}"
permanent_response="$(json_post \
	"${CONTROL_PLANE_URL}/api/v1/jobs/webhooks" \
	'{"url":"http://demo-backend:8081/missing","method":"POST","body":{"message":"permanent failure"},"max_attempts":3}')"
PERMANENT_JOB_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$permanent_response")"
permanent_status="$(wait_for_job_status "$PERMANENT_JOB_ID" failed)" || fail_with_diagnostics "permanent webhook job did not fail"
if ! jq -e '
	.status == "failed"
	and .current_attempt == 1
	and (.last_error | type == "string" and length > 0)
' <<<"$permanent_status" >/dev/null; then
	fail_with_diagnostics "unexpected permanent failure status: ${permanent_status}"
fi

echo "Submitting retrying webhook job..."
retry_response="$(json_post \
	"${CONTROL_PLANE_URL}/api/v1/jobs/webhooks" \
	'{"url":"http://demo-backend:9999/orders","method":"POST","body":{"message":"retry failure"},"max_attempts":2}')"
RETRY_JOB_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$retry_response")"
retry_status="$(wait_for_job_status "$RETRY_JOB_ID" failed)" || fail_with_diagnostics "retrying webhook job did not reach failed"
if ! jq -e '
	.status == "failed"
	and .current_attempt == 2
	and .max_attempts == 2
	and (.last_error | type == "string" and length > 0)
' <<<"$retry_status" >/dev/null; then
	fail_with_diagnostics "unexpected retry failure status: ${retry_status}"
fi

deadline=$((SECONDS + E2E_TIMEOUT_SECONDS))
dead_after=0
while ((SECONDS < deadline)); do
	dead_after="$(queue_ready_count edgeguard.jobs.dead.v1 2>/dev/null || echo 0)"
	dead_after="${dead_after:-0}"
	if [[ "$dead_after" =~ ^[0-9]+$ ]] && ((dead_after >= dead_before + 2)); then
		break
	fi
	sleep 1
done
if ! [[ "$dead_before" =~ ^[0-9]+$ && "$dead_after" =~ ^[0-9]+$ ]] || ((dead_after < dead_before + 2)); then
	fail_with_diagnostics "DLQ messages_ready=${dead_after}, want at least $((dead_before + 2))"
fi

echo "Submitting report job..."
FROM="$(date -u -d '24 hours ago' +'%Y-%m-%dT%H:%M:%SZ')"
TO="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"

report_response="$(json_post \
	"${CONTROL_PLANE_URL}/api/v1/jobs/reports" \
	"$(jq -nc --arg project_id "$PROJECT_ID" --arg from "$FROM" --arg to "$TO" '
		{
			project_id: $project_id,
			from: $from,
			to: $to,
			format: "json",
			max_attempts: 3
		}')")"
REPORT_JOB_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$report_response")"
report_status="$(wait_for_job_status "$REPORT_JOB_ID" succeeded)" || fail_with_diagnostics "report job failed"
if ! jq -e --arg suffix "/${REPORT_JOB_ID}.json" '
	.status == "succeeded"
	and .result_message == "report generated"
	and (.output_path | endswith($suffix))
	and .affected_rows >= 0
' <<<"$report_status" >/dev/null; then
	fail_with_diagnostics "unexpected report status: ${report_status}"
fi

notification_exec test -f "/data/reports/${REPORT_JOB_ID}.json" || \
	fail_with_diagnostics "report file was not created"

report_project_id="$(notification_exec cat "/data/reports/${REPORT_JOB_ID}.json" | jq -r '.project_id')"
if [[ "$report_project_id" != "$PROJECT_ID" ]]; then
	fail_with_diagnostics "report project_id=${report_project_id}, want ${PROJECT_ID}"
fi

echo "Submitting cleanup job..."
cleanup_response="$(json_post \
	"${CONTROL_PLANE_URL}/api/v1/jobs/cleanup" \
	"$(jq -nc --arg before "$TO" '{before: $before, batch_size: 50, max_attempts: 3}')")"
CLEANUP_JOB_ID="$(jq -er '.id | strings | select(length > 0)' <<<"$cleanup_response")"
cleanup_status="$(wait_for_job_status "$CLEANUP_JOB_ID" succeeded)" || fail_with_diagnostics "cleanup job failed"
if ! jq -e '
	.status == "succeeded"
	and .result_message == "expired refresh tokens cleaned"
	and .affected_rows >= 0
' <<<"$cleanup_status" >/dev/null; then
	fail_with_diagnostics "unexpected cleanup status: ${cleanup_status}"
fi

trap - EXIT

echo "MVP8 background jobs E2E test passed"
echo "project_id=${PROJECT_ID}"
echo "service_id=${SERVICE_ID}"
echo "route_id=${ROUTE_ID}"
echo "webhook_job_id=${WEBHOOK_JOB_ID}"
echo "permanent_failure_job_id=${PERMANENT_JOB_ID}"
echo "retry_failure_job_id=${RETRY_JOB_ID}"
echo "report_job_id=${REPORT_JOB_ID}"
echo "cleanup_job_id=${CLEANUP_JOB_ID}"
