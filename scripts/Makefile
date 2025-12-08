.PHONY: deps deps-all deps-tree deps-docker

## deps: Analyze dependencies for all services
deps:
	@go run scripts/deps/main.go

## deps-svc: Analyze dependencies for specific service (usage: make deps-svc SVC=simulation-svc)
deps-svc:
	@go run scripts/deps/main.go -path "./services/$(SVC)/..."

## deps-tree: Show dependency tree for all services
deps-tree:
	@go run scripts/deps/main.go -tree -depth 3

## deps-docker: Generate Dockerfile COPY commands
deps-docker:
	@go run scripts/deps/main.go -docker

## deps-help: Show deps analyzer help
deps-help:
	@go run scripts/deps/main.go -help
