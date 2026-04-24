COMPOSE_LOCAL = docker compose -f nginx_router/docker-compose.local.yml --env-file
ENV_FILE_LOCAL = --env-file=nginx_router/.env.local
COMPOSE_PROD = docker compose -f nginx_router/docker-compose.prod.yml
ENV_FILE_PROD = --env-file=nginx_router/.env.prod

COMPOSE_FLAGS = --project-directory nginx_router

local-up:
	$(COMPOSE_LOCAL) $(ENV_FILE_LOCAL) $(COMPOSE_FLAGS) up -d --build

prod-up:
	$(COMPOSE_PROD) $(ENV_FILE_PROD) $(COMPOSE_FLAGS) up -d --build

local-down:
	$(COMPOSE_LOCAL) $(ENV_FILE_LOCAL) $(COMPOSE_FLAGS) down

prod-down:
	$(COMPOSE_PROD) $(ENV_FILE_PROD) $(COMPOSE_FLAGS) down

local-logs:
	$(COMPOSE_LOCAL) $(ENV_FILE_LOCAL) $(COMPOSE_FLAGS) logs -f

prod-logs:
	$(COMPOSE_PROD) $(ENV_FILE_PROD) $(COMPOSE_FLAGS) logs -f

local-logs-%:
	$(COMPOSE_LOCAL) $(ENV_FILE_LOCAL) $(COMPOSE_FLAGS) logs -f $*

prod-logs-%:
	$(COMPOSE_PROD) $(ENV_FILE_PROD) $(COMPOSE_FLAGS) logs -f $*

.DEFAULT_GOAL := help
