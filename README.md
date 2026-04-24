# Messenger

Бэкенд мессенджера на Go, реализованный как набор микросервисов.

Потестить сервис можно тут [messenger-pet.mooo.com](https://messenger-pet.mooo.com/)

Cейчас реализовано:
  * **Регистрация.** При регистрации информация о пользователе добавляется в [бд](/migrations/001_users.up.sql), 
    генерируется access token и refresh token для автоматического логина в дальнейшем.
  * **Создание чатов.** Любой зарегистрированный пользователь может создавать чаты с кастомным названием.
  * **Добавление пользователей в чат.** Пользователей можно искать и добавлять по *username*


Что планируется сделать в ближайшее время:
  * Добавить rate limiter
  * Нормально интегрировать систему друзей. Сейчас используется только поиск пользователей.
  * Баг фикс

## Архитектура

```
Client
  │
  ▼
Nginx (:8080)
  ├── /api/v1/auth/     → auth_service    (:8081)
  ├── /api/v1/rooms/    → chat_service    (:8082) ── Redis
  └── /api/v1/friends/  → friends_service (:8083)
         │                      │
         └──────────────────────┘
                       │
              PostgreSQL
```

### Сервисы

| Сервис | Описание |
|---|---|
| `auth_service` | Регистрация, логин, JWT-токены, сессии |
| `chat_service` | Комнаты, сообщения, WebSocket-соединения |
| `friends_service` | Список друзей, заявки в друзья, поиск пользователей |

### Инфраструктура

- **PostgreSQL 16** — основное хранилище (пользователи, сообщения, комнаты, дружба)
- **Redis** — используется в chat_service (активные соединения)
- **Nginx** — API-gateway, проксирует запросы к сервисам

## Запуск
### Установка
```bash
git clone https://github.com/okatau/messenger.git
cd messenger
```

### Предварительная настройка
Для локального развертывания можно пропустить эту часть, т.к. все конфиги готовы `config/*/local.yaml` и [env](/nginx_router/.env.local).
И перейти [сюда](#локально)

```bash
cp nginx_router/.env.example nginx_router/.env.prod
```

Для деплоя на проде нужно заполнить [env](/nginx_router/.env.prod)

Команды для генерации RSA ключей
```bash
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem

base64 -w 0 -i private.pem
base64 -w 0 -i public.pem
```

Вставьте значения `AUTH_PUBLIC_PEM_BASE64` и `AUTH_PRIVATE_PEM_BASE64` в `nginx_router/.env.prod`

#### Получение SSL-сертификата (Let's Encrypt)

Выполните один раз на сервере:

```bash
cd nginx_router
bash init-certbot.sh
```

Скрипт создаёт временный self-signed сертификат, поднимает nginx, получает настоящий сертификат от Let's Encrypt и перезагружает nginx.

---

### Локально

Готовый конфиг для локального запуска есть - [конфиг](/nginx_router/.env.local)

```bash
cd frontend && npm install 
cd .. && make local-up
```

### На удаленном сервере

```bash
git clone https://github.com/okatau/messenger.git
scp -r messenger user@ip_address:/var/www/messenger
ssh user@ip_address

cd /var/www/messenger/frontend && rm -rf node_modules .next package-lock.json
npm cache clean --force
npm install 
cd .. && make prod-up
```

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


## Стек

- **Go**
- **Echo v5** — HTTP-фреймворк
- **PostgreSQL 16** + **pgx** — база данных
- **Redis** — кеш для храннеия сообщений
- **JWT (RS256)** — аутентификация
- **testify/mock** — моки в тестах
