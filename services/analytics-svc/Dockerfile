# services/analytics-svc/Dockerfile

# ============================================================
# Stage 1: Build
# ============================================================
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache \
    ca-certificates \
    git \
    tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Proto файлы
COPY gen/go/logistics/analytics/ ./gen/go/logistics/analytics/
COPY gen/go/logistics/common/ ./gen/go/logistics/common/
COPY gen/openapi/ ./gen/openapi/

# Общие пакеты
COPY pkg/apperror/ ./pkg/apperror/
COPY pkg/config/ ./pkg/config/
COPY pkg/interceptors/ ./pkg/interceptors/
COPY pkg/logger/ ./pkg/logger/
COPY pkg/metrics/ ./pkg/metrics/
COPY pkg/ratelimit/ ./pkg/ratelimit/
COPY pkg/server/ ./pkg/server/
COPY pkg/swagger/ ./pkg/swagger/
COPY pkg/telemetry/ ./pkg/telemetry/
COPY pkg/audit/ ./pkg/audit/

# Сервис
COPY services/analytics-svc/ ./services/analytics-svc/

# Сборка
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o /app/bin/analytics-svc \
    ./services/analytics-svc/cmd/main.go

# ============================================================
# Stage 2: Runtime
# ============================================================
FROM scratch

LABEL org.opencontainers.image.title="Analytics Service" \
    org.opencontainers.image.description="Analytics service for logistics network optimization"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/bin/analytics-svc /analytics-svc

EXPOSE 50053
EXPOSE 8053

ENV TZ=UTC \
    GRPC_PORT=50053 \
    METRICS_PORT=8053 \
    LOG_LEVEL=info \
    LOG_FORMAT=json

ENTRYPOINT ["/analytics-svc"]
