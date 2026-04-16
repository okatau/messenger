package config

import (
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type BaseConfig struct {
	Env    string        `env:"env" env-default:"local"`
	Server *ServerConfig `env-prefix:"SERVER_"`
}

type ServerConfig struct {
	Host string `env:"HOST" env-default:"0.0.0.0"`
	Port int    `env:"PORT" env-default:"8080"`
}

// TODO
type RedisConfig struct {
	Addr string `env:"ADDR" env-default:"redis:6379"`
}

type PostgresConfig struct {
	Host     string `env:"HOST" env-default:"postgres"`
	Port     int    `env:"PORT" env-default:"5432"`
	User     string `env:"USER" env-default:"postgres"`
	Password string `env:"PASSWORD" env-default:"postgres"`
	DBName   string `env:"DBNAME" env-default:"messenger"`
}

func Load[T any]() *T {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}

	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg T
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("error reading config: %v", err)
	}
	return &cfg
}
