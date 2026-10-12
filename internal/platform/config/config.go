package config

import "os"

const defaultPort = "8080"

// Config contains runtime process configuration.
type Config struct {
	Port string
}

// Load reads runtime configuration from the environment.
func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	return Config{Port: port}
}

// Address returns the HTTP listen address.
func (c Config) Address() string {
	return ":" + c.Port
}
