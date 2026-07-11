package database

import (
	"database/sql"
	"time"
)

type TokenModel struct {
	DB *sql.DB
}

type RefreshToken struct {
	Id        int       `json:"id"`
	UserId    int       `json:"userId"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

func (m TokenModel) CreateRefreshToken(userId int, token string, expiresAt time.Time) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)
	`

	_, err := m.DB.Exec(query, userId, token, expiresAt)
	return err
}

func (m TokenModel) GetRefreshToken(token string) (RefreshToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at
		FROM refresh_tokens
		WHERE token = $1 AND expires_at > NOW()
	`

	var rt RefreshToken
	err := m.DB.QueryRow(query, token).Scan(
		&rt.Id,
		&rt.UserId,
		&rt.Token,
		&rt.ExpiresAt,
		&rt.CreatedAt,
	)
	if err != nil {
		return RefreshToken{}, err
	}

	return rt, nil
}

func (m TokenModel) DeleteRefreshToken(token string) error {
	query := `DELETE FROM refresh_tokens WHERE token = $1`
	_, err := m.DB.Exec(query, token)
	return err
}

func (m TokenModel) DeleteUserTokens(userId int) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`
	_, err := m.DB.Exec(query, userId)
	return err
}