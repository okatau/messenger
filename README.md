# Messenger

Бэкенд мессенджера на Go, реализованный как набор микросервисов. Сервисы общаются через HTTP REST API, входящий трафик маршрутизируется через Nginx.

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

### Требования

- Docker и Docker Compose

### Подготовка конфигурации

```bash
cd nginx_router
cp chats.env.example chats.env
```

Заполните в `chats.env` значения для JWT-ключей:

```env
AUTH_PUBLIC_PEM_BASE64=<base64 публичного RSA-ключа>
AUTH_PRIVATE_PEM_BASE64=<base64 приватного RSA-ключа>
```

Сгенерировать ключи можно так:

```bash
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem
AUTH_PRIVATE_PEM_BASE64=$(base64 -i private.pem)
AUTH_PUBLIC_PEM_BASE64=$(base64 -i public.pem)
```

### Запуск

```bash
cd nginx_router
docker compose up --build
```

После запуска API доступно на `http://localhost:8080`.

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
