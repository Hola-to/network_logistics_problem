# services/gateway-svc/Dockerfile

FROM golang:1.25-alpine AS builder

RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Generated proto files
COPY gen/ ./gen/

# Shared packages
COPY pkg/config/ ./pkg/config/
COPY pkg/logger/ ./pkg/logger/
COPY pkg/metrics/ ./pkg/metrics/
COPY pkg/ratelimit/ ./pkg/ratelimit/

# Services
COPY services/gateway-svc/ ./services/gateway-svc/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o /app/bin/gateway-svc \
    ./services/gateway-svc/cmd/main.go

FROM scratch

LABEL org.opencontainers.image.title="Gateway Service" \
    org.opencontainers.image.description="API Gateway for logistics platform (ConnectRPC)"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/bin/gateway-svc /gateway-svc

EXPOSE 8080

ENV TZ=UTC \
    HTTP_PORT=8080 \
    LOG_LEVEL=info \
    LOG_FORMAT=json

ENTRYPOINT ["/gateway-svc"]
