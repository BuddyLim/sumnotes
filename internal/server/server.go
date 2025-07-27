package server

import (
	"main/internal/auth"
	"main/internal/config"
	"main/internal/database"
	"main/internal/handler"
	"main/internal/middleware"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/google"
)

type Server struct {
	*gin.Engine
	db    database.UserStore
	store sessions.Store
}

func New(cfg *config.Config, db database.UserStore) (*Server, error) {
	r := gin.Default()

	store, err := auth.NewStore(cfg.DatabaseURL, []byte(cfg.SessionSecret))
	if err != nil {
		return nil, err
	}

	googleScope := []string{"https://www.googleapis.com/auth/gmail.modify", "https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"}

	gp := google.New(cfg.ClientID, cfg.ClientSecret, cfg.ClientCallbackURL, googleScope...)

	gp.SetPrompt("consent")

	goth.UseProviders(gp)

	r.LoadHTMLGlob("templates/*")

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	h := handler.New(db, store, cfg, gp)
	api := r.Group("/api")
	api.GET("/", h.Home)
	api.GET("/auth/:provider", h.SignInWithProvider)
	api.GET("/auth/:provider/callback", h.CallbackHandler)
	api.GET("/auth/refresh", h.Refresh)

	authorized := api.Group("/")
	authorized.Use(middleware.Auth(store, db))
	{
		authorized.GET("/me", h.Me)
		authorized.GET("/success", h.Success)
		authorized.GET("/summaries", h.Summaries)
	}

	return &Server{r, db, store}, nil
}
