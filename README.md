# Messenger

Бэкенд мессенджера на Go, реализованный как набор микросервисов. Сервисы общаются через HTTP REST API, входящий трафик маршрутизируется через Nginx.

Потестить сервис можно тут [messenger-pet.mooo.com](https://messenger-pet.mooo.com/)

## Архитектура

```
Client
  │
  ▼
Nginx (:8080)
  ├── /api/v1/auth/     → auth_service    (:8081)
  ├── /api/v1/rooms/    → chat_service    (:8082)
  └── /api/v1/friends/  → friends_service (:8083)
         │                      │
         └──────────────────────┘
                       │
              PostgreSQL + Redis
```

### Сервисы

| Сервис | Порт | Описание |
|---|---|---|
| `auth_service` | 8081 | Регистрация, логин, JWT-токены, сессии |
| `chat_service` | 8082 | Комнаты, сообщения, WebSocket-соединения |
| `friends_service` | 8083 | Список друзей, заявки в друзья, поиск пользователей |

### Инфраструктура

- **PostgreSQL 16** — основное хранилище (пользователи, сообщения, комнаты, дружба)
- **Redis** — используется в chat_service (активные соединения, состояние хаба)
- **Nginx** — API-gateway, проксирует запросы к сервисам

## API

### Auth (`/api/v1/auth`)

| Метод | Путь | Описание |
|---|---|---|
| `POST` | `/register` | Регистрация пользователя |
| `POST` | `/login` | Вход, возвращает access + refresh токены |
| `POST` | `/refresh` | Обновление access-токена по refresh-токену |
| `POST` | `/logout` | Выход, инвалидация сессии |

### Rooms (`/api/v1/rooms`)

| Метод | Путь | Описание |
|---|---|---|
| `GET` | `/wss` | WebSocket-подключение для real-time сообщений |
| `GET` | `/` | Список комнат пользователя |
| `POST` | `/` | Создать комнату |
| `GET` | `/:roomId/messages` | История сообщений комнаты |
| `GET` | `/:roomId/active` | Активные пользователи в комнате |
| `POST` | `/:roomId/invite` | Пригласить пользователя в комнату |
| `POST` | `/:roomId/leave` | Покинуть комнату |

### Friends (`/api/v1/friends`)

| Метод | Путь | Описание |
|---|---|---|
| `GET` | `/` | Список друзей |
| `GET` | `/search?username=` | Поиск пользователей по имени |
| `POST` | `/add` | Отправить заявку в друзья |
| `POST` | `/accept` | Принять заявку |
| `POST` | `/decline` | Отклонить заявку |
| `POST` | `/cancel` | Отменить отправленную заявку |
| `DELETE` | `/:friendId` | Удалить из друзей |

## Запуск

### Локально

**Требования:** Docker, Docker Compose

Конфигурация для локального окружения уже готова — файлы `config/*/local.yaml` и `config/*/.env.local` лежат в репозитории с дефолтными значениями (PostgreSQL: `postgres/postgres`, Redis: `redis`).

Если хотите использовать собственную пару RSA-ключей вместо тестовой:

```bash
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem

AUTH_PRIVATE_PEM_BASE64=$(base64 -i private.pem)
AUTH_PUBLIC_PEM_BASE64=$(base64 -i public.pem)
```

Вставьте полученные значения в `config/auth_service/.env.local`.

Запуск:

```bash
cd nginx_router
docker compose -f docker-compose.local.yml up --build
```

После старта:
- Фронтенд + API: `http://localhost:8080`
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`

Миграции применяются автоматически при первом запуске.

---

### На удалённом сервере

**Требования:** Docker, Docker Compose, публичный домен, container registry (например [GHCR](https://ghcr.io))

#### 1. Подготовка конфигурации

Скопируйте примеры конфигов и заполните их:

```bash
# Env-файлы для сервисов
cp config/auth_service/.env.prod.example    config/auth_service/.env.prod
cp config/chat_service/.env.prod.example    config/chat_service/.env.prod
cp config/friends_service/.env.prod.example config/friends_service/.env.prod

# YAML-конфиги для сервисов (менять обычно не нужно)
cp config/auth_service/prod.yaml.example    config/auth_service/prod.yaml
cp config/chat_service/prod.yaml.example    config/chat_service/prod.yaml
cp config/friends_service/prod.yaml.example config/friends_service/prod.yaml

# Env-файл для docker-compose и nginx
cp nginx_router/.env.prod.example nginx_router/.env.prod
```

Сгенерируйте RSA-ключи и запишите их в env-файлы сервисов:

```bash
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem

AUTH_PRIVATE_PEM_BASE64=$(base64 -i private.pem)
AUTH_PUBLIC_PEM_BASE64=$(base64 -i public.pem)
```

Вставьте значения `AUTH_PUBLIC_PEM_BASE64` и `AUTH_PRIVATE_PEM_BASE64` в:
- `config/auth_service/.env.prod`
- `config/chat_service/.env.prod`
- `config/friends_service/.env.prod`

Заполните `nginx_router/.env.prod`:

```env
DOMAIN=yourdomain.com
CERTBOT_EMAIL=you@example.com
REGISTRY=ghcr.io/your-username/messenger

PG_PASSWORD=your_secure_pg_password
REDIS_PASSWORD=your_secure_redis_password
```

#### 2. Сборка и публикация образов

Выполните на машине разработчика (или в CI):

```bash
REGISTRY=ghcr.io/your-username/messenger

docker build -t $REGISTRY/auth:latest    -f auth_service/Dockerfile.prod    auth_service/
docker build -t $REGISTRY/chat:latest    -f chat_service/Dockerfile.prod    chat_service/
docker build -t $REGISTRY/friends:latest -f friends_service/Dockerfile.prod friends_service/
docker build -t $REGISTRY/frontend:latest frontend/

docker push $REGISTRY/auth:latest
docker push $REGISTRY/chat:latest
docker push $REGISTRY/friends:latest
docker push $REGISTRY/frontend:latest
```

#### 3. Получение SSL-сертификата (Let's Encrypt)

Выполните один раз на сервере:

```bash
cd nginx_router
bash init-certbot.sh
```

Скрипт создаёт временный self-signed сертификат, поднимает nginx, получает настоящий сертификат от Let's Encrypt и перезагружает nginx.

#### 4. Запуск всего стека

```bash
cd nginx_router
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

После старта:
- Приложение доступно на `https://yourdomain.com`
- Сертификат обновляется автоматически через контейнер `certbot`
- Миграции применяются автоматически при первом запуске

## Структура проекта

```
messenger/
├── auth_service/        # Сервис аутентификации
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── handler/     # HTTP-хендлеры
│   │   ├── service/     # Бизнес-логика
│   │   ├── repository/  # Слой работы с БД
│   │   ├── domain/      # Сущности и ошибки
│   │   ├── middleware/  # Middleware
│   │   └── components/  # DI / инициализация зависимостей
│   └── pkg/
├── chat_service/        # Сервис чатов и WebSocket
│   └── ...
├── friends_service/     # Сервис друзей
│   └── ...
├── migrations/          # SQL-миграции (применяются при старте)
└── nginx_router/        # Nginx-конфиг и docker-compose
```

Каждый сервис — независимый Go-модуль со своим `go.mod`.

## Стек

- **Go 1.25**
- **Echo v5** — HTTP-фреймворк
- **PostgreSQL 16** + **pgx** — база данных
- **Redis** — кеш / pub-sub для чатов
- **JWT (RS256)** — аутентификация
- **testify/mock** — моки в тестах
