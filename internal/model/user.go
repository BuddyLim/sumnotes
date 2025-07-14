package model

import "time"

type User struct {
	ID           string    `db:"id"`
	Email        string    `db:"email"`
	Name         string    `db:"name"`
	AvatarURL    string    `db:"avatar_url"`
	AccessToken  string    `db:"access_token"`
	RefreshToken string    `db:"refresh_token"`
	TokenExpiry  time.Time `db:"token_expiry"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}
