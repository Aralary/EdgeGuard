#!/usr/bin/env bash

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

require_kubernetes_tools
require_cluster

wait_timeout="${K8S_TIMEOUT_SECONDS}s"

print_failure_diagnostics() {
	local exit_code=$?
	local pod ready

	trap - ERR
	set +e

	echo >&2
	echo "Kubernetes workload did not become ready. Collecting diagnostics..." >&2
	kubectl_edgeguard -n "$K8S_NAMESPACE" get pods -o wide >&2
	echo >&2
	kubectl_edgeguard -n "$K8S_NAMESPACE" get statefulsets,deployments,jobs,pvc,ingress >&2
	echo >&2
	kubectl_edgeguard -n "$K8S_NAMESPACE" get events \
		--sort-by=.lastTimestamp | tail -n 100 >&2

	for pod in $(kubectl_edgeguard -n "$K8S_NAMESPACE" get pods -o name); do
		ready="$(kubectl_edgeguard -n "$K8S_NAMESPACE" get "$pod" \
			-o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null)"
		if [[ "$ready" == "True" ]]; then
			continue
		fi

		echo >&2
		echo "===== describe ${pod} =====" >&2
		kubectl_edgeguard -n "$K8S_NAMESPACE" describe "$pod" >&2
		echo >&2
		echo "===== logs ${pod} =====" >&2
		kubectl_edgeguard -n "$K8S_NAMESPACE" logs "$pod" \
			--all-containers=true --tail=200 >&2
		echo >&2
		echo "===== previous logs ${pod} =====" >&2
		kubectl_edgeguard -n "$K8S_NAMESPACE" logs "$pod" \
			--all-containers=true --previous --tail=200 >&2
	done

	exit "$exit_code"
}

trap print_failure_diagnostics ERR

echo "Waiting for PostgreSQL, Redis, Kafka, RabbitMQ, Prometheus, Tempo and Grafana..."
for statefulset in postgres redis kafka rabbitmq prometheus tempo grafana; do
	echo "Waiting for StatefulSet/${statefulset}..."
	kubectl_edgeguard -n "$K8S_NAMESPACE" rollout status \
		"statefulset/${statefulset}" --timeout="$wait_timeout"
done

echo "Waiting for OpenTelemetry Collector..."
kubectl_edgeguard -n "$K8S_NAMESPACE" rollout status \
	deployment/otel-collector --timeout="$wait_timeout"

echo "Waiting for database migrations..."
kubectl_edgeguard -n "$K8S_NAMESPACE" wait \
	--for=condition=complete \
	job/edgeguard-migrations \
	--timeout="$wait_timeout"

echo "Waiting for Kafka topic initialization..."
kubectl_edgeguard -n "$K8S_NAMESPACE" wait \
	--for=condition=complete \
	job/kafka-init \
	--timeout="$wait_timeout"

echo "Waiting for EdgeGuard applications..."
for deployment in gateway control-plane auth analytics-worker notification-worker demo-backend; do
	echo "Waiting for Deployment/${deployment}..."
	kubectl_edgeguard -n "$K8S_NAMESPACE" rollout status \
		"deployment/${deployment}" --timeout="$wait_timeout"
done

echo "All Kubernetes workloads are ready"
