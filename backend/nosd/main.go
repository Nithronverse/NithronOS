package main

import (
	"fmt"
	"net/http"

	"nithronos/backend/nosd/internal/config"
	"nithronos/backend/nosd/internal/server"
)

func main() {
	cfg := config.FromEnv()
	r := server.NewRouter(cfg)

	addr := fmt.Sprintf("127.0.0.1:%d", cfg.Port)
	server.Logger(cfg).Info().Msgf("nosd listening on http://%s", addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		server.Logger(cfg).Fatal().Err(err).Msg("server exited")
	}
}
