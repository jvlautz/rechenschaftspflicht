FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY src/go.mod src/go.sum ./
RUN go mod download
RUN apk add --no-cache build-base

COPY src ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o main .

FROM alpine
RUN apk add --no-cache ca-certificates

WORKDIR /app
RUN addgroup -g 1000 -S app && adduser -u 1000 -G app -S app
USER app
COPY --from=builder /app/main .

EXPOSE 8080
CMD ["./main"]
