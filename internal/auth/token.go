package auth

import (
	"main/internal/database"
	"main/internal/model"

	"github.com/markbates/goth"
)

func RefreshToken(u *model.User, db database.UserStore, p goth.Provider) error {
	n, err := p.RefreshToken(u.RefreshToken)
	if err != nil {
		return err
	}

	return db.UpdateUserTokens(u.ID, n.AccessToken, n.RefreshToken, n.Expiry)

}
