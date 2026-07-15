# EdgeGuard

[![CI](https://github.com/aralary/edgeguard/actions/workflows/ci.yml/badge.svg)](https://github.com/aralary/edgeguard/actions/workflows/ci.yml)

**EdgeGuard** — API Gateway платформа на Go для публикации, защиты и мониторинга backend-сервисов.

Проект разрабатывается как production-like backend-система и демонстрирует типовые задачи backend-разработки: reverse proxy, управление маршрутами, авторизацию, rate limiting, асинхронную обработку событий, аналитику трафика, observability и контейнеризацию.

## Цель проекта

EdgeGuard — это не просто reverse proxy. Цель проекта — реализовать небольшую, но полноценную API Gateway платформу, состоящую из двух основных частей:

* **Data Plane** — принимает внешний трафик и проксирует запросы к backend-сервисам.
* **Control Plane** — управляет сервисами, маршрутами, политиками доступа, API-ключами, лимитами и аналитикой.

Проект создается как pet project для практики и демонстрации навыков middle/senior Go backend-разработчика.

## Планируемая функциональность

### Gateway Service

* Reverse proxy для backend-сервисов
* Маршрутизация запросов по path prefix
* Проксирование HTTP-запросов и ответов
* Проброс заголовков
* Таймауты для upstream-сервисов
* Retry policy
* Circuit breaker
* Request ID propagation
* Структурированные логи
* Prometheus-метрики

### Control Plane API

* Управление проектами
* Управление upstream-сервисами
* Управление маршрутами
* Динамическая конфигурация gateway
* Управление API-ключами
* Управление rate limit policy
* Audit log

### Auth Service

* Регистрация пользователей
* Авторизация
* JWT access tokens
* Refresh tokens
* API keys
* Хеширование API-ключей
* RBAC
* Service accounts

### Rate Limiting

* Лимиты по IP
* Лимиты по API-ключу
* Лимиты по маршруту
* Redis-based counters
* Fixed window algorithm
* Token bucket algorithm

### Analytics

* События gateway
* Количество запросов по маршрутам
* Error rate по маршрутам
* Статистика latency
* События превышения rate limit
* Kafka-based event streaming
* Фоновая агрегация статистики

### Background Jobs

* Webhook-уведомления
* Daily usage reports
* Очистка истекших токенов
* Ротация API-ключей
* RabbitMQ-based task queue

### Observability

* Структурированные логи
* Prometheus-метрики
* OpenTelemetry tracing
* Health checks
* Readiness checks
* Grafana dashboards

## Общая архитектура

```text
External Client
      |
      v
+-------------------+
|  Gateway Service  |
|-------------------|
| route matching    |
| auth check        |
| rate limiting     |
| reverse proxy     |
| metrics/logs      |
+---------+---------+
          |
          v
+-------------------+
|   Demo Backend    |
|   Orders API      |
+-------------------+


Admin Client
      |
      v
+----------------------+
|  Control Plane API   |
|----------------------|
| services             |
| routes               |
| policies             |
| API keys             |
| audit log            |
+----------+-----------+
           |
           v
+----------------------+
|      PostgreSQL      |
+----------------------+

Gateway events
      |
      v
+----------------------+
|        Kafka         |
+----------------------+
           |
           v
+----------------------+
|  Analytics Worker    |
+----------------------+

Background jobs
      |
      v
+----------------------+
|       RabbitMQ       |
+----------------------+
           |
           v
+----------------------+
| Notification Worker  |
+----------------------+

Metrics from all services
      |
      v
+----------------------+      +----------------------+
|      Prometheus      |----->|       Grafana        |
+----------------------+      | metrics and traces   |
                              +----------+-----------+
Traces from all services                 ^
      |                                  |
      v                                  |
+----------------------+      +----------+-----------+
|    OTel Collector    |----->|        Tempo         |
+----------------------+      +----------------------+
```

## Структура репозитория

```text
edgeguard
├── .github
│   └── workflows
│       └── ci.yml
│
├── cmd
│   ├── gateway
│   │   └── main.go
│   ├── control-plane
│   │   └── main.go
│   ├── auth
│   │   └── main.go
│   ├── demo-backend
│   │   └── main.go
│   ├── analytics-worker
│   │   └── main.go
│   └── notification-worker
│       └── main.go
│
├── internal
│   ├── gateway
│   ├── controlplane
│   ├── auth
│   ├── demo
│   └── platform
│
├── configs
│   ├── gateway.yaml
│   └── gateway.docker.yaml
│
├── deployments
│   ├── docker-compose.yml
│   ├── docker
│   │   ├── gateway.Dockerfile
│   │   ├── demo-backend.Dockerfile
│   │   ├── control-plane.Dockerfile
│   │   ├── auth.Dockerfile
│   │   └── migrations.Dockerfile
│   └── k8s
│       ├── base
│       ├── kind
│       │   └── cluster.yaml
│       └── overlays
│           └── local
│
├── scripts
│   ├── e2e
│   └── k8s
│
├── migrations
├── docs
│   ├── ci.md
│   └── kubernetes.md
│
├── api
│   └── openapi.yaml
│
├── Makefile
├── go.mod
└── README.md
```

## Локальный запуск

Полная инфраструктура через Docker Compose запускается одной командой:

```bash
make compose-up
```

Порядок запуска контролируется Docker Compose:

```text
PostgreSQL становится healthy
        ↓
migrate применяет Goose-миграции и завершается с кодом 0
        ↓
control-plane и auth подключаются к подготовленной базе данных
        ↓
gateway дожидается готовности control-plane, auth, Redis и demo-backend
        ↓
gateway загружает routes snapshot и запускает polling
        ↓
Redis хранит распределенные счетчики rate limit
        ↓
Kafka принимает access events, analytics-worker агрегирует их в PostgreSQL
        ↓
RabbitMQ принимает background jobs, notification-worker обрабатывает их
        ↓
Prometheus собирает метрики, OTel Collector экспортирует traces в Tempo
        ↓
Grafana визуализирует метрики и позволяет исследовать traces
```

Контейнер `edgeguard-migrate` является одноразовым. Состояние `Exited (0)` после запуска — нормальное: оно означает, что миграции успешно применены.

В Docker Compose Gateway использует Control Plane как источник маршрутов и Auth Service для проверки API-ключей:

```text
CONTROL_PLANE_URL=http://control-plane:8082
AUTH_SERVICE_URL=http://auth:8083
REDIS_URL=redis://redis:6379/0
REDIS_RATE_LIMIT_PREFIX=edgeguard:rate-limit
ROUTES_REFRESH_INTERVAL=10s
```

YAML-конфигурация остается стартовым fallback. Если Control Plane недоступен при запуске, Gateway продолжает работать с YAML и повторяет загрузку по таймеру. Успешный пустой snapshot (`[]`) считается валидной конфигурацией и очищает fallback-маршруты. Поэтому в новой пустой базе запросы к `/api/v1/orders` начнут работать только после создания service и route через Control Plane.

Проверка состояния и API:

```bash
make compose-ps
make migrate-status
make smoke-test
```

Остановить сервисы, сохранив данные PostgreSQL:

```bash
make compose-down
```

Полностью удалить сервисы вместе с данными PostgreSQL:

```bash
make compose-reset
```

`compose-reset` удаляет named volume `postgres_data`, поэтому использовать эту команду следует только для полного сброса локальной базы данных.

## Continuous Integration

GitHub Actions автоматически проверяет проект при `push` и в pull requests:

- форматирование, `go mod tidy`, `go vet` и unit tests с race detector;
- сборку полного Docker Compose-стенда, smoke- и E2E-тесты;
- Kustomize render и server-side validation во временном Kind-кластере.

Полный Kubernetes smoke/E2E запускается вручную через `workflow_dispatch`, чтобы не поднимать тяжёлый stateful-стенд на каждом коммите.

Подробности: [docs/ci.md](docs/ci.md).

## Roadmap

### MVP 1 — Static Gateway

Первая версия проекта фокусируется на реализации работающего reverse proxy со статической YAML-конфигурацией.

Планируемый scope:

* Gateway service
* Demo backend service
* YAML-конфигурация маршрутов
* Поиск маршрута по path prefix
* Проксирование запроса
* Проксирование ответа
* Strip prefix support
* Upstream timeout
* Request ID middleware
* Структурированные логи
* Docker Compose

Пример конфигурации:

```yaml
http:
  addr: ":8080"

routes:
  - name: orders
    path_prefix: /api/v1
    upstream_url: http://demo-backend:8081
    strip_prefix: true
    timeout_ms: 3000
```

Ожидаемый flow:

```text
GET /api/v1/orders
        |
        v
Gateway
        |
        v
GET http://demo-backend:8081/orders
```

### MVP 2 — Docker Compose local run

* Dockerfile для gateway
* Dockerfile для demo-backend
* Docker Compose запуск всей текущей системы
* Отдельная Docker-конфигурация gateway
* Healthcheck для gateway
* Healthcheck для demo-backend
* Makefile-команды для локального запуска

### MVP 3 — Control Plane API

* PostgreSQL
* Миграции
* Projects
* Services
* Routes
* Route policies
* OpenAPI-документация

### MVP 4 — Dynamic Gateway Configuration

* Загрузка конфигурации gateway из Control Plane
* Config polling
* In-memory route cache
* Обновление маршрутов без рестарта gateway

### MVP 5 — Auth Service

* Регистрация пользователей
* Login
* JWT access tokens
* Refresh tokens
* API keys
* Хеширование API-ключей
* Базовые роли пользователей в JWT (полный project-level RBAC запланирован отдельно)

### Реализованное поведение MVP 5

Auth Service доступен на `:8083` и предоставляет регистрацию, login, refresh/logout, управление API-ключами и внутреннюю проверку ключей.

Маршрут Control Plane может быть защищен политикой:

```json
{
  "auth_required": true
}
```

Control Plane включает в snapshot `project_id` и `auth_required`. Gateway для защищенного маршрута:

```text
X-API-Key
    ↓
POST auth:8083/internal/v1/api-keys/validate
    ↓
проверка hash, enabled и expires_at
    ↓
сверка project_id ключа с project_id маршрута
    ↓
proxy request в upstream
```

Коды ответа Gateway:

* `401` — ключ отсутствует, невалиден, истек или отозван;
* `403` — валидный ключ принадлежит другому проекту;
* `503` — Auth Service недоступен;
* `200` — ключ валиден и принадлежит проекту маршрута.

Заголовок `X-API-Key` удаляется перед проксированием, поэтому секрет не передается upstream-сервису.

### MVP 6 — Redis Rate Limiting

* Интеграция с Redis
* Fixed window rate limiting
* Rate limit by IP
* Rate limit by API key
* Rate limit by route

### Реализованное поведение MVP 6

Политика ограничения задается на уровне маршрута через Control Plane:

```json
{
  "rate_limit_enabled": true,
  "rate_limit_requests": 100,
  "rate_limit_window_seconds": 60
}
```

Control Plane сохраняет политику в PostgreSQL и передает ее Gateway в динамическом routes snapshot. Gateway использует распределенный fixed-window счетчик в Redis:

```text
route policy
    ↓
route + client + fixed window
    ↓
atomic Lua INCR + PEXPIRE in Redis
    ↓
allowed request or HTTP 429
```

Идентификатор клиента выбирается так:

* для публичного маршрута — прямой IP соединения;
* для защищенного маршрута — `api_key_id`, полученный после проверки ключа в Auth Service.

Разрешенные ответы содержат `X-RateLimit-Limit`, `X-RateLimit-Remaining` и `X-RateLimit-Reset`. При превышении дополнительно возвращается `Retry-After` и статус `429 Too Many Requests`.

При временной недоступности Redis Gateway работает в режиме fail-open: пишет warning и продолжает проксирование, чтобы отказ дополнительного защитного механизма не остановил весь API-трафик.

### MVP 7 — Kafka Analytics

* Gateway access events
* Асинхронный Kafka producer
* Analytics consumer group
* Идемпотентное хранение raw events
* Почасовая агрегация статистики
* HTTP API сводки и временного ряда

### Реализованное поведение MVP 7

Gateway публикует версионированные access events в topic `edgeguard.gateway.access.v1`. Публикация не блокирует основной HTTP-трафик и работает в режиме fail-open: проблемы Kafka логируются, но не изменяют ответ клиенту.

Analytics Service читает сообщения consumer group `edgeguard-analytics-v1`. Offset подтверждается только после успешной PostgreSQL-транзакции. `event_id` используется как primary key, поэтому повторная Kafka delivery не увеличивает агрегаты повторно.

Каждое событие хранится в `gateway_access_events`. События известных маршрутов дополнительно агрегируются в `gateway_route_stats_hourly` по проекту, маршруту, HTTP-методу и часовому bucket.

Analytics HTTP API доступен на `:8084`:

```text
GET /api/v1/projects/:project_id/analytics/summary
GET /api/v1/projects/:project_id/analytics/hourly
```

Поддерживаются query-параметры `from`, `to`, `route_name`, `method`; для hourly endpoint также поддерживается `limit`. `from` и `to` передаются в RFC3339. Поскольку данные агрегируются почасово, интервал расширяется до границ соответствующих часовых buckets. Максимальный диапазон одного запроса — 90 дней.

### MVP 8 — RabbitMQ Background Jobs

* Durable RabbitMQ task queue
* Publisher confirms и persistent messages
* Notification worker с manual acknowledgements
* Webhook delivery с SSRF-защитой
* JSON/CSV report generation
* Cleanup expired refresh tokens
* Delayed retry для quorum queue
* Dead-letter exchange и DLQ
* PostgreSQL job statuses
* HTTP API чтения статуса задачи

### Реализованное поведение MVP 8

Control Plane принимает фоновые задачи через:

```text
POST /api/v1/jobs/webhooks
POST /api/v1/jobs/reports
POST /api/v1/jobs/cleanup
GET  /api/v1/jobs/:job_id
```

Перед публикацией создается запись в `background_jobs`. HTTP `202 Accepted` возвращается только после publisher confirmation от RabbitMQ. Если публикация не подтверждена, запись получает `publish_failed`; если broker все же принял сообщение, worker может перевести ее в `processing`, поскольку потеря confirmation является неоднозначным результатом. Статус задачи проходит состояния:

```text
publishing → queued → processing → succeeded
     ↓                   ↓
publish_failed       retrying → processing
                         ↓
                       failed → DLQ
```

Notification Worker использует manual acknowledgements. Успешная задача подтверждается только после сохранения статуса `succeeded` в PostgreSQL. При временной ошибке задача возвращается в quorum queue с delayed retry, а при permanent error или исчерпании `max_attempts` получает статус `failed` и отправляется в `edgeguard.jobs.dead.v1`.

Для защиты от повторного выполнения после сбоя между PostgreSQL update и RabbitMQ ack worker проверяет финальный статус. Повторно доставленная задача со статусом `succeeded` подтверждается без выполнения side effect еще раз.

Webhook worker запрещает private, loopback, link-local и другие служебные адреса, если host явно не добавлен в allowlist. Report jobs создают атомарно публикуемые JSON/CSV-файлы в volume `reports_data`. Cleanup jobs удаляют истекшие и давно отозванные refresh tokens ограниченными batch-операциями.

Пример ответа статуса:

```json
{
  "id": "4ebff7b39889f87063ae609ac254a31c",
  "type": "jobs.webhook.deliver",
  "status": "succeeded",
  "current_attempt": 1,
  "max_attempts": 3,
  "result_message": "webhook delivered",
  "affected_rows": 0,
  "created_at": "2026-07-14T12:00:00Z",
  "queued_at": "2026-07-14T12:00:00Z",
  "started_at": "2026-07-14T12:00:01Z",
  "completed_at": "2026-07-14T12:00:01Z",
  "updated_at": "2026-07-14T12:00:01Z"
}
```

### MVP 9 — Observability

* Prometheus metrics
* OpenTelemetry tracing
* Grafana dashboard
* Health и readiness endpoints

### Реализованное поведение MVP 9

Все HTTP-сервисы публикуют `/health`, `/ready` и `/metrics`. Liveness проверяет только работоспособность процесса, а readiness дополнительно проверяет обязательные зависимости. Gateway сообщает состояние optional-зависимостей как `degraded`, не останавливая трафик при отказе Redis, Kafka или Control Plane.

Prometheus доступен на `:9090` и собирает RED-метрики сервисов:

```text
edgeguard_http_requests_total
edgeguard_http_request_duration_seconds
edgeguard_http_response_size_bytes
edgeguard_http_requests_in_flight
```

Label `route` содержит route template, а не конкретный UUID или пользовательский path, что ограничивает cardinality временных рядов.

OpenTelemetry spans передаются по OTLP/gRPC через Collector в Tempo. Trace context распространяется через:

```text
HTTP:       traceparent header
Kafka:      record headers
RabbitMQ:   AMQP headers
```

Поэтому один trace может охватывать синхронную и асинхронную цепочку:

```text
Gateway → Demo Backend
       └→ Kafka → Analytics

Control Plane → RabbitMQ → Notification Worker → webhook
```

Сервисы возвращают `X-Trace-ID`, по которому trace можно получить из Tempo или открыть в Grafana.

Grafana доступна на `:3000`. Data sources Prometheus и Tempo, а также dashboard `EdgeGuard Overview` создаются автоматически из version-controlled provisioning-файлов. Dashboard содержит request rate, 5xx rate, P95 latency, in-flight requests, response size, memory, goroutines и состояние Prometheus targets.

Локальные адреса:

```text
Prometheus:     http://localhost:9090
Tempo API:      http://localhost:3200
OTel Collector: http://localhost:13133
Grafana:        http://localhost:3000
```

Учетные данные Grafana для локального Compose:

```text
admin / admin
```

### MVP 10 — Kubernetes Deployment

Реализован локальный production-like Kubernetes-стенд:

* Kustomize `base` для application workloads, stateful-инфраструктуры и observability;
* локальный overlay с development-секретами, одной репликой каждого приложения и образами `edgeguard-*:local`;
* Kind-кластер из control-plane и worker node;
* PostgreSQL, Redis, Kafka в KRaft-режиме и RabbitMQ с persistent volumes;
* Prometheus, OpenTelemetry Collector, Tempo и Grafana с provisioning;
* отдельные Job для Goose-миграций и создания Kafka topic;
* `startup`, `liveness` и `readiness` probes;
* resource requests/limits и hardened container security context;
* Ingress для Gateway через Cloud Provider KIND;
* сборка и загрузка локальных Docker-образов в Kind;
* повторяемый `make k8s-up`, включая пересоздание init Job и rollout приложений;
* Kubernetes smoke-тест и полный E2E-набор MVP4–MVP9.

Локальная конфигурация не является HA-развёртыванием: stateful-компоненты работают в одной реплике и предназначены для разработки и демонстрации проекта.

Подробное описание архитектуры стенда, порядка запуска, проверок и диагностики находится в [Kubernetes-гайде](docs/kubernetes.md).

## Технологический стек

Планируемые технологии:

* **Language:** Go
* **HTTP:** Echo, net/http
* **Reverse Proxy:** net/http/httputil
* **Database:** PostgreSQL
* **Migrations:** goose
* **Cache / Rate Limiting:** Redis
* **Event Streaming:** Kafka
* **Task Queue:** RabbitMQ
* **Authentication:** JWT, API keys
* **Observability:** Prometheus, OpenTelemetry, Grafana
* **Containers:** Docker, Docker Compose
* **Deployment:** Kubernetes
* **Testing:** testing, testify, testcontainers-go
* **CI/CD:** GitHub Actions — отдельный следующий этап

## Локальный запуск

### Запуск без Docker

В первом терминале:

```bash
make run-demo
```

Во втором терминале:

```bash
make run-gateway
```

Проверка через gateway:

```bash
curl http://localhost:8080/api/v1/orders
```

### Запуск через Docker Compose

Вся текущая система запускается одной командой:

```bash
make compose-up
```

После запуска gateway будет доступен по адресу:

```text
http://localhost:8080
```

Быстрая проверка:

```bash
make smoke-test
```

Или вручную:

```bash
curl http://localhost:8080/health
curl http://localhost:8082/health
curl http://localhost:8082/internal/v1/routes
curl http://localhost:9090/-/ready
curl http://localhost:3200/ready
curl http://localhost:3000/api/health
```

После создания маршрута через Control Plane Gateway подхватит его не позднее чем через `ROUTES_REFRESH_INTERVAL` и начнет проксировать соответствующие запросы без перезапуска.

### Запуск через Kubernetes (Kind)

Полная документация: [docs/kubernetes.md](docs/kubernetes.md).

Для локального Kubernetes-стенда необходимы:

```text
Docker
kind
kubectl
curl
jq
```

Перед запуском рекомендуется остановить Docker Compose, чтобы не расходовать ресурсы на два одинаковых стенда:

```bash
make compose-down
```

Проверить итоговый manifest без подключения к кластеру:

```bash
make k8s-render > /tmp/edgeguard-k8s.yaml
```

Создать Kind-кластер, собрать и загрузить локальные образы, применить манифесты, дождаться readiness и выполнить smoke-тест:

```bash
make k8s-up
```

Команда использует кластер `edgeguard` и контекст `kind-edgeguard`. Все вызовы `kubectl` в Makefile передают контекст явно, поэтому ранее выбранный неработающий context в `~/.kube/config` не влияет на стенд.

Порядок запуска:

```text
Kind cluster
    ↓
Cloud Provider KIND
    ↓
сборка edgeguard-*:local
    ↓
kind load docker-image
    ↓
server-side dry-run
    ↓
Kustomize apply
    ↓
PostgreSQL и Goose migrations
    ↓
Kafka и topic init Job
    ↓
Redis и RabbitMQ
    ↓
EdgeGuard applications
    ↓
Prometheus, OTel Collector, Tempo, Grafana
    ↓
Ingress и smoke-test
```

Повторный `make k8s-up` безопасно пересоздаёт migration/topic Job и выполняет rollout приложений, чтобы Pod получили заново собранные образы с тегом `:local`.

Проверить состояние:

```bash
make k8s-status
```

Получить адрес Gateway через Ingress:

```bash
INGRESS_IP="$(kubectl --context kind-edgeguard \
  -n edgeguard get ingress edgeguard \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"

curl "http://${INGRESS_IP}/health"
curl "http://${INGRESS_IP}/ready"
```

Повторно запустить проверки:

```bash
make k8s-smoke-test
make k8s-e2e-test
```

`k8s-e2e-test` запускает те же сценарии MVP4–MVP9, что и Compose-вариант, но использует Ingress, временные `kubectl port-forward` и `kubectl exec`.

Просмотреть логи:

```bash
make k8s-logs
```

Открыть Grafana локально:

```bash
kubectl --context kind-edgeguard \
  -n edgeguard port-forward service/grafana 3000:3000
```

После этого Grafana доступна на `http://localhost:3000`, локальные учётные данные — `admin / admin`.

Остановить и удалить весь Kind-кластер вместе с локальными PersistentVolume:

```bash
make k8s-down
```

Файл `deployments/k8s/overlays/local/secret-patch.yaml` содержит только development-значения. Для production секреты должны поступать из внешнего secret manager, CI/CD или environment-specific overlay.

### E2E-проверка динамической конфигурации

Для проверки полного сценария MVP4 требуется `jq`:

```bash
sudo dnf install jq
```

После запуска контейнеров выполните:

```bash
make e2e-test
```

Тест автоматически:

1. Проверяет health endpoints Gateway и Control Plane.
2. Проверяет, что новый уникальный path prefix пока неизвестен Gateway и возвращает `404`.
3. Создает уникальные project, service и route через Control Plane API.
4. Проверяет, что route появился в `GET /internal/v1/routes`.
5. Ожидает очередное polling-обновление Gateway.
6. Выполняет запрос к Demo Backend через новый динамический маршрут и проверяет ответ.

Проверяемый поток:

```text
E2E script
    |
    | POST project/service/route
    v
Control Plane ----> PostgreSQL
    |
    | GET /internal/v1/routes
    v
Gateway route polling
    |
    | atomic in-memory snapshot replacement
    v
GET /e2e-<run-id>/orders ----> Demo Backend /orders
```

Каждый запуск использует уникальные имена и path prefix, поэтому тест можно запускать повторно без сброса PostgreSQL. Созданные тестовые записи сохраняются в базе; для полной очистки используйте `make compose-reset`.

Переменные теста можно переопределить:

```bash
CONTROL_PLANE_URL=http://localhost:8082 \
GATEWAY_URL=http://localhost:8080 \
E2E_TIMEOUT_SECONDS=30 \
make e2e-test
```

Логи контейнеров:

```bash
make compose-logs
```

Остановка:

```bash
make compose-down
```

Ожидаемый результат: запрос будет обработан gateway и проксирован в demo backend. Demo-backend внутри Docker Compose не публикуется наружу и доступен gateway по внутреннему DNS-имени `demo-backend`.

### E2E-проверка API keys

Полная проверка MVP5:

```bash
make e2e-auth-test
```

Сценарий регистрирует пользователя, получает JWT, создает защищенный маршрут и API-ключ, а затем проверяет:

* отсутствие ключа возвращает `401`;
* невалидный ключ возвращает `401`;
* ключ другого проекта возвращает `403`;
* корректный ключ дает доступ к Demo Backend;
* `last_used_at` обновляется;
* отозванный ключ снова возвращает `401`.

Эта проверка также входит в общий `make e2e-test`.

### E2E-проверка Redis rate limiting

Полная проверка MVP6:

```bash
make e2e-rate-limit-test
```

Тест создает уникальные публичный и защищенный маршруты с коротким fixed window и проверяет:

* первые запросы проходят с `200`;
* заголовок `X-RateLimit-Remaining` уменьшается;
* запрос сверх лимита возвращает `429`;
* присутствуют `Retry-After` и `X-RateLimit-Reset`;
* после начала следующего окна запрос снова проходит;
* для защищенного маршрута два API-ключа имеют независимые счетчики.

Чтобы избежать нестабильности на границе fixed window, тест сначала читает `X-RateLimit-Reset`, дожидается нового окна и только затем выполняет точную последовательность проверок.

Параметры можно переопределить:

```bash
CONTROL_PLANE_URL=http://localhost:8082 \
AUTH_URL=http://localhost:8083 \
GATEWAY_URL=http://localhost:8080 \
RATE_LIMIT_WINDOW_SECONDS=5 \
E2E_TIMEOUT_SECONDS=40 \
make e2e-rate-limit-test
```

Команда `make e2e-test` выполняет проверки MVP4–MVP9 последовательно.

### E2E-проверка Kafka Analytics

```bash
make e2e-analytics-test
```

Тест создает уникальные project, service и route, выполняет запросы через Gateway и ожидает прохождения всей цепочки:

```text
Gateway → Kafka → Analytics Consumer → PostgreSQL → Analytics HTTP API
```

Ручной запрос сводки за последние 24 часа:

```bash
curl --get \
  "http://localhost:8084/api/v1/projects/${PROJECT_ID}/analytics/summary" \
  --data-urlencode "route_name=${ROUTE_NAME}" \
  --data-urlencode "method=GET" | jq
```

Почасовой временной ряд:

```bash
curl --get \
  "http://localhost:8084/api/v1/projects/${PROJECT_ID}/analytics/hourly" \
  --data-urlencode "from=2026-07-13T00:00:00Z" \
  --data-urlencode "to=2026-07-14T00:00:00Z" \
  --data-urlencode "route_name=${ROUTE_NAME}" \
  --data-urlencode "method=GET" \
  --data-urlencode "limit=100" | jq
```

### E2E-проверка RabbitMQ Background Jobs

```bash
make e2e-jobs-test
```

Тест проверяет полный жизненный цикл задач:

* успешный webhook и создание заказа в Demo Backend;
* чтение статуса через `GET /api/v1/jobs/:id`;
* permanent webhook failure и DLQ;
* временную ошибку, delayed retry и исчерпание `max_attempts`;
* генерацию JSON-отчета и наличие файла в `reports_data`;
* cleanup job;
* корректные `current_attempt`, `last_error`, `result_message`, `output_path` и `affected_rows`.

Проверяемый поток:

```text
Control Plane → background_jobs(publishing) → RabbitMQ confirm
              → background_jobs(queued)
              → Notification Worker
              → processing / retrying / succeeded / failed
              → GET /api/v1/jobs/:id
```

Timeout теста можно увеличить:

```bash
E2E_TIMEOUT_SECONDS=180 make e2e-jobs-test
```

### E2E-проверка Observability

```bash
make e2e-observability-test
```

Тест проверяет:

* готовность Prometheus, Tempo и Grafana;
* состояние всех EdgeGuard targets в Prometheus;
* provisioning data sources `prometheus` и `tempo`;
* provisioning dashboard `edgeguard-overview`;
* увеличение HTTP-счетчика после запроса через Gateway;
* trace `Gateway → Demo Backend → Kafka → Analytics`;
* trace `Control Plane → RabbitMQ → Notification Worker → Demo Backend`.

Trace ID возвращается в заголовке:

```http
X-Trace-ID: 24f413607fa16aeae6b186db87a4d924
```

Получить trace напрямую из Tempo:

```bash
curl -fsS "http://localhost:3200/api/traces/${TRACE_ID}" | jq
```

Открыть dashboard:

```text
http://localhost:3000/d/edgeguard-overview/edgeguard-overview
```

При медленном локальном окружении timeout можно увеличить:

```bash
E2E_TIMEOUT_SECONDS=180 make e2e-observability-test
```

## Архитектурные решения

Архитектурные решения документируются в директории `docs/adr`.

Планируемые ADR:

* `ADR-001` — Monorepo structure
* `ADR-002` — Data Plane and Control Plane separation
* `ADR-003` — Static gateway config for the first MVP
* `ADR-004` — PostgreSQL as primary storage
* `ADR-005` — Redis for rate limiting
* `ADR-006` — Kafka for gateway events
* `ADR-007` — RabbitMQ for background jobs

## Что демонстрирует проект

Проект предназначен для практики и демонстрации следующих backend-навыков:

* Реализация reverse proxy
* Работа с HTTP request/response
* Middleware design
* Configuration management
* Clean Architecture
* Проектирование схемы PostgreSQL
* Redis-based rate limiting
* Kafka event streaming
* RabbitMQ background processing
* JWT и API key authentication
* Observability
* Docker и Kubernetes deployment
* System design documentation

## Статус проекта

Текущий реализованный этап:

```text
MVP 10 — Kubernetes Deployment — завершён
Post-MVP10 — GitHub Actions CI — завершён
```

CI автоматически проверяет Go-код, Docker Compose smoke/E2E и Kubernetes-манифесты. Полный Kubernetes E2E доступен как ручной workflow run.

## License

MIT
