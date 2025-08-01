package database

import (
	"database/sql"
	"main/internal/model"
	"time"

	"github.com/google/uuid"
)

// UserStorer defines the interface for user database operations.
type UserStore interface {
	FindUserByEmail(email string) (*model.User, error)
	FindUserByID(id string) (*model.User, error)
	CreateUser(user *model.User) (*model.User, error)
	UpdateUserTokens(userID, accessToken, refreshToken string, tokenExpiry time.Time) error
}

// DB holds the database connection pool.
type DB struct {
	*sql.DB
}

// NewUserStore creates a new DB instance.
func NewUserStore(db *sql.DB) *DB {
	return &DB{db}
}

func (db *DB) FindUserByEmail(email string) (*model.User, error) {
	user := &model.User{}
	var accessToken, refreshToken sql.NullString
	var tokenExpiry sql.NullTime

	err := db.QueryRow("SELECT id, email, name, avatar_url, access_token, refresh_token, token_expiry, created_at, updated_at FROM users WHERE email = $1", email).Scan(&user.ID, &user.Email, &user.Name, &user.AvatarURL, &accessToken, &refreshToken, &tokenExpiry, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No user found is not an error
		}
		return nil, err
	}

	user.AccessToken = accessToken.String
	user.RefreshToken = refreshToken.String
	user.TokenExpiry = tokenExpiry.Time

	return user, nil
}

func (db *DB) FindUserByID(id string) (*model.User, error) {
	user := &model.User{}
	var accessToken, refreshToken sql.NullString
	var tokenExpiry sql.NullTime

	err := db.QueryRow("SELECT id, email, name, avatar_url, access_token, refresh_token, token_expiry, created_at, updated_at FROM users WHERE id = $1", id).Scan(&user.ID, &user.Email, &user.Name, &user.AvatarURL, &accessToken, &refreshToken, &tokenExpiry, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No user found is not an error
		}
		return nil, err
	}

	user.AccessToken = accessToken.String
	user.RefreshToken = refreshToken.String
	user.TokenExpiry = tokenExpiry.Time

	return user, nil
}

func (db *DB) CreateUser(user *model.User) (*model.User, error) {
	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := db.Exec("INSERT INTO users (id, email, name, avatar_url, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
		user.ID, user.Email, user.Name, user.AvatarURL, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (db *DB) UpdateUserTokens(userID, accessToken, refreshToken string, tokenExpiry time.Time) error {
	_, err := db.Exec("UPDATE users SET access_token = $1, refresh_token = $2, token_expiry = $3, updated_at = $4 WHERE id = $5",
		accessToken, refreshToken, tokenExpiry, time.Now(), userID)
	return err
}
