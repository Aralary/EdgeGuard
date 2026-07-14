FROM golang:1.26.4-alpine AS builder

ARG GOOSE_VERSION=v3.27.2

RUN CGO_ENABLED=0 go install \
    -tags='no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb' \
    github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION}

FROM alpine:3.22

RUN addgroup -g 10001 appgroup \
    && adduser -D -u 10001 -G appgroup appuser

COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY migrations /migrations

USER 10001:10001

ENV GOOSE_DRIVER=postgres \
    GOOSE_MIGRATION_DIR=/migrations

ENTRYPOINT ["goose"]
CMD ["up"]
