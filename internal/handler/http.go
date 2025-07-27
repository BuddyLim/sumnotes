package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"main/internal/auth"
	"main/internal/config"
	"main/internal/database"
	"main/internal/model"
	"net/http"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type Handler struct {
	db    database.UserStore
	store sessions.Store
	cfg   *config.Config
	p     goth.Provider
}

func New(db database.UserStore, store sessions.Store, cfg *config.Config, p goth.Provider) *Handler {
	return &Handler{db, store, cfg, p}
}

func (h *Handler) Home(c *gin.Context) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(c.Writer, gin.H{})
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
}

func (h *Handler) SignInWithProvider(c *gin.Context) {
	provider := c.Param("provider")
	q := c.Request.URL.Query()
	q.Add("provider", provider)
	c.Request.URL.RawQuery = q.Encode()

	gothic.BeginAuthHandler(c.Writer, c.Request)
}

func (h *Handler) CallbackHandler(c *gin.Context) {
	provider := c.Param("provider")
	q := c.Request.URL.Query()
	q.Add("provider", provider)
	q.Del("scope")
	c.Request.URL.RawQuery = q.Encode()

	gothUser, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		fmt.Println("Error: ", err)
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	dbUser, err := h.db.FindUserByEmail(gothUser.Email)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if dbUser == nil {
		dbUser, err = h.db.CreateUser(&model.User{
			Email:     gothUser.Email,
			Name:      gothUser.Name,
			AvatarURL: gothUser.AvatarURL,
		})
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	err = h.db.UpdateUserTokens(dbUser.ID, gothUser.AccessToken, gothUser.RefreshToken, gothUser.ExpiresAt)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	session, err := auth.GetSession(h.store, c.Request)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	session.Values["user_id"] = dbUser.ID
	if err := session.Save(c.Request, c.Writer); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/api/success")
}

func (h *Handler) Success(c *gin.Context) {
	c.Redirect(http.StatusPermanentRedirect, h.cfg.FrontendURL)
}

func (h *Handler) Refresh(c *gin.Context) {
	session, err := auth.GetSession(h.store, c.Request)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	userID, ok := session.Values["user_id"].(string)
	if !ok || userID == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	user, err := h.db.FindUserByID(userID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if user == nil {
		c.AbortWithError(http.StatusNotFound, err)
		return
	}

	err = auth.RefreshToken(user, h.db, h.p)
	if err != nil {
		session.Options.MaxAge = -1
		if err := session.Save(c.Request, c.Writer); err != nil {
			panic("Failed to remove session for user: " + userID)
		}
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) Me(c *gin.Context) {
	session, err := auth.GetSession(h.store, c.Request)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	userID, ok := session.Values["user_id"].(string)
	if !ok || userID == "" {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		c.Abort()
		return
	}

	user, err := h.db.FindUserByID(userID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if user == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, struct {
		AvatarURL string
		CreatedAt time.Time
		Email     string
		ID        string
		Name      string
	}{
		AvatarURL: user.AvatarURL,
		CreatedAt: user.CreatedAt,
		Email:     user.Email,
		ID:        user.ID,
		Name:      user.Name,
	})
}

func (h *Handler) Summaries(c *gin.Context) {
	session, err := auth.GetSession(h.store, c.Request)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	userID, ok := session.Values["user_id"].(string)
	if !ok || userID == "" {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		c.Abort()
		return
	}

	user, err := h.db.FindUserByID(userID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if user == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	token := &oauth2.Token{
		AccessToken:  user.AccessToken,
		RefreshToken: user.RefreshToken,
		Expiry:       user.TokenExpiry,
		TokenType:    "Bearer",
	}

	ctx := context.Background()
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	gmailService, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	mails, err := gmailService.Users.Messages.List("me").Do()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	m, err := gmailService.Users.Messages.Get("me", mails.Messages[0].Id).Do()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	data, err := base64.URLEncoding.DecodeString(m.Payload.Body.Data)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	converter := md.NewConverter("", true, nil)

	markdown, err := converter.ConvertString(string(data))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(markdown))
}
