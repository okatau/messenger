#!/bin/bash
set -e

if [ ! -f ./config/.env.prod ]; then
  echo "Ошибка: файл .env.prod не найден. Скопируй .env.prod.example и заполни."
  exit 1
fi

source ./config/.env.prod

if [ -z "$DOMAIN" ] || [ -z "$CERTBOT_EMAIL" ]; then
  echo "Ошибка: DOMAIN и CERTBOT_EMAIL должны быть заполнены в .env.prod"
  exit 1
fi

COMPOSE="docker compose -f ./docker/docker-compose.prod.yml --env-file ./config/.env.prod"

echo "==> Убеждаемся что порт 80 свободен..."
$COMPOSE down --remove-orphans 2>/dev/null || true

echo "==> Получаем сертификат Let's Encrypt для $DOMAIN (standalone)..."
$COMPOSE run --rm -p 80:80 \
  --entrypoint "" certbot \
  certbot certonly --standalone \
  --config-dir /certdata \
  --work-dir /certdata/work \
  --logs-dir /certdata/logs \
  -d "$DOMAIN" \
  --email "$CERTBOT_EMAIL" \
  --agree-tos --no-eff-email

echo ""
echo "Готово! Сертификат получен. Теперь запускай весь стек:"
echo "  docker compose -f ./docker/docker-compose.prod.yml --env-file ./config/.env.prod up -d"
