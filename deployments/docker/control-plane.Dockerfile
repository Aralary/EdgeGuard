FROM golang:1.26.4-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/control-plane ./cmd/control-plane

FROM alpine:3.22

RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /bin/control-plane /app/control-plane

EXPOSE 8082

USER appuser

ENTRYPOINT ["/app/control-plane"]
