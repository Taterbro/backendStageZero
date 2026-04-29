package database

import (
	"database/sql"
	"fmt"
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

type TokenGetter struct {
	ID     string
	Hash   string
	UserId string
}

func AddRefreshToken(hash string, userId string) error {
	var token RefreshToken
	var revoked sql.NullTime
	token.ID = uuid.New().String()
	token.UserID = userId
	token.TokenHash = hash
	token.IssuedAt = time.Now()
	token.ExpiresAt = time.Now().Add(5 * time.Minute)
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

func GetRefreshToken(id TokenGetter) (RefreshToken, error) {
	var (
		row   *sql.Row
		value string
		rt    RefreshToken
	)

	switch {
	case id.ID != "":
		row = db.QueryRow(
			`SELECT id, user_id, token_hash, issued_at, expires_at, revoked_at
			 FROM refresh_tokens
			 WHERE id = ?`,
			id.ID,
		)
		value = id.ID

	case id.Hash != "":
		row = db.QueryRow(
			`SELECT id, user_id, token_hash, issued_at, expires_at, revoked_at
			 FROM refresh_tokens
			 WHERE token_hash = ?`,
			id.Hash,
		)
		value = id.Hash

	case id.UserId != "":
		row = db.QueryRow(
			`SELECT id, user_id, token_hash, issued_at, expires_at, revoked_at
			 FROM refresh_tokens
			 WHERE user_id = ?
			 LIMIT 1`,
			id.UserId,
		)
		value = id.UserId

	default:
		return rt, fmt.Errorf("no token identifier provided")
	}

	err := row.Scan(
		&rt.ID,
		&rt.UserID,
		&rt.TokenHash,
		&rt.IssuedAt,
		&rt.ExpiresAt,
		&rt.RevokedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return rt, fmt.Errorf("no refresh token found for %s", value)
		}
		return rt, err
	}

	return rt, nil
}

func DeleteRefreshToken(id TokenGetter) error {
	var (
		result sql.Result
		err    error
		value  string
	)

	switch {
	case id.ID != "":
		result, err = db.Exec(
			"DELETE FROM refresh_tokens WHERE id = ?",
			id.ID,
		)
		value = id.ID

	case id.Hash != "":
		result, err = db.Exec(
			"DELETE FROM refresh_tokens WHERE token_hash = ?",
			id.Hash,
		)
		value = id.Hash

	case id.UserId != "":
		result, err = db.Exec(
			"DELETE FROM refresh_tokens WHERE user_id = ?",
			id.UserId,
		)
		value = id.UserId

	default:
		return fmt.Errorf("no token identifier provided")
	}

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no refresh token found for %s", value)
	}

	return nil
}
