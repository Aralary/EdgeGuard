# Continuous Integration

EdgeGuard использует GitHub Actions workflow:

```text
.github/workflows/ci.yml
```

CI проверяет Go-код, Docker Compose-стенд и Kubernetes-манифесты. Полный Kubernetes E2E оставлен ручным, потому что он требует сборки семи локальных образов и запуска полного stateful-стенда.

## Триггеры

Workflow запускается автоматически:

- при любом `push`;
- при создании и обновлении `pull request`.

Также доступен ручной запуск через `Actions → CI → Run workflow`.

Для ручного запуска предусмотрен параметр:

```text
run_kubernetes_e2e
```

Если он включён, дополнительно запускается полный Kubernetes smoke/E2E job.

## Права и конкурентность

Workflow использует минимальные права:

```yaml
permissions:
  contents: read
```

CI не публикует образы, не создаёт releases и не изменяет репозиторий.

Для одной ветки одновременно выполняется только актуальный workflow run:

```yaml
concurrency:
  group: ci-${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```

Если в ветку быстро отправлено несколько коммитов, устаревшая проверка отменяется.

## Job `Go quality`

Job использует версию Go из `go.mod` через `actions/setup-go`.

Проверки выполняются в следующем порядке:

```text
go mod download
      ↓
gofmt check
      ↓
go mod tidy + git diff
      ↓
go vet ./...
      ↓
go test -race -count=1 ./...
```

### Форматирование

CI запускает `gofmt` для всех Go-файлов. Проверка падает, если найден хотя бы один неотформатированный файл.

Локальная команда:

```bash
gofmt -w ./cmd ./internal
```

### Актуальность `go.mod` и `go.sum`

CI выполняет:

```bash
go mod tidy
git diff --exit-code -- go.mod go.sum
```

Если `go mod tidy` изменяет module files, разработчик должен зафиксировать эти изменения в коммите.

### `go vet`

Статический анализ:

```bash
go vet ./...
```

Он обнаруживает часть ошибок, которые компилятор допускает, например некорректные форматные строки или подозрительное копирование synchronization primitives.

### Unit tests и race detector

Тесты запускаются так:

```bash
go test -race -count=1 ./...
```

`-race` включает Go race detector, а `-count=1` отключает повторное использование test cache.

## Job `Docker Compose smoke and E2E`

Job валидирует Compose-конфигурацию:

```bash
docker compose -f deployments/docker-compose.yml config --quiet
```

Затем выполняет полный локальный сценарий:

```text
make compose-up
      ↓
make smoke-test
      ↓
make e2e-test
      ↓
make compose-reset
```

`make compose-up` собирает все EdgeGuard Docker-образы, включая migrations image.

Smoke-тест проверяет:

- health/readiness приложений;
- PostgreSQL migrations;
- Redis;
- Kafka topic;
- RabbitMQ queues;
- Prometheus;
- OpenTelemetry Collector и Tempo;
- Grafana.

E2E покрывает сценарии MVP4–MVP9:

- dynamic routes;
- API keys;
- rate limiting;
- Kafka analytics;
- RabbitMQ background jobs;
- metrics и tracing.

При ошибке workflow выводит:

```bash
docker compose ps -a
docker compose logs --tail=300
```

Cleanup выполняется с `if: always()`, поэтому контейнеры и volumes удаляются даже после падения теста.

## Job `Kubernetes manifests`

Этот job не запускает приложения и не загружает EdgeGuard-образы. Его задача — проверить Kubernetes-конфигурацию быстрее полного E2E.

Порядок:

```text
установка Kind
      ↓
Kustomize render
      ↓
создание временного Kind-кластера
      ↓
server-side dry-run
      ↓
удаление кластера
```

Используются версии:

```text
Kind node: Kubernetes 1.35.0
Kind CLI:  v0.31.0
```

Они соответствуют локальному Kubernetes-стенду проекта.

### Kustomize render

```bash
make k8s-render
```

Проверяются resources, patches, ConfigMap generators, image replacements и replica overrides.

### Server-side validation

```bash
make k8s-validate \
  KIND_CLUSTER_NAME=edgeguard-ci \
  KUBE_CONTEXT=kind-edgeguard-ci
```

В отличие от клиентского dry-run, server-side validation проверяет объекты через API Server конкретной версии Kubernetes.

Временный кластер удаляется независимо от результата job.

## Ручной job `Kubernetes smoke and E2E`

Полный Kubernetes job запускается только через `workflow_dispatch` с включённым `run_kubernetes_e2e`.

Он выполняет:

```text
make k8s-up
      ↓
make k8s-e2e-test
      ↓
make k8s-down
```

Внутри запускаются:

- двухузловой Kind-кластер;
- Cloud Provider KIND;
- локальная сборка и загрузка семи EdgeGuard images;
- PostgreSQL, Redis, Kafka и RabbitMQ;
- Prometheus, Tempo, OTel Collector и Grafana;
- migrations и Kafka init Jobs;
- Ingress;
- smoke-тест;
- полный E2E-набор.

Job сделан ручным, чтобы обычный commit не расходовал время на повторную сборку полного Kubernetes-стенда. Его следует запускать:

- перед merge крупных Kubernetes-изменений;
- после обновления Kind, Kubernetes или cloud-provider-kind;
- после изменений K8s probes, security contexts или persistent volumes;
- перед демонстрацией или release milestone.

## Локальное воспроизведение CI

Go-проверки:

```bash
go mod tidy
git diff --exit-code -- go.mod go.sum
go vet ./...
go test -race -count=1 ./...
```

Docker Compose:

```bash
make compose-up
make smoke-test
make e2e-test
make compose-reset
```

Kubernetes manifests:

```bash
make k8s-render > /tmp/edgeguard-k8s.yaml
make k8s-cluster-create
make k8s-validate
make k8s-down
```

Полный Kubernetes E2E:

```bash
make k8s-up
make k8s-e2e-test
make k8s-down
```

## Рекомендуемые branch protection rules

Для основной ветки рекомендуется потребовать успешное прохождение:

```text
Go quality
Docker Compose smoke and E2E
Kubernetes manifests
```

Ручной `Kubernetes smoke and E2E` не следует делать обязательным для каждого pull request.

Также полезно включить:

- запрет direct push в основную ветку;
- обязательный pull request;
- запрет merge при устаревшей ветке;
- хотя бы один review для изменений инфраструктуры.

## Публикация образов

Текущий workflow только проверяет проект. Он не публикует Docker images в GHCR.

Публикацию лучше реализовать отдельным release workflow, который:

- запускается по Git tag;
- авторизуется в `ghcr.io`;
- собирает versioned images;
- публикует immutable tags;
- при необходимости формирует SBOM и provenance attestations.

Это отделяет CI-проверки от release/deployment-процесса.
