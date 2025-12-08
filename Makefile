# Makefile
#
# Logistics Network Optimization
#

# ============================================================
# Переменные
# ============================================================

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

DOCKER_REGISTRY ?= ghcr.io/your-org
DOCKER_PLATFORM ?= linux/amd64
K8S_NAMESPACE ?= logistics

GO := go
GOFLAGS := -v
LDFLAGS := -ldflags="-w -s -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

ROOT_DIR := $(shell pwd)
BIN_DIR := $(ROOT_DIR)/bin
COVERAGE_DIR := $(ROOT_DIR)/coverage

SERVICES := analytics-svc audit-svc auth-svc gateway-svc history-svc report-svc simulation-svc solver-svc validation-svc
DB_SERVICES := auth-svc history-svc audit-svc simulation-svc report-svc

# ============================================================
# Основные цели
# ============================================================

.PHONY: all
all: lint test build

.PHONY: help
help:
	@echo "Logistics Platform - Makefile"
	@echo "=============================="
	@echo ""
	@echo "Разработка:"
	@echo "  make dev              - Запустить dev окружение с hot-reload"
	@echo "  make dev-down         - Остановить dev окружение"
	@echo "  make infra            - Запустить только инфраструктуру"
	@echo ""
	@echo "Сборка:"
	@echo "  make build            - Собрать все сервисы"
	@echo "  make build-<service>  - Собрать конкретный сервис"
	@echo "  make build-linux      - Собрать для Linux"
	@echo ""
	@echo "Тесты:"
	@echo "  make test             - Unit тесты"
	@echo "  make test-coverage    - Тесты с покрытием"
	@echo "  make test-integration - Интеграционные тесты"
	@echo "  make test-benchmark   - Бенчмарки"
	@echo ""
	@echo "Качество кода:"
	@echo "  make lint             - Запустить линтер"
	@echo "  make fmt              - Форматировать код"
	@echo "  make tidy             - Обновить go.mod"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build     - Собрать все образы"
	@echo "  make docker-push      - Запушить все образы"
	@echo "  make compose-up       - Запустить docker-compose"
	@echo "  make compose-down     - Остановить docker-compose"
	@echo ""
	@echo "Kubernetes:"
	@echo "  make k8s-apply-dev    - Применить dev конфигурацию"
	@echo "  make k8s-apply-staging - Применить staging"
	@echo "  make k8s-status       - Статус подов"
	@echo ""
	@echo "Helm:"
	@echo "  make helm-install-staging - Установить в staging"
	@echo "  make helm-install-prod    - Установить в production"
	@echo ""
	@echo "Деплой:"
	@echo "  make deploy-staging   - Полный деплой в staging"
	@echo "  make deploy-prod      - Полный деплой в production"
	@echo ""
	@echo "Утилиты:"
	@echo "  make proto            - Сгенерировать proto"
	@echo "  make tools            - Установить инструменты"
	@echo "  make clean            - Очистить артефакты"
	@echo "  make info             - Информация о проекте"

# ============================================================
# Разработка
# ============================================================

.PHONY: dev
dev:
	@echo "🚀 Запуск dev окружения..."
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml up --build

.PHONY: dev-down
dev-down:
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml down -v

.PHONY: dev-logs
dev-logs:
	docker-compose -f docker-compose.yml -f docker-compose.dev.yml logs -f

.PHONY: infra
infra:
	@echo "🏗 Запуск инфраструктуры..."
	docker-compose up -d postgres redis jaeger prometheus grafana
	@echo "✅ Инфраструктура запущена"
	@echo "   PostgreSQL: localhost:5432"
	@echo "   Redis:      localhost:6379"
	@echo "   Jaeger UI:  http://localhost:16686"
	@echo "   Prometheus: http://localhost:9090"
	@echo "   Grafana:    http://localhost:3000"

.PHONY: infra-down
infra-down:
	docker-compose down -v

# ============================================================
# Сборка
# ============================================================

.PHONY: build
build: $(addprefix build-,$(SERVICES))
	@echo "✅ Все сервисы собраны"

.PHONY: build-%
build-%:
	@echo "📦 Сборка $*..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/$* ./services/$*/cmd/main.go

