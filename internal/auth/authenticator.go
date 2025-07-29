package auth

import (
	"net/http"

	"github.com/markbates/goth"
)

// Authenticator describes an object that can complete user authentication.
type Authenticator interface {
	CompleteUserAuth(w http.ResponseWriter, r *http.Request) (goth.User, error)
}
