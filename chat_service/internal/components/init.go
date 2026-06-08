package components

import (
	"context"
	"encoding/base64"
	"log"
	"log/slog"

	"chat_service/internal/clients"
	"chat_service/internal/db"
	"chat_service/internal/pubsub"
	"chat_service/internal/repository"
	"chat_service/internal/service"
	"chat_service/pkg/config"
	sl "chat_service/pkg/service_logger"
	"chat_service/pkg/token_manager"

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
	ServerConfig       config.ServerConfig `yaml:"http"`
	OriginWhitelist    []string            `yaml:"origin_whitelist"`
	FriendsGRPCAddress string              `yaml:"friends_grpc_addr" env:"FRIENDS_GRPC_ADDR" env-required:"true"`
}

type Components struct {
	Postgres     *pgxpool.Pool
	Redis        redis.UniversalClient
	Svc          service.Hub
	TokenManager *token_manager.TokenManager
	Logger       *slog.Logger
	grpcConn     *grpc.ClientConn
}

func InitComponents(ctx, hubCtx context.Context, cfg *Config) *Components {
	logger := sl.InitLogger(cfg.Env)

	pool := db.Connect(ctx, cfg.Postgres)
	if err := db.Run(cfg.Postgres); err != nil {
		log.Fatal("migration failed:", err)
	}

	rdb := initRedis(ctx, cfg.Redis)
	manager := initTokenManager(cfg.Auth, logger)
	svc, conn := initSvc(hubCtx, pool, rdb, logger, cfg.FriendsGRPCAddress)

	return &Components{
		Postgres:     pool,
		Redis:        rdb,
		Svc:          svc,
		TokenManager: manager,
		Logger:       logger,
		grpcConn:     conn,
	}
}

func (c *Components) Shutdown() {
	c.Postgres.Close()
	if err := c.Redis.Close(); err != nil {
		c.Logger.Error("error closing redis conn", sl.Err(err))
	}
	c.Svc.Shutdown()
	if err := c.grpcConn.Close(); err != nil {
		c.Logger.Error("error closing grpc conn", sl.Err(err))
	}
}

func initRedis(ctx context.Context, cfg config.RedisConfig) redis.UniversalClient {
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    cfg.Addrs,
		Password: cfg.Password,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}

	return rdb
}

func initTokenManager(cfg config.AuthConfig, logger *slog.Logger) *token_manager.TokenManager {
	pemBytes, err := base64.StdEncoding.DecodeString(cfg.PublicKeyPEMBase64)
	if err != nil {
		log.Fatal("error decoding public pem")
	}

	manager, err := token_manager.NewTokenManager(pemBytes, []byte{}, cfg.AccessTokenTTL, logger)
	if err != nil {
		log.Fatal(err)
	}

	return manager
}

func initSvc(
	hubCtx context.Context,
	pool *pgxpool.Pool,
	rdb redis.UniversalClient,
	logger *slog.Logger,
	grpcAddr string,
) (service.Hub, *grpc.ClientConn) {
	roomRepo := repository.NewRoomRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	msgRepo := repository.NewMessageRepository(pool, rdb)
	ps := pubsub.NewPubSub(rdb)

	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}

	friendsClient := clients.NewFriendshipClient(conn)

	return service.NewHub(hubCtx, userRepo, roomRepo, msgRepo, logger, friendsClient, ps), conn
}
