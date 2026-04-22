package database

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

var db *sql.DB
var data SeedData

func init() {
	caCert, err := os.ReadFile("ca.pem")
	if err != nil {
		panic(err)
	}

	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCert)

	mysql.RegisterTLSConfig("aiven", &tls.Config{
		RootCAs: caPool,
	})
}

func Connect() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	cfg := mysql.NewConfig()
	cfg.User = os.Getenv("DBUSER")
	cfg.Passwd = os.Getenv("DBPASS")
	cfg.Net = "tcp"
	cfg.Addr = os.Getenv("DBADDRESS")
	cfg.DBName = os.Getenv("DBNAME")
	cfg.TLSConfig = "aiven"

	// Get a database handle.
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Database Connected!")
	//Migrate(db)
	//fmt.Println("Migrated successfully!")

}

type User struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Gender             string  `json:"gender"`
	GenderProbability  float64 `json:"gender_probability"`
	Age                int     `json:"age"`
	AgeGroup           string  `json:"age_group"`
	CountryID          string  `json:"country_id"`
	CountryName        string  `json:"country_name"`
	CountryProbability float64 `json:"country_probability"`
	CreatedAt          string  `json:"created_at"`
}
type SeedData struct {
	Profiles []UserSeed `json:"profiles"`
}
type UserSeed struct {
	Name               string  `json:"name"`
	Gender             string  `json:"gender"`
	GenderProbability  float64 `json:"gender_probability"`
	Age                int     `json:"age"`
	AgeGroup           string  `json:"age_group"`
	CountryID          string  `json:"country_id"`
	CountryName        string  `json:"country_name"`
	CountryProbability float64 `json:"country_probability"`
}
type SearchFilter struct {
	Gender                *string
	AgeGroup              *string
	CountryID             *string
	MinAge                *int
	MaxAge                *int
	MinGenderProbability  *int
	MinCountryProbability *int
	SortBy                *string
	Order                 *string
}

func SeedDB() {
	file, err := os.ReadFile("internal/database/seed_profiles.json")
	if err != nil {
		log.Fatal("Error reading seed_profiles.json: \n", err)
	}

	err = json.Unmarshal(file, &data)
	if err != nil {
		log.Fatal(err)
	}

	// 🚀 START TRANSACTION
	tx, err := db.Begin()
	if err != nil {
		log.Fatal("failed to start transaction:", err)
	}

	// optional: rollback if something crashes
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO profiles (
			id, name, gender, gender_probability,
			age, age_group, country_id, country_name,
			country_probability, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for _, p := range data.Profiles {
		id := uuid.New().String()
		createdAt := time.Now()

		_, err := stmt.Exec(
			id,
			p.Name,
			p.Gender,
			p.GenderProbability,
			p.Age,
			p.AgeGroup,
			p.CountryID,
			p.CountryName,
			p.CountryProbability,
			createdAt,
		)

		if err != nil {
			log.Println("insert error:", err)
		}
	}

	// 🚀 COMMIT ONCE
	err = tx.Commit()
	if err != nil {
		log.Fatal("commit failed:", err)
	}

	fmt.Println("Seeding completed")
}
func QueryAllUsers(filters SearchFilter, limit int, offset int) ([]User, error) {
	allowedSort := map[string]string{
		"name":                "name",
		"age":                 "age",
		"created_at":          "created_at",
		"gender_probability":  "gender_probability",
		"country_probability": "country_probability",
	}

	allowedOrder := map[string]string{
		"asc":  "ASC",
		"desc": "DESC",
	}
	queryCommand := "SELECT * FROM profiles WHERE 1=1"
	var args = make([]any, 0, 12)
	if filters.Gender != nil {
		queryCommand += " AND gender = ?"
		args = append(args, *filters.Gender)
	}
	if filters.AgeGroup != nil {
		queryCommand += " AND age_group = ?"
		args = append(args, *filters.AgeGroup)
	}
	if filters.CountryID != nil {
		queryCommand += " AND country_id = ?"
		args = append(args, *filters.CountryID)

	}
	if filters.MinAge != nil {
		queryCommand += " AND age >= ?"
		args = append(args, *filters.MinAge)
	}
	if filters.MaxAge != nil {
		queryCommand += " AND age <= ?"
		args = append(args, *filters.MaxAge)
	}
	if filters.MinGenderProbability != nil {
		queryCommand += " AND gender_probability >= ?"
		args = append(args, *filters.MinGenderProbability)
	}
	if filters.MinCountryProbability != nil {
		queryCommand += " AND country_probability >= ?"
		args = append(args, *filters.MinCountryProbability)
	}
	if filters.SortBy != nil {
		if col, ok := allowedSort[*filters.SortBy]; ok {
			queryCommand += " ORDER BY " + col

			if filters.Order != nil {
				if ord, ok := allowedOrder[*filters.Order]; ok {
					queryCommand += " " + ord
				}
			}
		}
	}
	var users []User
	args = append(args, limit, offset)

	rows, err := db.Query(queryCommand+" LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, fmt.Errorf("QueryAllUsers: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var alb User
		if err := rows.Scan(&alb.ID, &alb.Name, &alb.Gender, &alb.GenderProbability, &alb.Age, &alb.AgeGroup, &alb.CountryID, &alb.CountryName, &alb.CountryProbability, &alb.CreatedAt); err != nil {
			return nil, fmt.Errorf("QueryAllUsers: %v", err)
		}
		users = append(users, alb)
	}
	fmt.Println("users is: \n", users)
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("QueryAllUsers: %v", err)
	}
	return users, nil
}

func QuerySingleProfileById(id string) (User, error) {
	var alb User

	row := db.QueryRow("SELECT * FROM profiles WHERE id = ?", id)
	if err := row.Scan(&alb.ID, &alb.Name, &alb.Gender, &alb.GenderProbability, &alb.Age, &alb.AgeGroup, &alb.CountryID, &alb.CountryName, &alb.CountryProbability, &alb.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return alb, fmt.Errorf("QuerySingleProfileById %s: no such album", id)
		}
		return alb, fmt.Errorf("QuerySingleProfileById %s: %v", id, err)
	}
	return alb, nil
}

type Store struct {
	ById   map[string]*User
	ByName map[string]*User
}

var UserStore = Store{
	ById:   make(map[string]*User),
	ByName: make(map[string]*User),
}

func (s *Store) AddUser(user *User) {
	s.ById[user.ID] = user
	s.ByName[user.Name] = user
}

func (s *Store) GetById(id string) *User {
	return s.ById[id]
}

func (s *Store) GetByName(name string) *User {
	return s.ByName[name]
}

func (s *Store) GetAllUsers() []User {
	var all = make([]User, 0, len(s.ById))
	for _, value := range s.ById {
		all = append(all, *value)
	}
	return all
}

func (s *Store) GetSomeUsers(gender string, ageGroup string, countryId string) []User {
	var all = make([]User, 0, len(s.ById))
	for _, value := range s.ById {
		if value.isFilterValid(gender, ageGroup, countryId) {
			all = append(all, *value)
		}
	}
	return all
}

func (s *Store) DeleteUser(id string) {
	name := s.ById[id].Name
	delete(s.ById, id)
	delete(s.ByName, name)
}

func (u *User) isFilterValid(gender string, ageGroup string, countryId string) bool {
	if gender != "" && !strings.EqualFold(u.Gender, gender) {
		return false
	}

	if ageGroup != "" && !strings.EqualFold(u.AgeGroup, ageGroup) {
		return false
	}

	if countryId != "" && !strings.EqualFold(u.CountryID, countryId) {
		return false
	}

	return true
}
