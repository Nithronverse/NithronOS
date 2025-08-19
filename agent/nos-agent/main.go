package main

import (
	"log"
	"nithronos/agent/nos-agent/internal/server"
)

func main() {
	log.Printf("nos-agent serving on unix socket %s", server.SocketPath)
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
