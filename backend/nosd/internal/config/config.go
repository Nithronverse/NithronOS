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
	SecretPath      string
	FirstBootPath   string
	SharesPath      string
	SessionHashKey  []byte
	SessionBlockKey []byte
	EtcDir          string
	AppsDataDir     string
	AppsInstallDir  string
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
	secret := os.Getenv("NOS_SECRET_PATH")
	if secret == "" {
		secret = "/etc/nos/secret.key"
	}
	firstboot := os.Getenv("NOS_FIRSTBOOT_PATH")
	if firstboot == "" {
		firstboot = "/var/lib/nos/state/firstboot.json"
	}
	shares := os.Getenv("NOS_SHARES_PATH")
	if shares == "" {
		shares = "./devdata/shares.json"
	}

	hashKey := []byte(os.Getenv("NOS_SESSION_HASH_KEY"))
	blockKey := []byte(os.Getenv("NOS_SESSION_BLOCK_KEY"))
	if len(hashKey) == 0 {
		hashKey = []byte("0123456789abcdef0123456789abcdef") // 32 bytes
	}
	if len(blockKey) == 0 {
		blockKey = []byte("abcdef0123456789abcdef0123456789") // 32 bytes
	}

	etcDir := os.Getenv("NOS_ETC_DIR")
	if etcDir == "" {
		etcDir = "/etc"
	}
	appsData := os.Getenv("NOS_APPS_DATA_DIR")
	if appsData == "" {
		appsData = "/var/lib/nos/apps"
	}
	appsInstall := os.Getenv("NOS_APPS_INSTALL_DIR")
	if appsInstall == "" {
		appsInstall = "/opt/nos/apps"
	}

	return Config{
		Port:            port,
		LogLevel:        level,
		UsersPath:       users,
		SecretPath:      secret,
		FirstBootPath:   firstboot,
		SharesPath:      shares,
		SessionHashKey:  hashKey,
		SessionBlockKey: blockKey,
		EtcDir:          etcDir,
		AppsDataDir:     appsData,
		AppsInstallDir:  appsInstall,
	}
}
