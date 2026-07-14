# Kubernetes-развёртывание EdgeGuard

Этот документ описывает локальное Kubernetes-развёртывание EdgeGuard, реализованное в MVP10.

Стенд предназначен для разработки, демонстрации и интеграционного тестирования всей системы в Kubernetes. Он воспроизводит production-like подходы — отдельные workloads, persistent volumes, probes, resource limits, migrations Job, observability и Ingress — но не является отказоустойчивой production-конфигурацией.

## Состав стенда

### EdgeGuard applications

| Компонент | Тип workload | Порт | Назначение |
|---|---|---:|---|
| Gateway | Deployment | 8080 | Внешний data plane, маршрутизация и reverse proxy |
| Demo Backend | Deployment | 8081 | Тестовый upstream Orders API |
| Control Plane | Deployment | 8082 | Управление проектами, сервисами, маршрутами и jobs |
| Auth | Deployment | 8083 | Пользователи, JWT и API keys |
| Analytics Worker | Deployment | 8084 | Обработка Kafka access events |
| Notification Worker | Deployment | 8085 | Обработка RabbitMQ background jobs |

### Stateful-инфраструктура

| Компонент | Тип workload | Хранилище | Назначение |
|---|---|---:|---|
| PostgreSQL | StatefulSet | 5Gi | Основное хранилище Control Plane, Auth и Analytics |
| Redis | StatefulSet | 1Gi | Распределённые rate-limit counters |
| Kafka | StatefulSet | 5Gi | Поток access events, KRaft без ZooKeeper |
| RabbitMQ | StatefulSet | 2Gi | Очередь background jobs и DLQ |
| Prometheus | StatefulSet | 5Gi | Сбор и хранение метрик |
| Tempo | StatefulSet | 5Gi | Хранение distributed traces |
| Grafana | StatefulSet | 2Gi | Визуализация метрик и traces |
| OTel Collector | Deployment | — | Приём и экспорт OTLP traces в Tempo |

Notification Worker дополнительно использует PVC `reports-data` размером `1Gi` для JSON/CSV-отчётов.

## Структура Kubernetes-конфигурации

```text
deployments/k8s
├── base
│   ├── namespace.yaml
│   ├── service-account.yaml
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── migrations-job.yaml
│   ├── kafka-init-job.yaml
│   ├── gateway.yaml
│   ├── control-plane.yaml
│   ├── auth.yaml
│   ├── analytics-worker.yaml
│   ├── notification-worker.yaml
│   ├── demo-backend.yaml
│   ├── postgres.yaml
│   ├── redis.yaml
│   ├── kafka.yaml
│   ├── rabbitmq.yaml
│   ├── prometheus.yaml
│   ├── prometheus-rbac.yaml
│   ├── otel-collector.yaml
│   ├── tempo.yaml
│   ├── grafana.yaml
│   ├── reports-pvc.yaml
│   ├── observability
│   └── kustomization.yaml
├── kind
│   └── cluster.yaml
└── overlays
    └── local
        ├── kustomization.yaml
        ├── configmap-patch.yaml
        ├── secret-patch.yaml
        └── ingress.yaml
```

### Base

`deployments/k8s/base` содержит общие Kubernetes-ресурсы:

- application Deployments и Services;
- StatefulSets и persistent volumes;
- migrations и Kafka topic Jobs;
- Prometheus RBAC;
- observability configuration;
- probes, resources и security contexts;
- placeholder-образы и placeholder-секреты.

Base не следует применять в production без environment-specific overlay.

### Local overlay

`deployments/k8s/overlays/local` адаптирует base для Kind:

- заменяет GHCR-образы на `edgeguard-*:local`;
- устанавливает одну реплику каждого stateless-приложения;
- задаёт development-секреты;
- включает полный tracing sampling;
- добавляет Ingress для Gateway;
- использует `IngressClass/cloud-provider-kind`.

## Требования

На хосте должны быть установлены:

```text
Docker
kind
kubectl
curl
jq
```

Проверка:

```bash
docker version
kind version
kubectl version --client
curl --version
jq --version
```

Makefile также выполняет автоматическую проверку:

```bash
make k8s-check-tools
```

## Быстрый запуск

Если Docker Compose-стенд запущен, его рекомендуется остановить, чтобы не держать два комплекта инфраструктуры:

```bash
make compose-down
```

Запустить весь Kubernetes-стенд:

```bash
make k8s-up
```

Команда последовательно:

1. создаёт Kind-кластер `edgeguard`;
2. ждёт готовности control-plane и worker node;
3. запускает Cloud Provider KIND;
4. собирает семь локальных EdgeGuard-образов;
5. загружает образы в обе Kind-ноды;
6. выполняет server-side dry-run;
7. применяет local Kustomize overlay;
8. ожидает infrastructure StatefulSets;
9. ожидает migrations и Kafka topic Jobs;
10. ожидает application Deployments;
11. запускает Kubernetes smoke-тест.

