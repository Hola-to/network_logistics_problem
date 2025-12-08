# ============================================================
# Stage 1: Build
# ============================================================
FROM golang:1.25.4-alpine AS builder

# Установка необходимых пакетов для сборки
RUN apk add --no-cache \
    ca-certificates \
    git \
    tzdata

# Рабочая директория
WORKDIR /app

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./

# Скачиваем зависимости
RUN go mod download && go mod verify

# Минимальный набор файлов
# Сгенерированные proto файлы
COPY gen/go/logistics/audit/ ./gen/go/logistics/audit/
COPY gen/go/logistics/common/ ./gen/go/logistics/common/
COPY gen/go/logistics/optimization/ ./gen/go/logistics/optimization/
COPY gen/go/logistics/simulation/ ./gen/go/logistics/simulation/
COPY gen/openapi/ ./gen/openapi/

# Общие пакеты (pkg)
COPY pkg/apperror/ ./pkg/apperror/
COPY pkg/audit/ ./pkg/audit/
COPY pkg/client/ ./pkg/client/
COPY pkg/config/ ./pkg/config/
COPY pkg/database/ ./pkg/database/
COPY pkg/interceptors/ ./pkg/interceptors/
COPY pkg/logger/ ./pkg/logger/
COPY pkg/metrics/ ./pkg/metrics/
COPY pkg/ratelimit/ ./pkg/ratelimit/
COPY pkg/server/ ./pkg/server/
COPY pkg/swagger/ ./pkg/swagger/
COPY pkg/telemetry/ ./pkg/telemetry/

# Сервисы
COPY services/simulation-svc/ ./services/simulation-svc/

# Migrations
COPY migrations/ ./migrations/

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev} -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /app/bin/simulation-svc \
    ./services/simulation-svc/cmd/main.go

# ============================================================
# Stage 2: Runtime (minimal)
# ============================================================
FROM scratch

# Метаданные образа
LABEL org.opencontainers.image.title="Simulation Service" \
    org.opencontainers.image.description="Simulation service for logistics network optimization" \
    org.opencontainers.image.vendor="Logistics Platform" \
    org.opencontainers.image.source="https://github.com/your-org/logistics"

# Копируем сертификаты для HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Копируем информацию о временных зонах
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Копируем бинарник
COPY --from=builder /app/bin/simulation-svc /simulation-svc

# Копируем миграции если нужны
COPY --from=builder /app/migrations /migrations

# Порт gRPC
EXPOSE 50058

# Порт метрик/health
EXPOSE 8088

# Переменные окружения по умолчанию
ENV TZ=UTC \
    GRPC_PORT=50058 \
    METRICS_PORT=8088 \
    LOG_LEVEL=info \
    LOG_FORMAT=json

# Точка входа
ENTRYPOINT ["/simulation-svc"]
