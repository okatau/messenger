COMPOSE_LOCAL = docker compose -f nginx_router/docker-compose.local.yml
ENV_FILE_LOCAL = --env-file=nginx_router/.env.local
COMPOSE_PROD = docker compose -f nginx_router/docker-compose.prod.yml
ENV_FILE_PROD = --env-file=nginx_router/.env.prod
COMPOSE_STAGING = docker compose -f nginx_router/docker-compose.prod.yml -f nginx_router/docker-compose.staging.yml

COMPOSE_FLAGS = --project-directory nginx_router

REGISTRY ?= $(shell grep '^REGISTRY=' nginx_router/.env.prod 2>/dev/null | cut -d= -f2)
VERSION  ?= $(shell git rev-parse --short HEAD)

local-up:
	$(COMPOSE_LOCAL) $(ENV_FILE_LOCAL) $(COMPOSE_FLAGS) up --build

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

staging-up:
	$(COMPOSE_STAGING) $(ENV_FILE_PROD) $(COMPOSE_FLAGS) up -d

staging-down:
	$(COMPOSE_STAGING) $(ENV_FILE_PROD) $(COMPOSE_FLAGS) down

staging-logs:
	$(COMPOSE_STAGING) $(ENV_FILE_PROD) $(COMPOSE_FLAGS) logs -f

build-prod:
	docker build -t $(REGISTRY)/auth:$(VERSION)     -t $(REGISTRY)/auth:latest     ./auth_service
	docker build -t $(REGISTRY)/chat:$(VERSION)     -t $(REGISTRY)/chat:latest     ./chat_service
	docker build -t $(REGISTRY)/friends:$(VERSION)  -t $(REGISTRY)/friends:latest  ./friends_service
	docker build -t $(REGISTRY)/frontend:$(VERSION) -t $(REGISTRY)/frontend:latest ./frontend

push-prod:
	docker push $(REGISTRY)/auth:$(VERSION)     && docker push $(REGISTRY)/auth:latest
	docker push $(REGISTRY)/chat:$(VERSION)     && docker push $(REGISTRY)/chat:latest
	docker push $(REGISTRY)/friends:$(VERSION)  && docker push $(REGISTRY)/friends:latest
	docker push $(REGISTRY)/frontend:$(VERSION) && docker push $(REGISTRY)/frontend:latest

release-prod: build-prod push-prod

.DEFAULT_GOAL := help