Makefile всегда передаёт контекст явно:

```text
kind-edgeguard
```

Поэтому другой `current-context` в `~/.kube/config` не влияет на команды проекта.

## Основные Makefile-команды

| Команда | Назначение |
|---|---|
| `make k8s-render` | Собрать итоговый YAML локального overlay |
| `make k8s-cluster-create` | Создать Kind-кластер и запустить cloud provider |
| `make k8s-cloud-provider-up` | Перезапустить Cloud Provider KIND |
| `make k8s-images` | Собрать локальные EdgeGuard Docker images |
| `make k8s-load-images` | Загрузить локальные images в Kind-ноды |
| `make k8s-validate` | Выполнить server-side dry-run |
| `make k8s-deploy` | Применить overlay и перезапустить приложения |
| `make k8s-wait` | Дождаться готовности всех workloads и Jobs |
| `make k8s-up` | Выполнить полный цикл запуска и smoke-тест |
| `make k8s-status` | Показать состояние nodes и ресурсов EdgeGuard |
| `make k8s-smoke-test` | Повторно запустить быстрые интеграционные проверки |
| `make k8s-e2e-test` | Запустить полный E2E-набор MVP4–MVP9 |
| `make k8s-logs` | Показать логи application workloads |
| `make k8s-down` | Удалить cloud provider и Kind-кластер |

Параметры можно переопределять через переменные Makefile:

```bash
K8S_TIMEOUT_SECONDS=600 make k8s-up
LOCAL_IMAGE_TAG=dev make k8s-images
```

## Рендеринг и валидация

Проверить Kustomize без подключения к кластеру:

```bash
make k8s-render > /tmp/edgeguard-k8s.yaml
```

Это проверяет:

- синтаксис Kustomize;
- наличие всех resources и patches;
- генерацию ConfigMap;
- применение image replacements и replica overrides.

После создания Kind-кластера выполнить серверную проверку Kubernetes API:

```bash
make k8s-validate
```

`server-side dry-run` проверяет ресурсы через API Server, но не сохраняет их в кластере.

## Локальные Docker-образы

Приложения используют теги:

```text
edgeguard-gateway:local
edgeguard-control-plane:local
edgeguard-auth:local
edgeguard-analytics-worker:local
edgeguard-notification-worker:local
edgeguard-demo-backend:local
edgeguard-migrate:local
```

Kind не видит образы хостового Docker daemon автоматически. После сборки они загружаются в ноды:

```bash
make k8s-load-images
```

Поскольку при повторной сборке тег остаётся `:local`, `make k8s-deploy` выполняет `rollout restart` application Deployments. Иначе существующие Pod могли бы продолжить использовать предыдущий image ID.

Для локальных образов используется:

```yaml
imagePullPolicy: IfNotPresent
```

## Порядок запуска и миграции

Kubernetes не гарантирует порядок создания ресурсов. Поэтому готовность обеспечивается явно.

### PostgreSQL migrations

Goose запускается отдельным Job:

```text
edgeguard-migrations
```

Job сначала ждёт PostgreSQL, затем выполняет `goose up`.

Приложения с PostgreSQL-зависимостью используют init containers и ждут появления конкретных таблиц:

| Приложение | Ожидаемая таблица |
|---|---|
| Auth | `users` |
| Control Plane | `background_jobs` |
| Analytics Worker | `gateway_access_events` |
| Notification Worker | `background_jobs` |

Таким образом Pod не стартует в промежутке между готовностью PostgreSQL и завершением миграций.

PostgreSQL использует:

```text
PGDATA=/var/lib/postgresql/data/pgdata
```

Данные хранятся во вложенной директории PVC, чтобы непривилегированный пользователь PostgreSQL мог корректно инициализировать каталог.

### Kafka topic

Отдельный Job `kafka-init` идемпотентно создаёт:

```text
edgeguard.gateway.access.v1
```

Параметры локального стенда:

```text
partitions:          3
replication-factor:  1
```

Analytics Worker ждёт фактического появления topic, а не только открытого Kafka-порта.

### RabbitMQ

Control Plane и Notification Worker ожидают доступность RabbitMQ до запуска основного контейнера.

Notification Worker создаёт и использует:

```text
edgeguard.jobs.main.v1
edgeguard.jobs.dead.v1
```

## Probes

Application workloads имеют три типа проверок:

