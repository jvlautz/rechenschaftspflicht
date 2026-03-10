FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY src/go.mod src/go.sum ./
RUN go mod download
RUN apk add --no-cache build-base

COPY src ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o main .

FROM litestream/litestream:0.5 AS litestream

FROM alpine AS prod
RUN apk add --no-cache ca-certificates
COPY --from=litestream /usr/local/bin/litestream /usr/local/bin/litestream
COPY --from=litestream /usr/local/lib/litestream-vfs.so /usr/local/lib/litestream-vfs.so

WORKDIR /app
RUN addgroup -g 1000 -S app && adduser -u 1000 -G app -S app
USER app
COPY --from=builder /app/main .
COPY src/etc/litestream.yml /etc/litestream.yml
COPY src/etc/entrypoint.sh /app/etc/entrypoint.sh

EXPOSE 8080
ENTRYPOINT ["/app/etc/entrypoint.sh"]
