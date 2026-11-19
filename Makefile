# Convenience targets. Everything is also available as plain docker compose / go / npm commands.
.DEFAULT_GOAL := help
COMPOSE ?= docker compose

.PHONY: help up up-docker down restart logs token claude-login reset-auth shell ps test dev build

help: ## Show this help
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-14s %s\n", $$1, $$2}'

up: ## Build and start everything (creates .env from .env.example on first run)
	@test -f .env || (cp .env.example .env && echo "Created .env from .env.example")
	$(COMPOSE) up -d --build
	@echo ""
	@echo "Ayumi Link is starting. Open http://localhost:8081 (or your AYUMI_PORT)."
	@echo "First run? Get the setup token with:  make token"

up-docker: ## Build and start with host Docker access (detects the Docker socket of this machine)
	@test -f .env || (cp .env.example .env && echo "Created .env from .env.example")
	@sh scripts/detect-docker.sh
	$(COMPOSE) -f docker-compose.yml -f docker-compose.host-docker.yml up -d --build
	@echo "Host Docker is available in the Terminal tab:  docker ps"

down: ## Stop and remove the containers (data volumes are kept)
	$(COMPOSE) down

restart: ## Restart the app
	$(COMPOSE) restart ayumi-link

logs: ## Follow the logs
	$(COMPOSE) logs -f --tail=100 ayumi-link

token: ## Print the first-run setup token
	@$(COMPOSE) logs ayumi-link 2>&1 | grep -A1 "SETUP DIPERLUKAN" | tail -2 || true
	@echo "(no token shown = an account already exists)"

claude-login: ## Log the Claude CLI in inside the container
	$(COMPOSE) exec -it ayumi-link claude auth login

reset-auth: ## Delete the account + 2FA (then run first-time setup again)
	$(COMPOSE) exec -it ayumi-link ayumi-link reset-auth
	$(COMPOSE) restart ayumi-link

shell: ## Open a shell in the running container
	$(COMPOSE) exec -it --entrypoint sh ayumi-link

ps: ## Show container status and health
	$(COMPOSE) ps

test: ## Run backend tests and the frontend type check
	cd backend && go vet ./... && go test ./...
	cd frontend && npm ci && npx tsc -b

dev: ## How to run backend and frontend locally for development
	@echo "Terminal 1:  cd backend && go run ."
	@echo "Terminal 2:  cd frontend && npm install && npm run dev"
	@echo "Open http://localhost:5173"

build: ## Build the Docker image only
	$(COMPOSE) build