- `startupProbe` — позволяет приложению завершить запуск и подключиться к зависимостям;
- `livenessProbe` — проверяет, что процесс жив и не завис;
- `readinessProbe` — исключает Pod из Service endpoints, пока обязательные зависимости не готовы.

Используемые endpoints:

```text
/health — состояние процесса
/ready  — готовность обязательных зависимостей
```

Infrastructure workloads используют подходящие HTTP или exec probes: `pg_isready`, `redis-cli ping`, Kafka broker API, RabbitMQ diagnostics, Prometheus, Tempo и Grafana health endpoints.

## Security context

EdgeGuard containers запускаются с числовым UID/GID `10001`:

```yaml
runAsNonRoot: true
runAsUser: 10001
runAsGroup: 10001
```

Для контейнеров включены:

```yaml
allowPrivilegeEscalation: false
readOnlyRootFilesystem: true
capabilities:
  drop:
    - ALL
```

Также используется:

```yaml
seccompProfile:
  type: RuntimeDefault
```

Stateful-компоненты используют собственные непривилегированные UID/GID и `fsGroup` для доступа к PVC.

ServiceAccount token для EdgeGuard applications не монтируется:

```yaml
automountServiceAccountToken: false
```

Доступ к Kubernetes API нужен только Prometheus для namespace-scoped Pod discovery. Его права ограничены `get`, `list` и `watch` для Pod в namespace `edgeguard`.

## Ingress и Cloud Provider KIND

Gateway публикуется через Kubernetes Ingress:

```text
Ingress edgeguard
    ↓
Service gateway:8080
    ↓
Gateway Pod
```

В local overlay указан класс:

```yaml
ingressClassName: cloud-provider-kind
```

Cloud Provider KIND запускается отдельным Docker-контейнером на хосте. Makefile:

- определяет endpoint текущего Docker context;
- поддерживает rootless Docker socket;
- пробрасывает реальный Unix socket в контейнер;
- явно выбирает Docker provider;
- отключает SELinux label isolation для socket mount на Fedora;
- ждёт появления `IngressClass/cloud-provider-kind`.

Проверить состояние:

```bash
docker inspect \
  --format 'status={{.State.Status}} restarts={{.RestartCount}}' \
  edgeguard-cloud-provider-kind

docker logs --tail=200 edgeguard-cloud-provider-kind
kubectl --context kind-edgeguard get ingressclass
```

Получить адрес Gateway:

```bash
INGRESS_IP="$(kubectl --context kind-edgeguard \
  -n edgeguard get ingress edgeguard \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"

curl "http://${INGRESS_IP}/health"
curl "http://${INGRESS_IP}/ready"
```

## Observability

### Prometheus

Prometheus обнаруживает Pod по annotations:

```yaml
prometheus.io/scrape: "true"
prometheus.io/path: /metrics
prometheus.io/port: "<service port>"
```

В метрики добавляются Kubernetes labels:

```text
kubernetes_app
kubernetes_namespace
kubernetes_pod
```

### Tracing

Приложения отправляют OTLP/gRPC traces:

```text
EdgeGuard services
    ↓
otel-collector:4317
    ↓
tempo:4317
```

Trace context распространяется через HTTP, Kafka headers и RabbitMQ headers.

### Grafana provisioning

При запуске автоматически создаются:

- Prometheus datasource с UID `prometheus`;
- Tempo datasource с UID `tempo`;
- dashboard `EdgeGuard Overview`.

Для локального overlay используются credentials:

```text
admin / admin
```

Для временного доступа к UI:

```bash
kubectl --context kind-edgeguard \
  -n edgeguard port-forward service/grafana 13000:3000
```

После этого Grafana доступна по адресу:

```text
http://127.0.0.1:13000
```

## Smoke-тест

```bash
make k8s-smoke-test
```

Smoke-тест проверяет:

- `/health` и `/ready` EdgeGuard-сервисов;
- доступность Gateway через Ingress;
- динамическую загрузку маршрута из Control Plane;
- proxy request в Demo Backend;
- Redis `PONG`;
- наличие Kafka topic;
- RabbitMQ и обе очереди;
- готовность Prometheus, Tempo и Grafana;
- provisioning Grafana datasources и dashboard.

В чистой Kubernetes-БД нет заранее созданного demo route. Поэтому smoke-тест сам создаёт временный project, service и route с уникальным path prefix, ждёт обновления Gateway snapshot и проверяет проксирование через Ingress.

## Полный E2E

```bash
make k8s-e2e-test
```

Запускаются те же сценарии, что и для Docker Compose:

```text
MVP4 — Dynamic Routes
MVP5 — API Key Authentication
MVP6 — Rate Limiting
MVP7 — Kafka Analytics
MVP8 — RabbitMQ Background Jobs
MVP9 — Observability
```

