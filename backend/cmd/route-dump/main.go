package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"nithronos/backend/nosd/internal/config"
	"nithronos/backend/nosd/internal/server"

	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := config.Defaults()
	r := server.NewRouter(cfg).(*chi.Mux)
	var routes []map[string]string
	_ = chi.Walk(r, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		routes = append(routes, map[string]string{"method": method, "path": route})
		return nil
	})
	b, _ := json.Marshal(routes)
	fmt.Println(string(b))
}
