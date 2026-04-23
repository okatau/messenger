#!/bin/bash
set -e

if [ ! -f .env.prod ]; then
  echo "Ошибка: файл .env.prod не найден. Скопируй .env.prod.example и заполни."
  exit 1
fi

source .env.prod

if [ -z "$DOMAIN" ] || [ -z "$CERTBOT_EMAIL" ]; then
  echo "Ошибка: DOMAIN и CERTBOT_EMAIL должны быть заполнены в .env.prod"
  exit 1
fi

echo "==> Создаём временный self-signed сертификат для $DOMAIN..."
mkdir -p ./certbot/conf/live/$DOMAIN
openssl req -x509 -nodes -newkey rsa:2048 -days 1 \
  -keyout ./certbot/conf/live/$DOMAIN/privkey.pem \
  -out ./certbot/conf/live/$DOMAIN/fullchain.pem \
  -subj "/CN=$DOMAIN" 2>/dev/null

echo "==> Запускаем nginx с временным сертификатом..."
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d nginx-prod
sleep 3

echo "==> Удаляем временный сертификат..."
rm -rf ./certbot/conf/live/$DOMAIN

echo "==> Получаем настоящий сертификат Let's Encrypt для $DOMAIN..."
docker compose -f docker-compose.prod.yml --env-file .env.prod run --rm certbot \
  certonly --webroot -w /var/www/certbot \
  -d $DOMAIN \
  --email $CERTBOT_EMAIL \
  --agree-tos --no-eff-email

echo "==> Перезагружаем nginx с настоящим сертификатом..."
docker compose -f docker-compose.prod.yml --env-file .env.prod exec nginx-prod nginx -s reload

echo ""
echo "Готово! Сертификат получен. Теперь запускай весь стек:"
echo "  docker compose -f docker-compose.prod.yml --env-file .env.prod up -d"
