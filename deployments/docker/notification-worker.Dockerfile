FROM golang:1.26.4-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/notification-worker ./cmd/notification-worker

FROM alpine:3.22

RUN addgroup -g 10001 appgroup \
    && adduser -D -u 10001 -G appgroup appuser \
    && mkdir -p /data/reports \
    && chown -R appuser:appuser /data

WORKDIR /app

COPY --from=builder /bin/notification-worker /app/notification-worker

USER 10001:10001

ENTRYPOINT ["/app/notification-worker"]
