FROM golang:1.26.4-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/demo-backend ./cmd/demo-backend

FROM alpine:3.22

RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /bin/demo-backend /app/demo-backend

EXPOSE 8081

USER appuser

ENTRYPOINT ["/app/demo-backend"]
