# Native (make build / make dev) atau Docker (make up).
.DEFAULT_GOAL := help
COMPOSE ?= docker compose

.PHONY: help build dev service-install service-uninstall service-start service-stop service-restart service-status service-logs up up-docker redeploy down restart logs token claude-login reset-auth shell ps test docker-build

help: ## Show this help
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-18s %s\n", $$1, $$2}'

# ---- Native (tanpa Docker) ----

build: ## Native: build FE+BE and serve on http://localhost:8081
	@chmod +x scripts/native-build.sh
	@sh scripts/native-build.sh

dev: ## Native: API :8081 + Vite HMR :5173 (satu perintah)
	@chmod +x scripts/native-dev.sh
	@sh scripts/native-dev.sh

# Service (macOS: LaunchAgent, Linux: systemd): setiap restart = make build lagi (lihat skill ayumi-link-native-service)
service-install: ## Native: pasang service + start (macOS/Linux)
	@chmod +x scripts/native-service*.sh scripts/native-build.sh
	@sh scripts/native-service.sh install

service-uninstall: ## Native: hapus service
	@sh scripts/native-service.sh uninstall

service-start: ## Native: start service
	@sh scripts/native-service.sh start

service-stop: ## Native: stop service
	@sh scripts/native-service.sh stop

service-restart: ## Native: restart → make build jalan lagi (redeploy)
	@chmod +x scripts/native-service*.sh scripts/native-build.sh
	@sh scripts/native-service.sh restart

service-status: ## Native: status service
	@sh scripts/native-service.sh status

service-logs: ## Native: tail log service
	@sh scripts/native-service.sh logs

# ---- Docker ----

up: ## Docker: build and start (http://localhost:8081)
	@test -f .env || (cp .env.example .env && echo "Created .env from .env.example")
	$(COMPOSE) up -d --build
	@echo ""
	@echo "Ayumi Link is starting. Open http://localhost:8081 (or your AYUMI_PORT)."
	@echo "First run? Get the setup token with:  make token"

up-docker: ## Docker + host Docker socket (Terminal tab: docker ps)
	@test -f .env || (cp .env.example .env && echo "Created .env from .env.example")
	@sh scripts/detect-docker.sh
	$(COMPOSE) -f docker-compose.yml -f docker-compose.host-docker.yml up -d --build
	@echo "Host Docker is available in the Terminal tab:  docker ps"

docker-build: ## Docker: build image only
	$(COMPOSE) build

redeploy: ## Rebuild + restart safely (Mac terminal or inside the app container)
	@sh scripts/redeploy.sh

down: ## Stop and remove the containers (data volumes are kept)
	$(COMPOSE) down

restart: ## Restart the app container
	$(COMPOSE) restart ayumi-link

logs: ## Follow the container logs
	$(COMPOSE) logs -f --tail=100 ayumi-link

token: ## Print the first-run setup token (Docker logs)
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
