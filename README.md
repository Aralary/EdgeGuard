# EdgeGuard

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
```

## Структура репозитория

```text
edgeguard
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
│   │   └── demo-backend.Dockerfile
│   └── k8s
│
├── migrations
├── docs
│   ├── architecture.md
│   ├── system-design.md
│   └── adr
│
├── api
│   └── openapi.yaml
│
├── Makefile
├── go.mod
└── README.md
```

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
* RBAC

### MVP 6 — Redis Rate Limiting

* Интеграция с Redis
* Fixed window rate limiting
* Rate limit by IP
* Rate limit by API key
* Rate limit by route

### MVP 7 — Kafka Analytics

* Gateway access events
* Kafka producer
* Analytics worker
* Агрегация статистики запросов

### MVP 8 — RabbitMQ Background Jobs

* Notification worker
* Webhook jobs
* Report generation jobs
* Cleanup jobs
* Retry и dead-letter queue

### MVP 9 — Observability

* Prometheus metrics
* OpenTelemetry tracing
* Grafana dashboard
* Health и readiness endpoints

### MVP 10 — Kubernetes Deployment

* Kubernetes manifests
* ConfigMap
* Secret
* Deployment
* Service
* Ingress
* Liveness probe
* Readiness probe
* Resource limits

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
* **CI/CD:** GitHub Actions

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
curl http://localhost:8080/api/v1/orders
curl http://localhost:8080/api/v1/orders/ord_1
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

Проект находится на начальном этапе разработки.

Текущий фокус:

```text
MVP 2 — Docker Compose local run
```

## License

MIT
