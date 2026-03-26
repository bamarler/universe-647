# Universe 647 Makefile
# Requires: docker, sops, age, restic, tailscale, bun

SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c

# Colors
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
CYAN := \033[0;36m
BLUE := \033[0;34m
NC := \033[0m

# Stack compose files
CORE     := stacks/core/compose.yaml
DATA     := stacks/data/compose.yaml
SOPHON   := stacks/sophon/compose.yaml
STORAGE  := stacks/storage/compose.yaml
VOICE    := stacks/voice/compose.yaml
SMARTHOME := stacks/smarthome/compose.yaml

# All stacks in dependency order
ALL_STACKS := $(CORE) $(DATA) $(SOPHON) $(STORAGE) $(VOICE) $(SMARTHOME)

# Map friendly names to compose files (usage: make up STACK=core)
STACK_FILE = $(if $(filter core,$(STACK)),$(CORE),\
             $(if $(filter data,$(STACK)),$(DATA),\
             $(if $(filter sophon,$(STACK)),$(SOPHON),\
             $(if $(filter storage,$(STACK)),$(STORAGE),\
             $(if $(filter voice,$(STACK)),$(VOICE),\
             $(if $(filter smarthome,$(STACK)),$(SMARTHOME),\
             ))))))

# ==================== Helpers ====================

define check_secrets
	@if ! ls stacks/*/\.env 1>/dev/null 2>&1; then \
		printf "$(RED)✗ Secrets not decrypted$(NC)\n" >&2; \
		printf "$(YELLOW)  Run: make decrypt$(NC)\n" >&2; \
		exit 1; \
	fi
	@printf "$(GREEN)✓ Secrets decrypted$(NC)\n"
endef

define check_running
	@if ! docker compose -f $(CORE) ps --quiet 2>/dev/null | head -n1 | grep -q .; then \
		printf "$(RED)✗ Core stack is not running$(NC)\n" >&2; \
		printf "$(YELLOW)  Run: make up$(NC)\n" >&2; \
		exit 1; \
	fi
endef

define check_tools_fn
	@printf "$(YELLOW)Checking required tools...$(NC)\n"
	@command -v docker >/dev/null 2>&1 || { printf "$(RED)✗ Docker not installed$(NC)\n" >&2; exit 1; }
	@command -v sops >/dev/null 2>&1   || { printf "$(RED)✗ SOPS not installed$(NC)\n" >&2; exit 1; }
	@command -v age >/dev/null 2>&1    || { printf "$(RED)✗ age not installed$(NC)\n" >&2; exit 1; }
	@command -v restic >/dev/null 2>&1 || { printf "$(RED)✗ restic not installed$(NC)\n" >&2; exit 1; }
	@command -v bun >/dev/null 2>&1    || { printf "$(RED)✗ bun not installed$(NC)\n" >&2; exit 1; }
	@printf "$(GREEN)✓ All required tools installed$(NC)\n"
endef

# ==================== Help ====================

.PHONY: help
help: ## Show this help message
	@printf "$(BLUE)Universe 647 Management$(NC)\n"
	@printf "=======================\n\n"
	@printf "Usage: make <target> [STACK=core|data|sophon|storage|voice|smarthome]\n\n"
	@awk 'BEGIN {FS = ":.*##"; section=""} \
		/^##@/ { section=substr($$0, 5); printf "\n$(BLUE)%s$(NC)\n", section; next } \
		/^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2 }' \
		$(MAKEFILE_LIST)

.DEFAULT_GOAL := help

##@ Setup

.PHONY: setup
setup: ## Run first-time server setup script
	@printf "$(YELLOW)Running server setup...$(NC)\n"
	@./scripts/setup.sh
	@printf "$(GREEN)✓ Setup complete$(NC)\n"

.PHONY: check-tools
check-tools: ## Verify all required tools are installed
	$(check_tools_fn)

.PHONY: info
info: ## Display system and tool versions
	@printf "$(CYAN)System:$(NC)\n"
	@uname -a
	@printf "\n$(CYAN)Tools:$(NC)\n"
	@docker --version 2>/dev/null  || printf "$(RED)Docker not found$(NC)\n"
	@sops --version 2>/dev/null    || printf "$(RED)SOPS not found$(NC)\n"
	@age --version 2>/dev/null     || printf "$(RED)age not found$(NC)\n"
	@restic version 2>/dev/null    || printf "$(RED)restic not found$(NC)\n"
	@tailscale version 2>/dev/null || printf "$(RED)Tailscale not found$(NC)\n"
	@bun --version 2>/dev/null     || printf "$(RED)bun not found$(NC)\n"

.PHONY: age-keygen
age-keygen: ## Generate age keypair and install secret key
	@if [ -f ~/.config/sops/age/keys.txt ]; then \
		printf "$(RED)✗ Key already exists at ~/.config/sops/age/keys.txt$(NC)\n" >&2; \
		printf "$(YELLOW)  Delete it first if you want to regenerate$(NC)\n" >&2; \
		exit 1; \
	fi
	@mkdir -p ~/.config/sops/age
	@age-keygen -o ~/.config/sops/age/keys.txt
	@printf "$(GREEN)✓ Secret key saved to ~/.config/sops/age/keys.txt$(NC)\n"
	@grep "public key" ~/.config/sops/age/keys.txt | awk '{print $$NF}'
	@printf "$(YELLOW)  ⚠ Back up the secret key to 1Password NOW$(NC)\n"
	
##@ Secrets

.PHONY: decrypt
decrypt: ## Decrypt all .env.enc files → .env
	@printf "$(YELLOW)Decrypting secrets...$(NC)\n"
	@./scripts/decrypt-secrets.sh
	@printf "$(GREEN)✓ All secrets decrypted$(NC)\n"

.PHONY: encrypt
encrypt: ## Encrypt all .env files → .env.enc (run before git commit)
	@printf "$(YELLOW)Encrypting secrets...$(NC)\n"
	@./scripts/encrypt-secrets.sh
	@printf "$(GREEN)✓ All secrets encrypted$(NC)\n"
	@printf "$(YELLOW)  Remember: git add *.env.enc && git commit$(NC)\n"

##@ Services

.PHONY: build
build: ## Build custom images (STACK=sophon to build mcpo only)
ifdef STACK
	@if [ -z "$(STACK_FILE)" ]; then \
		printf "$(RED)✗ Unknown stack: $(STACK)$(NC)\n" >&2; \
		exit 1; \
	fi
	@printf "$(YELLOW)Building $(STACK) stack images...$(NC)\n"
	@docker compose -f $(STACK_FILE) build --pull
	@printf "$(GREEN)✓ $(STACK) stack images built$(NC)\n"
else
	@printf "$(YELLOW)Building all custom images...$(NC)\n"
	@for stack in $(ALL_STACKS); do \
		if grep -q "^\s*build:" $$stack 2>/dev/null; then \
			name=$$(basename $$(dirname $$stack)); \
			printf "$(CYAN)  ↳ Building $$name...$(NC)\n"; \
			docker compose -f $$stack build --pull; \
		fi; \
	done
	@printf "$(GREEN)✓ All custom images built$(NC)\n"
endif

.PHONY: up
up: ## Start stacks (all, or STACK=core|data|sophon|storage|voice|smarthome)
	$(check_secrets)
ifdef STACK
	@if [ -z "$(STACK_FILE)" ]; then \
		printf "$(RED)✗ Unknown stack: $(STACK)$(NC)\n" >&2; \
		printf "$(YELLOW)  Valid: core, data, sophon, storage, voice, smarthome$(NC)\n" >&2; \
		exit 1; \
	fi
	@printf "$(YELLOW)Starting $(STACK) stack...$(NC)\n"
	@if grep -q "^\s*build:" $(STACK_FILE) 2>/dev/null; then \
		printf "$(CYAN)  ↳ Building custom images...$(NC)\n"; \
		docker compose -f $(STACK_FILE) build --pull; \
	fi
	@if docker compose -f $(STACK_FILE) up -d; then \
		printf "$(GREEN)✓ $(STACK) stack started$(NC)\n"; \
	else \
		printf "$(RED)✗ Failed to start $(STACK) stack$(NC)\n" >&2; \
		exit 1; \
	fi
else
	@printf "$(YELLOW)Starting all stacks in dependency order...$(NC)\n"
	@for stack in $(ALL_STACKS); do \
		name=$$(basename $$(dirname $$stack)); \
		if grep -q "^\s*build:" $$stack 2>/dev/null; then \
			printf "$(CYAN)  ↳ Building $$name custom images...$(NC)\n"; \
			docker compose -f $$stack build --pull; \
		fi; \
		printf "$(CYAN)  ↳ Starting $$name...$(NC)\n"; \
		docker compose -f $$stack up -d || { \
			printf "$(RED)✗ Failed to start $$name$(NC)\n" >&2; \
			exit 1; \
		}; \
	done
	@printf "$(GREEN)✓ All stacks started$(NC)\n"
endif

.PHONY: down
down: ## Stop stacks (all, or STACK=name)
ifdef STACK
	@printf "$(YELLOW)Stopping $(STACK) stack...$(NC)\n"
	@docker compose -f $(STACK_FILE) down
	@printf "$(GREEN)✓ $(STACK) stack stopped$(NC)\n"
else
	@printf "$(YELLOW)Stopping all stacks...$(NC)\n"
	@for stack in $(SMARTHOME) $(VOICE) $(STORAGE) $(SOPHON) $(DATA) $(CORE); do \
		name=$$(basename $$(dirname $$stack)); \
		printf "$(CYAN)  ↳ Stopping $$name...$(NC)\n"; \
		docker compose -f $$stack down 2>/dev/null || true; \
	done
	@printf "$(GREEN)✓ All stacks stopped$(NC)\n"
endif

.PHONY: restart
restart: ## Restart stacks (all, or STACK=name)
	@$(MAKE) --no-print-directory down STACK=$(STACK)
	@$(MAKE) --no-print-directory up STACK=$(STACK)

.PHONY: pull
pull: ## Pull latest images for all stacks
	@printf "$(YELLOW)Pulling latest images...$(NC)\n"
	@for stack in $(ALL_STACKS); do \
		name=$$(basename $$(dirname $$stack)); \
		printf "$(CYAN)  ↳ Pulling $$name...$(NC)\n"; \
		docker compose -f $$stack pull 2>/dev/null || true; \
	done
	@printf "$(GREEN)✓ All images pulled$(NC)\n"

.PHONY: status
status: ## Show status of all containers
	@printf "$(YELLOW)Container status:$(NC)\n\n"
	@for stack in $(ALL_STACKS); do \
		name=$$(basename $$(dirname $$stack)); \
		if docker compose -f $$stack ps --quiet 2>/dev/null | head -n1 | grep -q .; then \
			printf "$(BLUE)$$name:$(NC)\n"; \
			docker compose -f $$stack ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null; \
			printf "\n"; \
		fi; \
	done

.PHONY: logs
logs: ## Tail logs (requires STACK=name)
ifndef STACK
	@printf "$(RED)✗ STACK is required$(NC)\n" >&2
	@printf "$(YELLOW)  Usage: make logs STACK=core$(NC)\n" >&2
	@exit 1
endif
	@docker compose -f $(STACK_FILE) logs -f

##@ Database

.PHONY: db-migrate
db-migrate: ## Run pending database migrations
	$(check_running)
	@printf "$(YELLOW)Running database migrations...$(NC)\n"
	@docker compose -f $(CORE) --profile migrate run --rm migrate
	@printf "$(GREEN)✓ Migrations complete$(NC)\n"

.PHONY: db-migrate-create
db-migrate-create: ## Create a new migration pair (usage: make db-migrate-create NAME=add_foo)
ifndef NAME
	@printf "$(RED)✗ NAME is required$(NC)\n" >&2
	@printf "$(YELLOW)  Usage: make db-migrate-create NAME=add_foo$(NC)\n" >&2
	@exit 1
endif
	@printf "$(YELLOW)Creating migration: $(NAME)...$(NC)\n"
	@docker run --rm -v ./stacks/core/postgres/migrations:/migrations \
		migrate/migrate:v4.18.1 create -ext sql -dir /migrations -seq $(NAME)
	@printf "$(GREEN)✓ Migration files created in stacks/core/postgres/migrations/$(NC)\n"

.PHONY: db-dump
db-dump: ## Dump PostgreSQL to /tmp/pg_dump_<date>.sql.gz
	$(check_running)
	@printf "$(YELLOW)Dumping PostgreSQL...$(NC)\n"
	@DUMP_FILE="/tmp/pg_dump_$$(date +%Y%m%d_%H%M).sql.gz"; \
	docker exec postgres pg_dumpall -U postgres | gzip -9 > $$DUMP_FILE; \
	printf "$(GREEN)✓ Dump saved to $$DUMP_FILE$(NC)\n"

.PHONY: db-shell
db-shell: ## Open PostgreSQL interactive shell
	$(check_running)
	@printf "$(YELLOW)Connecting to PostgreSQL...$(NC)\n"
	@docker exec -it postgres psql -U postgres

##@ Backup & Restore

.PHONY: backup
backup: ## Run full backup (local + Cloudflare R2)
	$(check_running)
	@printf "$(YELLOW)Starting backup...$(NC)\n"
	@./scripts/backup.sh
	@printf "$(GREEN)✓ Backup complete$(NC)\n"

.PHONY: backup-verify
backup-verify: ## Test restore to /tmp and verify integrity
	@printf "$(YELLOW)Verifying backup integrity...$(NC)\n"
	@./scripts/backup.sh --verify-only
	@printf "$(GREEN)✓ Backup verification complete$(NC)\n"

.PHONY: restore
restore: ## Restore from restic backup (interactive)
	@printf "$(RED)⚠️  This will overwrite current data with backup data$(NC)\n"
	@printf "Are you sure? (y/N): " && read confirm && \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		printf "$(YELLOW)Starting restore...$(NC)\n"; \
		./scripts/restore.sh; \
		printf "$(GREEN)✓ Restore complete$(NC)\n"; \
	else \
		printf "$(YELLOW)Restore cancelled$(NC)\n"; \
	fi

##@ Monitoring

.PHONY: setup-monitors
setup-monitors: ## Seed Uptime Kuma monitors (run once after first deploy)
ifndef UPTIME_KUMA_USER
	@printf "$(RED)✗ UPTIME_KUMA_USER is required$(NC)\n" >&2
	@printf "$(YELLOW)  Usage: make setup-monitors UPTIME_KUMA_USER=admin UPTIME_KUMA_PASS=yourpass$(NC)\n" >&2
	@exit 1
endif
ifndef UPTIME_KUMA_PASS
	@printf "$(RED)✗ UPTIME_KUMA_PASS is required$(NC)\n" >&2
	@printf "$(YELLOW)  Usage: make setup-monitors UPTIME_KUMA_USER=admin UPTIME_KUMA_PASS=yourpass$(NC)\n" >&2
	@exit 1
endif
	$(check_running)
	@printf "$(YELLOW)Seeding Uptime Kuma monitors...$(NC)\n"
	@docker run --rm \
		--network container:uptime-kuma \
		-e UPTIME_KUMA_URL=http://localhost:3001 \
		-e UPTIME_KUMA_USER=$(UPTIME_KUMA_USER) \
		-e UPTIME_KUMA_PASS=$(UPTIME_KUMA_PASS) \
		-v $(PWD)/scripts/setup-uptime-kuma.py:/setup.py:ro \
		python:3.11-slim \
		bash -c "pip install uptime-kuma-api --quiet && python /setup.py"
	@printf "$(GREEN)✓ Monitors seeded$(NC)\n"

##@ Security

.PHONY: trivy-scan
trivy-scan: ## Scan all running container images for vulnerabilities
	@printf "$(YELLOW)Scanning container images for vulnerabilities...$(NC)\n"
	@if ! command -v trivy >/dev/null 2>&1; then \
		printf "$(CYAN)Installing Trivy...$(NC)\n"; \
		docker run --rm aquasec/trivy --version >/dev/null 2>&1; \
	fi
	@docker ps --format '{{.Image}}' | sort -u | while read image; do \
		printf "\n$(CYAN)Scanning: $$image$(NC)\n"; \
		docker run --rm aquasec/trivy image --severity HIGH,CRITICAL --quiet "$$image" || true; \
	done
	@printf "\n$(GREEN)✓ Scan complete$(NC)\n"

.PHONY: revoke-device
revoke-device: ## Revoke a device (requires DEVICE=<tailscale-node-key>)
ifndef DEVICE
	@printf "$(RED)✗ DEVICE is required$(NC)\n" >&2
	@printf "$(YELLOW)  Usage: make revoke-device DEVICE=nodekey:abc123$(NC)\n" >&2
	@exit 1
endif
	@printf "$(RED)⚠️  Revoking device: $(DEVICE)$(NC)\n"
	@printf "This will remove the device from Tailscale and flush all Authelia sessions.\n"
	@printf "Are you sure? (y/N): " && read confirm && \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		./scripts/revoke-device.sh $(DEVICE); \
	else \
		printf "$(YELLOW)Revocation cancelled$(NC)\n"; \
	fi
