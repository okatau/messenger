package components

import (
	"api_gateway/pkg/config"
	"api_gateway/pkg/service_logger"
	"api_gateway/pkg/token_manager"
	"context"
	"encoding/base64"
	"log"
	"log/slog"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Env          string `env:"ENV" env-default:"local"`
	Redis        config.RedisConfig
	Auth         config.AuthConfig
	ServerConfig config.HTTPConfig `yaml:"http"`
	RateLimits   Limits            `yaml:"limits"`

	AuthAddr    string `yaml:"auth_addr" env-default:"http://auth-local:8081"`
	ChatAddr    string `yaml:"chat_addr" env-default:"http://chat-local:8082"`
	FriendsAddr string `yaml:"friends_addr" env-default:"http://friends-local:8083"`
}

type Limits struct {
	Al AuthLimits
	Cl ChatLimits
	Fl FriendsLimits
}

type AuthLimits struct {
	RegisterLimit int `yaml:"auth_register_limit" env-default:"10"`
	LoginLimit    int `yaml:"auth_login_limit" env-default:"5"`
}

type ChatLimits struct {
	CreateRoomLimit int `yaml:"chat_create_room" env-default:"5"`
	InviteLimit     int `yaml:"chat_invite" env-default:"10"`
	MessagesLimit   int `yaml:"chat_messages" env-default:"30"`
}

type FriendsLimits struct {
	SearchLimit int `yaml:"friends_search" env-default:"20"`
	AddLimit    int `yaml:"friends_add" env-default:"10"`
}

type Components struct {
	Limiter      *redis_rate.Limiter
	TokenManager *token_manager.TokenManager
	Logger       *slog.Logger
}

func InitComponents(ctx context.Context, cfg *Config) *Components {
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    cfg.Redis.Addrs,
		Password: cfg.Redis.Password,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}
	limiter := redis_rate.NewLimiter(rdb)

	logger := service_logger.InitLogger(cfg.Env)

	publicPemBytes, err := base64.StdEncoding.DecodeString(cfg.Auth.PublicKeyPEMBase64)
	if err != nil {
		log.Fatal("error decoding public pem")
	}
	privatePemBytes, err := base64.StdEncoding.DecodeString(cfg.Auth.PrivateKeyPEMBase64)
	if err != nil {
		log.Fatal("error decoding private pem")
	}

	manager, err := token_manager.NewTokenManager(publicPemBytes, privatePemBytes, cfg.Auth.AccessTokenTTL, logger)
	if err != nil {
		log.Fatal(err)
	}

	return &Components{
		Limiter:      limiter,
		TokenManager: manager,
		Logger:       logger,
	}
}

// func (c *Components) Shutdown(ctx context.Context) {
// 	// c.Postgres.Close()
// }
