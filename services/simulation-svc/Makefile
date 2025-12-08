.PHONY: build run test docker-build docker-push docker-run clean

# Переменные
SERVICE_NAME := simulation-svc
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
DOCKER_REGISTRY ?= your-registry.com
IMAGE_NAME := $(DOCKER_REGISTRY)/$(SERVICE_NAME)

# Сборка локально
build:
	@echo "Building $(SERVICE_NAME)..."
	go build -ldflags="-w -s -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)" \
		-o bin/$(SERVICE_NAME) ./cmd/main.go

# Запуск локально
run: build
	./bin/$(SERVICE_NAME)

# Тесты
test:
	go test -v -race -coverprofile=coverage.out ./...

# Сборка Docker образа
docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(IMAGE_NAME):$(VERSION) \
		-t $(IMAGE_NAME):latest \
		-f Dockerfile \
		../..

# Push Docker образа
docker-push: docker-build
	docker push $(IMAGE_NAME):$(VERSION)
	docker push $(IMAGE_NAME):latest

# Запуск в Docker
docker-run:
	docker run --rm -it \
		-p 50058:50058 \
		-p 8088:8088 \
		-e LOG_LEVEL=debug \
		$(IMAGE_NAME):latest

# Docker Compose
compose-up:
	docker-compose up -d

compose-down:
	docker-compose down -v

compose-logs:
	docker-compose logs -f simulation-svc

# Очистка
clean:
	rm -rf bin/ tmp/ coverage.out
	docker-compose down -v --rmi local 2>/dev/null || true
