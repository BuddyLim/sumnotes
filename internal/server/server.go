package server

import (
	"database/sql"
	"fmt"
	"main/internal/auth"
	"main/internal/config"
	"main/internal/handler"
	"main/internal/middleware"

	"github.com/antonlindstrom/pgstore"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/google"
)

type Server struct {
	*gin.Engine
	db    *sql.DB
	store *pgstore.PGStore
}

func New(cfg *config.Config, db *sql.DB) (*Server, error) {
	r := gin.Default()

	store, err := auth.NewStore(cfg.DatabaseURL, []byte(cfg.SessionSecret))
	if err != nil {
		return nil, err
	}

	googleScope := []string{"https://www.googleapis.com/auth/gmail.modify", "https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"}

	gp := google.New(cfg.ClientID, cfg.ClientSecret, cfg.ClientCallbackURL, googleScope...)

	gp.SetPrompt("consent")

	goth.UseProviders(gp)

	p, _ := goth.GetProvider("google")

	fmt.Println("Name: ", p.Name(), gp.Name())

	r.LoadHTMLGlob("templates/*")

	h := handler.New(db, store)

	r.GET("/", h.Home)
	r.GET("/auth/:provider", h.SignInWithProvider)
	r.GET("/auth/:provider/callback", h.CallbackHandler)

	authorized := r.Group("/")
	authorized.Use(middleware.Auth(store, db))
	{
		authorized.GET("/success", h.Success)
		authorized.GET("/summaries", h.Summaries)
	}

	return &Server{r, db, store}, nil
}
