#!/usr/bin/env bash

set -euo pipefail

KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-edgeguard}"
KUBE_CONTEXT="${KUBE_CONTEXT:-kind-${KIND_CLUSTER_NAME}}"
K8S_NAMESPACE="${K8S_NAMESPACE:-edgeguard}"
K8S_OVERLAY="${K8S_OVERLAY:-deployments/k8s/overlays/local}"
K8S_TIMEOUT_SECONDS="${K8S_TIMEOUT_SECONDS:-300}"

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

kubectl_edgeguard() {
	kubectl --context "$KUBE_CONTEXT" "$@"
}

require_kubernetes_tools() {
	require_command docker
	require_command kind
	require_command kubectl
	require_positive_integer K8S_TIMEOUT_SECONDS "$K8S_TIMEOUT_SECONDS"
}

require_cluster() {
	if ! kind get clusters 2>/dev/null | grep -Fxq "$KIND_CLUSTER_NAME"; then
		echo "Kind cluster ${KIND_CLUSTER_NAME} does not exist. Run: make k8s-cluster-create" >&2
		exit 1
	fi

	if ! kubectl_edgeguard cluster-info >/dev/null 2>&1; then
		echo "Kubernetes context ${KUBE_CONTEXT} is not reachable" >&2
		exit 1
	fi
}

wait_for_http() {
	local name="$1"
	local url="$2"
	local deadline=$((SECONDS + K8S_TIMEOUT_SECONDS))

	while ((SECONDS < deadline)); do
		if curl -fsS "$url" >/dev/null 2>&1; then
			echo "${name}: ready"
			return 0
		fi
		sleep 2
	done

	echo "${name} did not become ready: ${url}" >&2
	return 1
}

start_port_forward() {
	local resource="$1"
	local mapping="$2"
	local log_file="$3"

	kubectl_edgeguard -n "$K8S_NAMESPACE" port-forward "$resource" "$mapping" >"$log_file" 2>&1 &
	echo $!
}

stop_processes() {
	local pid
	for pid in "$@"; do
		if [[ -n "$pid" ]]; then
			kill "$pid" >/dev/null 2>&1 || true
			wait "$pid" >/dev/null 2>&1 || true
		fi
	done
}

get_ingress_address() {
	local deadline=$((SECONDS + K8S_TIMEOUT_SECONDS))
	local address=""

	while ((SECONDS < deadline)); do
		address="$(kubectl_edgeguard -n "$K8S_NAMESPACE" get ingress edgeguard \
			-o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || true)"
		if [[ -z "$address" ]]; then
			address="$(kubectl_edgeguard -n "$K8S_NAMESPACE" get ingress edgeguard \
				-o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>/dev/null || true)"
		fi
		if [[ -n "$address" ]]; then
			printf '%s' "$address"
			return 0
		fi
		sleep 2
	done

	echo "Ingress did not receive an external address" >&2
	return 1
}
