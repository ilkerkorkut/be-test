GO_SVC_DIR := services/go-svc
NODE_SVC_DIR := services/node-svc
GO_SVC_IMAGE ?= be-test-go-svc
GO_WORKER_IMAGE ?= be-test-go-worker
NODE_SVC_IMAGE ?= be-test-node-svc
GO_SVC_PORT ?= 8081
NODE_SVC_PORT ?= 8082
COMPOSE ?= docker compose

.DEFAULT_GOAL := help

.PHONY: help go-run go-worker-run go-test go-test-race go-vet go-build go-worker-build go-clean go-docker-build go-worker-docker-build go-docker-run go-prod node-install node-run node-test node-build node-docker-build node-docker-run node-prod compose-build compose-up compose-down compose-clean compose-logs compose-ps

help: ## List available development and production commands.
	@awk 'BEGIN {FS = ":.*##"}; /^[a-zA-Z0-9_-]+:.*##/ {printf "%-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

go-run: ## Run the Go API locally, loading services/go-svc/.env when available.
	@if [ -f "$(GO_SVC_DIR)/.env" ]; then set -a; . "$(GO_SVC_DIR)/.env"; set +a; fi; cd "$(GO_SVC_DIR)" && go run ./cmd/api

go-test: ## Run all Go service tests.
	@cd "$(GO_SVC_DIR)" && go test ./...

go-test-race: ## Run Go service tests with the race detector.
	@cd "$(GO_SVC_DIR)" && go test -race ./...

go-vet: ## Run Go's static analysis checks.
	@cd "$(GO_SVC_DIR)" && go vet ./...

go-build: ## Build the Go API binary for local development.
	@mkdir -p build
	@cd "$(GO_SVC_DIR)" && go build -o ../../build/go-svc ./cmd/api

go-clean: ## Remove locally built Go binaries.
	@rm -f build/go-svc build/go-svc-worker

go-docker-build: ## Build the production Go service image.
	@docker build --target api --file "$(GO_SVC_DIR)/Dockerfile" --tag "$(GO_SVC_IMAGE)" .

go-worker-docker-build: ## Build the production Go worker image.
	@docker build --target worker --file "$(GO_SVC_DIR)/Dockerfile" --tag "$(GO_WORKER_IMAGE)" .

go-docker-run: ## Run the production Go service image on the configured local port.
	@docker run --rm --env-file .env -p "$(GO_SVC_PORT):8081" "$(GO_SVC_IMAGE)"

go-prod: go-docker-build go-docker-run ## Build and run the Go production image.

go-worker-run: ## Run the Go settlement worker locally.
	@if [ -f "$(GO_SVC_DIR)/.env" ]; then set -a; . "$(GO_SVC_DIR)/.env"; set +a; fi; cd "$(GO_SVC_DIR)" && go run ./cmd/worker

go-worker-build: ## Build the Go worker binary for local development.
	@mkdir -p build
	@cd "$(GO_SVC_DIR)" && go build -o ../../build/go-svc-worker ./cmd/worker

node-install: ## Install Node service dependencies from the lockfile.
	@cd "$(NODE_SVC_DIR)" && npm ci

node-run: ## Run the Node service in watch mode.
	@cd "$(NODE_SVC_DIR)" && npm run dev

node-test: ## Run Node service contract tests.
	@cd "$(NODE_SVC_DIR)" && npm test

node-build: ## Compile the Node service for production.
	@cd "$(NODE_SVC_DIR)" && npm run build

node-docker-build: ## Build the production Node service image.
	@docker build --file "$(NODE_SVC_DIR)/Dockerfile" --tag "$(NODE_SVC_IMAGE)" "$(NODE_SVC_DIR)"

node-docker-run: ## Run the Node production image with the seeded data mounted read-only.
	@docker run --rm --env-file .env -e PORT=8082 -e DATA_DIRECTORY=/app/data -v "$(CURDIR)/data:/app/data:ro" -p "$(NODE_SVC_PORT):8082" "$(NODE_SVC_IMAGE)"

node-prod: node-docker-build node-docker-run ## Build and run the Node production image.

compose-build: ## Build all Compose services.
	@$(COMPOSE) build

compose-up: ## Build and start the UI, APIs, worker, Redis, and Kafka in the background.
	@$(COMPOSE) up -d --build

compose-down: ## Stop and remove all Compose services and dependencies, preserving data files.
	@$(COMPOSE) down --remove-orphans

compose-clean: ## Stop the full Compose stack and remove Docker volumes.
	@$(COMPOSE) down --volumes --remove-orphans

compose-logs: ## Follow logs for all Compose services.
	@$(COMPOSE) logs --follow

compose-ps: ## Show the status of all Compose services.
	@$(COMPOSE) ps
