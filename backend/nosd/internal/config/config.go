package config

import (
	"os"
	"strconv"

	"github.com/rs/zerolog"
)

type Config struct {
	Port            int
	LogLevel        zerolog.Level
	UsersPath       string
	SessionHashKey  []byte
	SessionBlockKey []byte
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

	users := os.Getenv("NOS_USERS_PATH")
	if users == "" {
		users = "./devdata/users.json"
	}

	hashKey := []byte(os.Getenv("NOS_SESSION_HASH_KEY"))
	blockKey := []byte(os.Getenv("NOS_SESSION_BLOCK_KEY"))
	if len(hashKey) == 0 {
		hashKey = []byte("0123456789abcdef0123456789abcdef") // 32 bytes
	}
	if len(blockKey) == 0 {
		blockKey = []byte("abcdef0123456789abcdef0123456789") // 32 bytes
	}

	return Config{
		Port:            port,
		LogLevel:        level,
		UsersPath:       users,
		SessionHashKey:  hashKey,
		SessionBlockKey: blockKey,
	}
}
