package database

import (
	"database/sql"
	"log"
)

var query = `
CREATE TABLE IF NOT EXISTS profiles (
	id VARCHAR(36) PRIMARY KEY,
	name VARCHAR(255),
	gender VARCHAR(50),
	gender_probability FLOAT,
	age INT,
	age_group VARCHAR(50),
	country_id VARCHAR(50),
	country_name VARCHAR(100),
	country_probability FLOAT,
	created_at TIMESTAMP
);
`

var tokensQuery = `
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP NULL,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
`

func Migrate(db *sql.DB) {
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}
