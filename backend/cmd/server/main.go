package main

import (
	"log"
	"net/http"

	"gitflame-codepilot/backend/internal/config"
	"gitflame-codepilot/backend/internal/httpapi"
)

func main() {
	cfg := config.Load()
	server, err := httpapi.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("GitFlame CodePilot backend listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, server.Router()); err != nil {
		log.Fatal(err)
	}
}
