package auth

import (
	"database/sql"
	"main/internal/database"
	"main/internal/model"

	"github.com/antonlindstrom/pgstore"
	"github.com/markbates/goth"
)

type Handler struct {
	db    *sql.DB
	store *pgstore.PGStore
}

func RefreshToken(u *model.User, db *sql.DB) error {
	p, err := goth.GetProvider("google")
	if err != nil {
		return err
	}

	n, err := p.RefreshToken(u.RefreshToken)
	if err != nil {
		return err
	}

	return database.UpdateUserTokens(db, u.ID, n.AccessToken, n.RefreshToken, n.Expiry)

}
