package config

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type ServerConfig struct {
	HTTPPort        int           `yaml:"http_port" env-default:"8080"`
	GRPCPort        int           `yaml:"grpc_port" env-default:"50051"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env-default:"10s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env-default:"10s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env-default:"10s"`
}

// No prefix needed
type RedisConfig struct {
	Addrs    []string `yaml:"addrs" env-required:"true"`
	Password string   `env:"REDIS_PASSWORD" env-required:"true"`
}

// Prefix needed
type PostgresConfig struct {
	Host     string `env:"HOST" env-required:"true"`
	Port     int    `env:"PORT" env-required:"true"`
	User     string `env:"USER" env-required:"true"`
	Password string `env:"PASSWORD" env-required:"true"`
	DBName   string `env:"DBNAME" env-required:"true"`
}

type AuthConfig struct {
	AccessTokenTTL      time.Duration `yaml:"access_token_ttl" env-default:"15m"`
	RefreshTokenTTL     time.Duration `yaml:"refresh_token_ttl" env-default:"720h"` // 30 days
	PublicKeyPEMBase64  string        `env:"AUTH_PUBLIC_PEM_BASE64" env-required:"true"`
	PrivateKeyPEMBase64 string        `env:"AUTH_PRIVATE_PEM_BASE64"`
}

func Load[T any]() *T {
	envPath, configPath := fetchPaths()

	if envPath == "" {
		log.Fatal("'.env' file path is empty")
	}

	if configPath == "" {
		log.Fatal("config path is empty")
	}

	if err := godotenv.Load(envPath); err != nil {
		log.Fatal("no .env file found")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg T
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("error reading config: %v", err)
	}

	return &cfg
}

func fetchPaths() (string, string) {
	var envPath, configPath string

	flag.StringVar(&envPath, "env", "", "path to '.env' file")
	flag.StringVar(&configPath, "config", "", "path to config file")
	flag.Parse()

	if envPath == "" {
		envPath = os.Getenv("ENV_PATH")
	}

	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
	}

	return envPath, configPath
}
