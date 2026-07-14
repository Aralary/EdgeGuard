#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
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
start_forward service/prometheus 19090:9090 prometheus
start_forward service/tempo 13200:3200 tempo
start_forward service/grafana 13000:3000 grafana

wait_for_http "Control Plane" http://127.0.0.1:18082/health
wait_for_http "Auth" http://127.0.0.1:18083/health
wait_for_http "Analytics" http://127.0.0.1:18084/health
wait_for_http "Prometheus" http://127.0.0.1:19090/-/ready
wait_for_http "Tempo" http://127.0.0.1:13200/ready
wait_for_http "Grafana" http://127.0.0.1:13000/api/health

ingress_address="$(get_ingress_address)"
gateway_url="http://${ingress_address}"
wait_for_http "Gateway Ingress" "${gateway_url}/health"

export CONTROL_PLANE_URL=http://127.0.0.1:18082
export AUTH_URL=http://127.0.0.1:18083
export ANALYTICS_URL=http://127.0.0.1:18084
export GATEWAY_URL="$gateway_url"
export PROMETHEUS_URL=http://127.0.0.1:19090
export TEMPO_URL=http://127.0.0.1:13200
export GRAFANA_URL=http://127.0.0.1:13000
export GRAFANA_USER=admin
export GRAFANA_PASSWORD=admin
export UPSTREAM_URL=http://demo-backend:8081
export E2E_RUNTIME=kubernetes
export KUBE_CONTEXT
export K8S_NAMESPACE
export E2E_TIMEOUT_SECONDS="${E2E_TIMEOUT_SECONDS:-180}"

cd "$REPOSITORY_ROOT"
./scripts/e2e/mvp4_dynamic_routes.sh
./scripts/e2e/mvp5_api_key_auth.sh
./scripts/e2e/mvp6_rate_limiting.sh
./scripts/e2e/mvp7_analytics.sh
./scripts/e2e/mvp8_background_jobs.sh
./scripts/e2e/mvp9_observability.sh

echo "Kubernetes E2E test passed"
echo "gateway_url=${gateway_url}"
