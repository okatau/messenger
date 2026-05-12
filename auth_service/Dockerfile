FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o bin/server ./cmd/main.go

FROM alpine:3.23

WORKDIR /app

COPY --from=builder /app/bin/server ./server

ENV APP_ENV_FILE=/etc/auth_service/.env.local
ENV APP_CONFIG_FILE=/etc/auth_service/local.yaml

EXPOSE 8081
CMD ./server -env $APP_ENV_FILE -config $APP_CONFIG_FILE