В Kubernetes-режиме тесты используют:

- Ingress для Gateway;
- временные `kubectl port-forward` для внутренних HTTP API;
- `kubectl exec` для PostgreSQL, Kafka и RabbitMQ;
- `kubectl logs` для диагностики;
- Kubernetes DNS для upstream URL.

Таймаут можно увеличить:

```bash
E2E_TIMEOUT_SECONDS=300 make k8s-e2e-test
```

## Диагностика

### Общий статус

```bash
make k8s-status
```

```bash
kubectl --context kind-edgeguard \
  -n edgeguard get pods,statefulsets,deployments,jobs,pvc,ingress -o wide
```

### События

```bash
kubectl --context kind-edgeguard \
  -n edgeguard get events \
  --sort-by=.lastTimestamp
```

### Проблемный Pod

```bash
kubectl --context kind-edgeguard \
  -n edgeguard describe pod <pod-name>

kubectl --context kind-edgeguard \
  -n edgeguard logs <pod-name> \
  --all-containers --tail=200
```

При ошибке `make k8s-wait` скрипт автоматически выводит:

- Pod status;
- StatefulSets, Deployments, Jobs, PVC и Ingress;
- последние events;
- `describe` неготовых Pod;
- текущие и previous logs.

### Ingress не получает ADDRESS

Проверить cloud provider:

```bash
docker inspect \
  --format 'status={{.State.Status}} restarting={{.State.Restarting}} restarts={{.RestartCount}}' \
  edgeguard-cloud-provider-kind

docker logs --tail=300 edgeguard-cloud-provider-kind
kubectl --context kind-edgeguard get ingressclass
kubectl --context kind-edgeguard \
  -n edgeguard get ingress edgeguard -o wide
```

Если в логах есть:

```text
Error: no supported container runtime found
```

следует проверить текущий Docker endpoint:

```bash
docker context inspect \
  --format '{{(index .Endpoints "docker").Host}}'
```

Он должен указывать на существующий Unix socket. Затем можно перезапустить provider:

```bash
make k8s-cloud-provider-up
```

### Проверка Gateway без Ingress

Для локализации проблемы можно временно использовать port-forward:

```bash
kubectl --context kind-edgeguard \
  -n edgeguard port-forward service/gateway 18080:8080
```

```bash
curl http://127.0.0.1:18080/health
curl http://127.0.0.1:18080/ready
```

`404` на `/api/v1/orders` в чистой БД сам по себе не означает неисправность Gateway: маршрут должен быть создан через Control Plane. Smoke- и E2E-тесты делают это автоматически.

### Повторное применение

Для повторного deployment без пересоздания кластера:

```bash
make k8s-images
make k8s-load-images
make k8s-validate
make k8s-deploy
make k8s-wait
make k8s-smoke-test
```

Или выполнить полный идемпотентный цикл:

```bash
make k8s-up
```

Migrations и Kafka init Jobs перед повторным применением удаляются и создаются заново, поскольку Job является one-shot workload и содержит immutable fields.

## Удаление стенда

```bash
make k8s-down
```

Удаляются:

- контейнер Cloud Provider KIND;
- Kind control-plane и worker nodes;
- Kubernetes resources;
- local PersistentVolumes кластера.

Локальные Docker images `edgeguard-*:local` остаются на хосте.

## Ограничения локальной конфигурации

Локальный стенд не является production HA-развёртыванием:

- PostgreSQL работает без replication и automatic failover;
- Redis работает без Sentinel или Cluster;
- Kafka использует один combined broker/controller;
- RabbitMQ работает как single node;
- persistent volumes привязаны к локальным Kind-нодам;
- секреты хранятся в development overlay;
- TLS для внешнего Ingress не настроен;
- отсутствуют PodDisruptionBudget и HorizontalPodAutoscaler;
- отсутствуют NetworkPolicy;
- observability storage не рассчитан на длительное хранение;
- контейнерные образы загружаются локально и не публикуются registry.

Для production следует использовать отдельный environment overlay, безопасное управление секретами, TLS, registry, backup policy, HA stateful services или managed services, autoscaling и сетевые политики.

## Критерии завершения MVP10

MVP10 считается завершённым, если успешно выполняются:

```bash
make k8s-render > /tmp/edgeguard-k8s.yaml
make k8s-up
make k8s-e2e-test
make k8s-status
```

И подтверждены:

- готовность всех nodes и workloads;
- успешное выполнение migrations и Kafka init Jobs;
- внешний адрес Ingress;
- работа динамического Gateway route;
- работоспособность PostgreSQL, Redis, Kafka и RabbitMQ;
- сбор метрик и traces;
- Grafana provisioning;
- прохождение smoke- и полного E2E-набора.
