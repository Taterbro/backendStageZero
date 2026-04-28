package database

import (
	"database/sql"
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
	RevokedAt sql.NullTime
}

func AddRefreshToken(hash string, userId string) error {
	var token RefreshToken
	var revoked sql.NullTime
	token.ID = uuid.New().String()
	token.UserID = userId
	token.TokenHash = hash
	token.IssuedAt = time.Now()
	token.ExpiresAt = time.Now().Add(5 * time.Minute)
	token.RevokedAt = revoked
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
		revoked,
	)
	if err != nil {
		log.Println(err)
	}
	return err
}
