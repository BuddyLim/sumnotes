package auth

import (
	"net/http"

	"github.com/antonlindstrom/pgstore"
	"github.com/gorilla/sessions"
)

const (
	SessionName = "sumnotes_session"
)

func NewStore(dbURL string, keyPairs ...[]byte) (*pgstore.PGStore, error) {
	store, err := pgstore.NewPGStore(dbURL, keyPairs...)
	if err != nil {
		return nil, err
	}

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}

	return store, nil
}

func GetSession(store sessions.Store, r *http.Request) (*sessions.Session, error) {
	return store.Get(r, SessionName)
}
