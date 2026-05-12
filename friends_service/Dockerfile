FROM golang:1.25-alpine AS builder

WORKDIR /friends-app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o bin/server ./cmd/main.go

FROM alpine:3.23

WORKDIR /friends-app

COPY --from=builder /friends-app/bin/server ./server

ENV APP_ENV_FILE=/etc/chat_service/.env.local
ENV APP_CONFIG_FILE=/etc/chat_service/local.yaml

EXPOSE 8083
CMD ./server -env $APP_ENV_FILE -config $APP_CONFIG_FILE


# EXPOSE 8083
# CMD ["./server", "-env", "/etc/friends_service/.env.local", "-config", "/etc/friends_service/local.yaml"]