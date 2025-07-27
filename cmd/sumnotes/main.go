package main

import (
	"log"
	"main/internal/config"
	"main/internal/database"
	"main/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	userStore := database.NewUserStore(db)

	srv, err := server.New(cfg, userStore)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	log.Println("Starting server on :9999")
	if err := srv.Run(":9999"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
