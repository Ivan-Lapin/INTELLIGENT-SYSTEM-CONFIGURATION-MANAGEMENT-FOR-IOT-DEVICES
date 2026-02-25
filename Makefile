.PHONY: up down logs ps health restart

-include .env
export

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