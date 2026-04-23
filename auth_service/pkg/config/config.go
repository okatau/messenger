package config

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type HTTPConfig struct {
	Port            int           `yaml:"port" env-required:"true"`
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
