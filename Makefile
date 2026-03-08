-include .env
export

.PHONY: up down logs ps health restart

up:
	docker compose up -d

down:
	docker compose down

restart:
	docker compose down && docker compose up -d

logs:
	docker compose logs -f --tail=200

ps:
	docker compose ps

health:
	./tools/scripts/healthcheck.sh


.PHONY: up_all down_all rebuild_all logs_all

up_all:
	docker compose -f docker-compose.yml -f docker-compose.services.yml up -d --build

down_all:
	docker compose -f docker-compose.yml -f docker-compose.services.yml down

rebuild_all:
	docker compose -f docker-compose.yml -f docker-compose.services.yml up -d --build --force-recreate

reset_all:
	docker compose -f docker-compose.yml -f docker-compose.services.yml down -v
	docker compose -f docker-compose.yml -f docker-compose.services.yml up -d --build

logs_all:
	docker compose -f docker-compose.yml -f docker-compose.services.yml logs -f --tail=200