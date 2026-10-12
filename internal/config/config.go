package config

import (
	"os"
	"strings"
)

const defaultPort = "8080"

type Config struct {
	Port string
}

func Load() Config {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = defaultPort
	}

	return Config{Port: port}
}

func (c Config) Address() string {
	return ":" + c.Port
}
