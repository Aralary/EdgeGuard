FROM golang:1.26.4-alpine AS builder

ARG GOOSE_VERSION=v3.27.2

RUN CGO_ENABLED=0 go install \
    -tags='no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb' \
    github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION}

FROM alpine:3.22

RUN adduser -D -g '' appuser

COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY migrations /migrations

USER appuser

ENV GOOSE_DRIVER=postgres \
    GOOSE_MIGRATION_DIR=/migrations

ENTRYPOINT ["goose"]
CMD ["up"]
