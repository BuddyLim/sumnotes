package auth

import (
	"errors"
	"fmt"
	"main/internal/database"
	"main/internal/model"

	"github.com/markbates/goth"
)

var (
	ErrRefreshFailed = errors.New("failed to refresh token with provider")
)

func RefreshToken(u *model.User, db database.UserStore, p goth.Provider) error {
	newToken, err := p.RefreshToken(u.RefreshToken)
	if err != nil {
		return ErrRefreshFailed
	}

	err = db.UpdateUserTokens(u.ID, newToken.AccessToken, newToken.RefreshToken, newToken.Expiry)
	if err != nil {
		return fmt.Errorf("failed to update user tokens in database: %w", err)
	}
	return nil
}