.PHONY: build-linux
build-linux:
	@mkdir -p $(BIN_DIR)
	@for svc in $(SERVICES); do \
		echo "Building $$svc for linux/amd64..."; \
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BIN_DIR)/$$svc ./services/$$svc/cmd/main.go; \
	done

# ============================================================
# Тестирование
# ============================================================

.PHONY: test
test:
	@echo "🧪 Запуск unit тестов..."
	$(GO) test -race -short ./pkg/... ./services/...

.PHONY: test-verbose
test-verbose:
	$(GO) test -race -short -v ./pkg/... ./services/...

.PHONY: test-coverage
test-coverage:
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./pkg/... ./services/...
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@$(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -1

.PHONY: test-integration
test-integration:
	$(GO) test -race -tags=integration -v ./tests/integration/...

.PHONY: test-benchmark
test-benchmark:
	$(GO) test -bench=. -benchmem -run=^$$ ./tests/benchmark/...

.PHONY: test-all
test-all: test test-integration test-benchmark

# ============================================================
# Линтинг и форматирование
# ============================================================

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: lint-fix
lint-fix:
	golangci-lint run --fix ./...

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: tidy
tidy:
	$(GO) mod tidy
	$(GO) mod verify

# ============================================================
# Генерация кода
# ============================================================

.PHONY: proto
proto:
	buf generate

.PHONY: proto-lint
proto-lint:
	buf lint

.PHONY: generate
generate: proto
	$(GO) generate ./...

# ============================================================
# Docker
# ============================================================

.PHONY: docker-build
docker-build: $(addprefix docker-build-,$(SERVICES))
	@echo "✅ Все Docker образы собраны"

.PHONY: docker-build-%
docker-build-%:
	@echo "🐳 Сборка Docker образа для $*..."
	docker build --platform $(DOCKER_PLATFORM) --build-arg VERSION=$(VERSION) -t $(DOCKER_REGISTRY)/$*:$(VERSION) -t $(DOCKER_REGISTRY)/$*:latest -f services/$*/Dockerfile .

.PHONY: docker-push
docker-push: $(addprefix docker-push-,$(SERVICES))

.PHONY: docker-push-%
docker-push-%: docker-build-%
	docker push $(DOCKER_REGISTRY)/$*:$(VERSION)
	docker push $(DOCKER_REGISTRY)/$*:latest

.PHONY: docker-clean
docker-clean:
	@for svc in $(SERVICES); do \
		docker rmi $(DOCKER_REGISTRY)/$$svc:$(VERSION) 2>/dev/null || true; \
		docker rmi $(DOCKER_REGISTRY)/$$svc:latest 2>/dev/null || true; \
	done
	docker image prune -f

# ============================================================
# Docker Compose
# ============================================================

.PHONY: compose-up
compose-up:
	docker-compose up -d --build

.PHONY: compose-down
compose-down:
	docker-compose down -v

.PHONY: compose-logs
compose-logs:
	docker-compose logs -f

.PHONY: compose-ps
compose-ps:
	docker-compose ps

.PHONY: compose-restart
compose-restart:
	docker-compose restart

# ============================================================
# Kubernetes
# ============================================================

.PHONY: k8s-apply-dev
k8s-apply-dev:
	kubectl apply -k deploy/k8s/overlays/development

.PHONY: k8s-apply-staging
k8s-apply-staging:
	kubectl apply -k deploy/k8s/overlays/staging

.PHONY: k8s-apply-prod
k8s-apply-prod:
	@read -p "Применить в PRODUCTION? [y/N] " confirm && [ "$$confirm" = "y" ]
	kubectl apply -k deploy/k8s/overlays/production

.PHONY: k8s-delete-dev
k8s-delete-dev:
	kubectl delete -k deploy/k8s/overlays/development

.PHONY: k8s-status
k8s-status:
	kubectl -n $(K8S_NAMESPACE) get pods -o wide

.PHONY: k8s-logs
k8s-logs:
	kubectl -n $(K8S_NAMESPACE) logs -l app.kubernetes.io/part-of=logistics -f --max-log-requests=20

.PHONY: k8s-port-forward
k8s-port-forward:
	kubectl -n $(K8S_NAMESPACE) port-forward svc/gateway-svc 8080:80

# ============================================================
# Helm
# ============================================================

.PHONY: helm-deps
helm-deps:
	helm dependency update deploy/helm/logistics-platform

.PHONY: helm-lint
helm-lint:
	helm lint deploy/helm/logistics-platform

.PHONY: helm-template
helm-template:
	helm template logistics deploy/helm/logistics-platform

.PHONY: helm-install-staging
helm-install-staging: helm-deps
	helm upgrade --install logistics deploy/helm/logistics-platform --namespace logistics-staging --create-namespace -f deploy/helm/logistics-platform/values.yaml -f deploy/helm/logistics-platform/values-staging.yaml --set image.tag=$(VERSION) --wait --timeout 10m

.PHONY: helm-install-prod
helm-install-prod: helm-deps
	@read -p "Установить в PRODUCTION? [y/N] " confirm && [ "$$confirm" = "y" ]
	helm upgrade --install logistics deploy/helm/logistics-platform --namespace logistics --create-namespace -f deploy/helm/logistics-platform/values.yaml -f deploy/helm/logistics-platform/values-production.yaml --set image.tag=$(VERSION) --wait --timeout 10m

.PHONY: helm-uninstall
helm-uninstall:
	helm uninstall logistics --namespace $(K8S_NAMESPACE)

.PHONY: helm-rollback
helm-rollback:
	helm rollback logistics --namespace $(K8S_NAMESPACE)

# ============================================================
# Деплой
# ============================================================

.PHONY: deploy-staging
deploy-staging: docker-build docker-push helm-install-staging
	@echo "✅ Деплой в staging завершен"

.PHONY: deploy-prod
deploy-prod:
	@read -p "Деплой в PRODUCTION? [y/N] " confirm && [ "$$confirm" = "y" ]
	$(MAKE) docker-build
	$(MAKE) docker-push
	$(MAKE) helm-install-prod

# ============================================================
# База данных
# ============================================================

.PHONY: db-shell
db-shell:
	docker-compose exec postgres psql -U logistics -d logistics

.PHONY: redis-shell
redis-shell:
	docker-compose exec redis redis-cli

.PHONY: db-reset
db-reset:
	@read -p "Сбросить ВСЕ данные? [y/N] " confirm && [ "$$confirm" = "y" ]
	docker-compose exec postgres psql -U logistics -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

# ============================================================
# Утилиты
# ============================================================

.PHONY: health
health:
	@echo "🏥 Проверка health endpoints..."
	@curl -s -o /dev/null -w "Gateway:    %{http_code}\n" http://localhost:8080/health || echo "Gateway:    DOWN"

.PHONY: tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/air-verse/air@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
	@echo "✅ Инструменты установлены"

.PHONY: vuln
vuln:
	@which govulncheck > /dev/null || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

# ============================================================
# Очистка
# ============================================================

.PHONY: clean
clean:
	rm -rf $(BIN_DIR) $(COVERAGE_DIR) coverage.out
	@for svc in $(SERVICES); do rm -rf services/$$svc/bin services/$$svc/tmp; done

.PHONY: clean-docker
clean-docker: docker-clean compose-down
	docker system prune -f

.PHONY: clean-all
clean-all: clean clean-docker

# ============================================================
# Информация
# ============================================================

.PHONY: info
info:
	@echo "Logistics Platform"
	@echo "=================="
	@echo "Версия:   $(VERSION)"
	@echo "Коммит:   $(GIT_COMMIT)"
	@echo "Ветка:    $(GIT_BRANCH)"
	@echo "Registry: $(DOCKER_REGISTRY)"
	@echo ""
	@echo "Сервисы: $(SERVICES)"

.PHONY: version
version:
	@echo $(VERSION)

# CI/CD
.PHONY: ci-lint
ci-lint:
	golangci-lint run --out-format=github-actions ./...

.PHONY: ci-test
ci-test:
	$(GO) test -race -coverprofile=coverage.out -covermode=atomic ./pkg/... ./services/...

.PHONY: ci-build
ci-build:
	@for svc in $(SERVICES); do CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/$$svc ./services/$$svc/cmd/main.go; done
