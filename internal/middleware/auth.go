package middleware

import (
	"database/sql"
	"main/internal/auth"
	"main/internal/database"
	"net/http"
	"time"

	"github.com/antonlindstrom/pgstore"
	"github.com/gin-gonic/gin"
)

// Auth is a middleware to protect routes that require authentication.
func Auth(store *pgstore.PGStore, db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := auth.GetSession(store, c.Request)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		userID, ok := session.Values["user_id"].(string)

		if !ok || userID == "" {
			c.Redirect(http.StatusTemporaryRedirect, "/")
			c.Abort()
			return
		}

		u, err := database.FindUserByID(db, userID)
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, "/")
			c.Abort()
			return
		}

		if time.Now().After(u.TokenExpiry) || time.Now().Equal(u.TokenExpiry) {
			if err = auth.RefreshToken(u, db); err != nil {
				session.Options.MaxAge = -1
				if err := session.Save(c.Request, c.Writer); err != nil {
					panic("Failed to remove session for user: " + userID)
				}
				c.Redirect(http.StatusTemporaryRedirect, "/")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
