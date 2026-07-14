#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

require_kubernetes_tools
require_command curl
require_command jq
require_cluster

work_dir="$(mktemp -d)"
pids=()
cleanup() {
	if ((${#pids[@]})); then
		stop_processes "${pids[@]}"
	fi
	rm -rf "$work_dir"
}
trap cleanup EXIT

start_forward() {
	local resource="$1"
	local mapping="$2"
	local name="$3"
	local pid
	pid="$(start_port_forward "$resource" "$mapping" "${work_dir}/${name}.log")"
	pids+=("$pid")
}

start_forward service/control-plane 18082:8082 control-plane
start_forward service/auth 18083:8083 auth
start_forward service/analytics-worker 18084:8084 analytics
start_forward service/notification-worker 18085:8085 notification
start_forward service/prometheus 19090:9090 prometheus
start_forward service/tempo 13200:3200 tempo
start_forward service/grafana 13000:3000 grafana

wait_for_http "Control Plane" http://127.0.0.1:18082/health
wait_for_http "Auth" http://127.0.0.1:18083/health
wait_for_http "Analytics" http://127.0.0.1:18084/health
wait_for_http "Notification Worker" http://127.0.0.1:18085/health
wait_for_http "Prometheus" http://127.0.0.1:19090/-/ready
wait_for_http "Tempo" http://127.0.0.1:13200/ready
wait_for_http "Grafana" http://127.0.0.1:13000/api/health

for url in \
	http://127.0.0.1:18082/ready \
	http://127.0.0.1:18083/ready \
	http://127.0.0.1:18084/ready \
	http://127.0.0.1:18085/ready; do
	curl -fsS "$url" | jq -e '.status == "ready"' >/dev/null
done

ingress_address="$(get_ingress_address)"
gateway_url="http://${ingress_address}"
wait_for_http "Gateway Ingress" "${gateway_url}/health"
curl -fsS "${gateway_url}/ready" | jq -e '.status == "ready"' >/dev/null

# A fresh Kubernetes database does not contain the demo route. Create a
# temporary route through the Control Plane and verify that Gateway refreshes
# its route snapshot and proxies the request through the Ingress.
run_id="$(date +%s)-$$-${RANDOM}"
project_name="k8s-smoke-project-${run_id}"
service_name="k8s-smoke-demo-backend-${run_id}"
route_name="k8s-smoke-route-${run_id}"
path_prefix="/k8s-smoke-${run_id}"
gateway_orders_url="${gateway_url}${path_prefix}/orders"

project_response="$(curl -fsS -X POST http://127.0.0.1:18082/api/v1/projects \
	-H 'Content-Type: application/json' \
	-d "$(jq -nc --arg name "$project_name" '{name: $name}')")"
project_id="$(jq -er '.id | strings | select(length > 0)' <<<"$project_response")"

service_response="$(curl -fsS -X POST \
	"http://127.0.0.1:18082/api/v1/projects/${project_id}/services" \
	-H 'Content-Type: application/json' \
	-d "$(jq -nc \
		--arg name "$service_name" \
		--arg upstream_url 'http://demo-backend:8081' \
		'{name: $name, upstream_url: $upstream_url}')")"
service_id="$(jq -er '.id | strings | select(length > 0)' <<<"$service_response")"

curl -fsS -X POST \
	"http://127.0.0.1:18082/api/v1/services/${service_id}/routes" \
	-H 'Content-Type: application/json' \
	-d "$(jq -nc \
		--arg name "$route_name" \
		--arg path_prefix "$path_prefix" \
		'{
			name: $name,
			path_prefix: $path_prefix,
			strip_prefix: true,
			timeout_ms: 3000,
			enabled: true
		}')" >/dev/null

echo "Waiting for Gateway to load smoke route ${path_prefix}..."
response_file="${work_dir}/gateway-orders.json"
deadline=$((SECONDS + K8S_TIMEOUT_SECONDS))
last_status=""
while ((SECONDS < deadline)); do
	last_status="$(curl -sS -o "$response_file" -w '%{http_code}' "$gateway_orders_url" || true)"
	if [[ "$last_status" == "200" ]] && jq -e \
		'type == "array" and any(.[]; .id == "ord_1")' \
		"$response_file" >/dev/null 2>&1; then
		echo "Gateway dynamic route: ready"
		break
	fi
	sleep 1
done

if [[ "$last_status" != "200" ]] || ! jq -e \
	'type == "array" and any(.[]; .id == "ord_1")' \
	"$response_file" >/dev/null 2>&1; then
	echo "Gateway did not load the Kubernetes smoke route" >&2
	echo "last HTTP status: ${last_status:-request failed}" >&2
	[[ ! -s "$response_file" ]] || cat "$response_file" >&2
	exit 1
fi

kubectl_edgeguard -n "$K8S_NAMESPACE" exec redis-0 -- redis-cli ping | grep -Fxq PONG
kubectl_edgeguard -n "$K8S_NAMESPACE" exec kafka-0 -- \
	/opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list | \
	grep -Fxq edgeguard.gateway.access.v1
kubectl_edgeguard -n "$K8S_NAMESPACE" exec rabbitmq-0 -- rabbitmq-diagnostics -q ping >/dev/null
kubectl_edgeguard -n "$K8S_NAMESPACE" exec rabbitmq-0 -- \
	rabbitmqctl -q list_queues -p edgeguard name | grep -Fxq edgeguard.jobs.main.v1
kubectl_edgeguard -n "$K8S_NAMESPACE" exec rabbitmq-0 -- \
	rabbitmqctl -q list_queues -p edgeguard name | grep -Fxq edgeguard.jobs.dead.v1

curl -fsS -u admin:admin http://127.0.0.1:13000/api/datasources/uid/prometheus | \
	jq -e '.uid == "prometheus"' >/dev/null
curl -fsS -u admin:admin http://127.0.0.1:13000/api/datasources/uid/tempo | \
	jq -e '.uid == "tempo"' >/dev/null
curl -fsS -u admin:admin http://127.0.0.1:13000/api/dashboards/uid/edgeguard-overview | \
	jq -e '.dashboard.uid == "edgeguard-overview"' >/dev/null

echo "Kubernetes smoke test passed"
echo "gateway_url=${gateway_url}"
echo "grafana_port_forward=http://127.0.0.1:13000"
