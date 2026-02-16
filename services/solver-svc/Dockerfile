# services/solver-svc/Dockerfile

FROM golang:1.26-alpine AS builder

RUN apk add --no-cache ca-certificates git tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download -x && go mod verify

# Generated proto files
COPY gen/go/logistics/audit/ ./gen/go/logistics/audit/
COPY gen/go/logistics/common/ ./gen/go/logistics/common/
COPY gen/go/logistics/optimization/ ./gen/go/logistics/optimization/
COPY gen/openapi/ ./gen/openapi/

# Shared packages
COPY pkg/apperror/ ./pkg/apperror/
COPY pkg/audit/ ./pkg/audit/
COPY pkg/cache/ ./pkg/cache/
COPY pkg/client/ ./pkg/client/
COPY pkg/config/ ./pkg/config/
COPY pkg/domain/ ./pkg/domain/
COPY pkg/interceptors/ ./pkg/interceptors/
COPY pkg/logger/ ./pkg/logger/
COPY pkg/metrics/ ./pkg/metrics/
COPY pkg/ratelimit/ ./pkg/ratelimit/
COPY pkg/server/ ./pkg/server/
COPY pkg/swagger/ ./pkg/swagger/
COPY pkg/telemetry/ ./pkg/telemetry/

# Services
COPY services/solver-svc/ ./services/solver-svc/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o /app/bin/solver-svc \
    ./services/solver-svc/cmd/main.go

FROM scratch

LABEL org.opencontainers.image.title="Solver Service" \
    org.opencontainers.image.description="Network flow optimization solver for logistics platform"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/bin/solver-svc /solver-svc

EXPOSE 50054
EXPOSE 8054

ENV TZ=UTC \
    GRPC_PORT=50054 \
    METRICS_PORT=8054 \
    LOG_LEVEL=info \
    LOG_FORMAT=json

ENTRYPOINT ["/solver-svc"]
