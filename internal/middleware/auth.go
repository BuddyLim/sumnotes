package middleware

import (
	"main/internal/auth"
	"main/internal/database"
	"main/internal/model"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

var userCtxKey = "authedUser"

func SetUser(c *gin.Context, user *model.User) {
	c.Set(userCtxKey, user)
}

func GetUser(c *gin.Context) (*model.User, bool) {
	u, exists := c.Get(userCtxKey)
	if !exists {
		return nil, false
	}

	user, ok := u.(*model.User)
	return user, ok
}

// Auth is a middleware to protect routes that require authentication.
func Auth(store sessions.Store, db database.UserStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := auth.GetSession(store, c.Request)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		userID, ok := session.Values["user_id"].(string)

		if !ok || userID == "" {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		u, err := db.FindUserByID(userID)
		if err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		if time.Now().After(u.TokenExpiry) || time.Now().Equal(u.TokenExpiry) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		SetUser(c, u)

		c.Next()
	}
}
