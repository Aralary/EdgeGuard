FROM golang:1.26.4-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/gateway ./cmd/gateway

FROM alpine:3.22

RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /bin/gateway /app/gateway
COPY configs/gateway.docker.yaml /app/configs/gateway.docker.yaml

ENV GATEWAY_CONFIG_PATH=/app/configs/gateway.docker.yaml

EXPOSE 8080

USER appuser

ENTRYPOINT ["/app/gateway"]
