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

func Migrate(db *sql.DB) {
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}
