# services/auth-svc/Makefile

.PHONY: build run test docker-build docker-push docker-run clean

SERVICE_NAME := auth-svc
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DOCKER_REGISTRY ?= your-registry.com
IMAGE_NAME := $(DOCKER_REGISTRY)/$(SERVICE_NAME)

build:
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
		-p 50055:50055 \
		-p 8055:8055 \
		-e LOG_LEVEL=debug \
		-e JWT_SECRET=dev-secret \
		$(IMAGE_NAME):latest

compose-up:
	docker-compose up -d

compose-down:
	docker-compose down -v

clean:
	rm -rf bin/ tmp/ coverage.out
