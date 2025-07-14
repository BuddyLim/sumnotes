package config

import (
	"log"
	"os"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/markbates/goth/gothic"
)

type Config struct {
	ClientID          string
	ClientSecret      string
	ClientCallbackURL string
	DatabaseURL       string
	SessionSecret     string
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(".env file failed to load!")
	}

	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	clientCallbackURL := os.Getenv("CLIENT_CALLBACK_URL")
	databaseURL := os.Getenv("DATABASE_URL")
	sessionSecret := os.Getenv("SESSION_SECRET")

	if clientID == "" || clientSecret == "" || clientCallbackURL == "" || databaseURL == "" || sessionSecret == "" {
		log.Fatal("Environment variables (CLIENT_ID, CLIENT_SECRET, CLIENT_CALLBACK_URL, DATABASE_URL, SESSION_SECRET) are required")
	}
	gothic.Store = sessions.NewCookieStore([]byte(sessionSecret))

	return &Config{
		ClientID:          clientID,
		ClientSecret:      clientSecret,
		ClientCallbackURL: clientCallbackURL,
		DatabaseURL:       databaseURL,
		SessionSecret:     sessionSecret,
	}, nil
}
