# services/analytics-svc/Makefile

.PHONY: build run test docker-build docker-push docker-run clean

SERVICE_NAME := analytics-svc
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DOCKER_REGISTRY ?= your-registry.com
IMAGE_NAME := $(DOCKER_REGISTRY)/$(SERVICE_NAME)

build:
	@echo "Building $(SERVICE_NAME)..."
	go build -ldflags="-w -s -X main.Version=$(VERSION)" \
		-o bin/$(SERVICE_NAME) ./cmd/main.go

run: build
	./bin/$(SERVICE_NAME)

test:
	go test -v -race -coverprofile=coverage.out ./...

docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		-t $(IMAGE_NAME):$(VERSION) \
		-t $(IMAGE_NAME):latest \
		-f Dockerfile \
		../..

docker-push: docker-build
	docker push $(IMAGE_NAME):$(VERSION)
	docker push $(IMAGE_NAME):latest

docker-run:
	docker run --rm -it \
		-p 50053:50053 \
		-p 8053:8053 \
		-e LOG_LEVEL=debug \
		$(IMAGE_NAME):latest

compose-up:
	docker-compose up -d

compose-down:
	docker-compose down -v

clean:
	rm -rf bin/ tmp/ coverage.out
