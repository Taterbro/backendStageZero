package database

import (
	"log"
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	IssuedAt  time.Time
	ExpiresAt time.Time
	RevokedAt time.Time
}

func AddRefreshToken(hash string, userId string) error {
	var token RefreshToken
	token.ID = uuid.New().String()
	token.UserID = userId
	token.TokenHash = hash
	token.IssuedAt = time.Now()
	token.ExpiresAt = time.Now().Add(5 * time.Minute)
	token.RevokedAt = time.Time{}
	query := `
		INSERT INTO refresh_tokens (
			id,
			user_id,
			token_hash,
			issued_at,
			expires_at,
			revoked_at
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(
		query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.IssuedAt,
		token.ExpiresAt,
		token.RevokedAt,
	)
	if err != nil {
		log.Println(err)
	}
	return err
}
