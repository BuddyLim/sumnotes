package auth

import (
	"net/http"

	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
)

// GothicAuthenticator is the real implementation of the Authenticator interface.
type GothicAuthenticator struct{}

// NewGothicAuthenticator creates a new GothicAuthenticator.
func NewGothicAuthenticator() *GothicAuthenticator {
	return &GothicAuthenticator{}
}

// CompleteUserAuth wraps the call to gothic.CompleteUserAuth.
func (a *GothicAuthenticator) CompleteUserAuth(w http.ResponseWriter, r *http.Request) (goth.User, error) {
	return gothic.CompleteUserAuth(w, r)
}
