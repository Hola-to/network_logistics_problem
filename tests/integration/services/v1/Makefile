.PHONY: up down test test-verbose clean

# Start all services
up:
	docker-compose up -d --build
	@echo "Waiting for services to be ready..."
	@sleep 10

# Stop all services
down:
	docker-compose down -v

# Run integration tests
test:
	INTEGRATION_TESTS=1 go test -v -timeout 5m ./...

# Run specific test
test-solver:
	INTEGRATION_TESTS=1 go test -v -run TestSolverService ./...

test-validation:
	INTEGRATION_TESTS=1 go test -v -run TestValidationService ./...

test-analytics:
	INTEGRATION_TESTS=1 go test -v -run TestAnalyticsService ./...

test-auth:
	INTEGRATION_TESTS=1 go test -v -run TestAuthService ./...

test-history:
	INTEGRATION_TESTS=1 go test -v -run TestHistoryService ./...

test-audit:
	INTEGRATION_TESTS=1 go test -v -run TestAuditService ./...

test-simulation:
	INTEGRATION_TESTS=1 go test -v -run TestSimulationService ./...

test-report:
	INTEGRATION_TESTS=1 go test -v -run TestReportService ./...

# Full cycle: up, test, down
full-test: up test down

# Clean up
clean:
	docker-compose down -v --rmi local
