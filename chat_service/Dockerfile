FROM golang:1.25-alpine AS builder

WORKDIR /chat-app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o bin/server ./cmd/main.go 

FROM alpine:3.23

WORKDIR /chat-app

COPY --from=builder /chat-app/bin/server ./server

ENV APP_ENV_FILE=/etc/chat_service/.env.local
ENV APP_CONFIG_FILE=/etc/chat_service/local.yaml

EXPOSE 8082
# CMD ["./server", "-env", "/etc/chat_service/.env.local", "-config", "/etc/chat_service/local.yaml"]
CMD ./server -env $APP_ENV_FILE -config $APP_CONFIG_FILE