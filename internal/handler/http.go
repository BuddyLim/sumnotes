package handler

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"html/template"
	"main/internal/auth"
	"main/internal/database"
	"main/internal/model"
	"net/http"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/antonlindstrom/pgstore"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth/gothic"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type Handler struct {
	db    *sql.DB
	store *pgstore.PGStore
}

func New(db *sql.DB, store *pgstore.PGStore) *Handler {
	return &Handler{db, store}
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

	dbUser, err := database.FindUserByEmail(h.db, gothUser.Email)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if dbUser == nil {
		dbUser, err = database.CreateUser(h.db, &model.User{
			Email:     gothUser.Email,
			Name:      gothUser.Name,
			AvatarURL: gothUser.AvatarURL,
		})
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	err = database.UpdateUserTokens(h.db, dbUser.ID, gothUser.AccessToken, gothUser.RefreshToken, gothUser.ExpiresAt)
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

	c.Redirect(http.StatusTemporaryRedirect, "/success")
}

func (h *Handler) Success(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", fmt.Appendf(nil, `
      <div style="
          background-color: #fff;
          padding: 40px;
          border-radius: 8px;
          box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
          text-align: center;
      ">
          <h1 style="
              color: #333;
              margin-bottom: 20px;
          ">You have Successfull signed in!</h1>
          
          </div>
      </div>
  `))
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

	user, err := database.FindUserByID(h.db, userID)
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

	data, err := base64.URLEncoding.DecodeString(m.Payload.Parts[1].Body.Data)
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

	c.Data(http.StatusOK, "text/markdown; charset=utf-8", fmt.Appendf(nil, "%s", markdown))
}
