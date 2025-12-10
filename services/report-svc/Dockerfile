# services/report-svc/Dockerfile

FROM golang:1.25-alpine AS builder

RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download -x && go mod verify

# Generated proto files
COPY gen/go/logistics/analytics/ ./gen/go/logistics/analytics/
COPY gen/go/logistics/audit/ ./gen/go/logistics/audit/
COPY gen/go/logistics/common/ ./gen/go/logistics/common/
COPY gen/go/logistics/optimization/ ./gen/go/logistics/optimization/
COPY gen/go/logistics/report/ ./gen/go/logistics/report/
COPY gen/go/logistics/simulation/ ./gen/go/logistics/simulation/
COPY gen/openapi/ ./gen/openapi/

# Shared packages
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

# Services
COPY services/report-svc/ ./services/report-svc/

# Migrations
COPY migrations/ ./migrations/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o /app/bin/report-svc \
    ./services/report-svc/cmd/main.go

FROM scratch

LABEL org.opencontainers.image.title="Report Service" \
    org.opencontainers.image.description="Report generation service for logistics platform"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/bin/report-svc /report-svc
COPY --from=builder /app/migrations /migrations

EXPOSE 50059
EXPOSE 8059

ENV TZ=UTC \
    GRPC_PORT=50059 \
    METRICS_PORT=8059 \
    LOG_LEVEL=info \
    LOG_FORMAT=json

ENTRYPOINT ["/report-svc"]
