package config

import (
	"os"
	"strconv"

	"github.com/rs/zerolog"
)

type Config struct {
	Port int
	LogLevel zerolog.Level
}

func FromEnv() Config {
	port := 9000
	if v := os.Getenv("NOS_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			port = p
		}
	}

	level := zerolog.InfoLevel
	if v := os.Getenv("NOS_LOG"); v != "" {
		if l, err := zerolog.ParseLevel(v); err == nil {
			level = l
		}
	}

	return Config{
		Port:     port,
		LogLevel: level,
	}
}



