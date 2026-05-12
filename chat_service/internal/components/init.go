package components

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"log/slog"

	"chat_service/internal/clients"
	"chat_service/internal/repository"
	"chat_service/internal/service"
	"chat_service/pkg/config"
	"chat_service/pkg/service_logger"
	"chat_service/pkg/token_manager"

	"github.com/go-redis/redis_rate/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Config struct {
	Env                string                `yaml:"env" env-default:"local"`
	Postgres           config.PostgresConfig `env-prefix:"PG_"`
	Redis              config.RedisConfig
	Auth               config.AuthConfig
	ServerConfig       config.HTTPConfig `yaml:"http"`
	Limits             RateLimiter       `yaml:"limits"`
	OriginWhitelist    []string          `yaml:"origin_whitelist"`
	FriendsGRPCAddress string            `yaml:"friends_grpc_addr" env:"FRIENDS_GRPC_ADDR" env-required:"true"`
}

type RateLimiter struct {
	CreateRoomLimit int `yaml:"create_room" env-default:"5"`
	InviteLimit     int `yaml:"invite" env-default:"10"`
	MessagesLimit   int `yaml:"messages" env-default:"30"`
}

type Components struct {
	Postgres     *pgxpool.Pool
	Redis        redis.UniversalClient
	Hub          service.Hub
	TokenManager *token_manager.TokenManager
	Logger       *slog.Logger
	Limiter      *redis_rate.Limiter
	grpcConn     *grpc.ClientConn
}

func InitComponents(ctx context.Context, hubCtx context.Context, cfg *Config) *Components {
	dsn := getPostgresDSN(cfg.Postgres)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	if err = pool.Ping(ctx); err != nil {
		log.Fatal(err)
	}

	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    cfg.Redis.Addrs,
		Password: cfg.Redis.Password,
	})
	if err = rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}

	limiter := redis_rate.NewLimiter(rdb)

	logger := service_logger.InitLogger(cfg.Env)

	pemBytes, err := base64.StdEncoding.DecodeString(cfg.Auth.PublicKeyPEMBase64)
	if err != nil {
		log.Fatal("error decoding public pem")
	}

	manager, err := token_manager.NewTokenManager(pemBytes, []byte{}, cfg.Auth.AccessTokenTTL, logger)
	if err != nil {
		log.Fatal(err)
	}

	roomRepo := repository.NewRoomRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	msgRepo := repository.NewMessageRepository(pool, rdb)

	conn, err := grpc.NewClient(
		cfg.FriendsGRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}

	friendsClient := clients.NewFriendshipClient(conn)

	hub := service.NewHub(hubCtx, userRepo, roomRepo, msgRepo, logger, friendsClient)

	return &Components{
		Postgres:     pool,
		Redis:        rdb,
		Hub:          hub,
		TokenManager: manager,
		Logger:       logger,
		Limiter:      limiter,
		grpcConn:     conn,
	}
}

func (c *Components) Shutdown(ctx context.Context) {
	c.Postgres.Close()
	c.Redis.Close()
	c.Hub.Shutdown(ctx)
	c.grpcConn.Close()
}

func getPostgresDSN(cfg config.PostgresConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
}
