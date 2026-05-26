package config

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

// ServerConfig holds HTTP/gRPC listen ports and timeout settings for the server.
type ServerConfig struct {
	Port            int           `yaml:"port" env-required:"true"`
	GRPCPort        int           `yaml:"grpc_port" env-default:"50051"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env-default:"10s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env-default:"10s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env-default:"10s"`
}

// RedisConfig holds the connection parameters for Redis.
// Env vars are read without a prefix (REDIS_PASSWORD is the full variable name).
type RedisConfig struct {
	Addrs    []string `yaml:"addrs" env-required:"true"`
	Password string   `env:"REDIS_PASSWORD" env-required:"true"`
}

// PostgresConfig holds the connection parameters for PostgreSQL.
// Env vars must be read with a service-specific prefix (e.g. cleanenv.ReadEnv with prefix "POSTGRES_").
type PostgresConfig struct {
	Host     string `env:"HOST" env-required:"true"`
	Port     int    `env:"PORT" env-required:"true"`
	User     string `env:"USER" env-required:"true"`
	Password string `env:"PASSWORD" env-required:"true"`
	DBName   string `env:"DBNAME" env-required:"true"`
}

// AuthConfig holds JWT key material and token lifetimes.
// PrivateKeyPEMBase64 is optional: omit it to run the manager in verify-only mode.
type AuthConfig struct {
	AccessTokenTTL      time.Duration `yaml:"access_token_ttl" env-default:"15m"`
	RefreshTokenTTL     time.Duration `yaml:"refresh_token_ttl" env-default:"720h"` // 30 days
	PublicKeyPEMBase64  string        `env:"AUTH_PUBLIC_PEM_BASE64" env-required:"true"`
	PrivateKeyPEMBase64 string        `env:"AUTH_PRIVATE_PEM_BASE64"`
}

// Load reads configuration into T from the .env file and YAML config file whose
// paths are resolved via -env / -config flags or ENV_PATH / CONFIG_PATH env vars.
// Calls log.Fatal if any required value is missing or the files cannot be read.
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

func fetchPaths() (envPath, configPath string) {
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
