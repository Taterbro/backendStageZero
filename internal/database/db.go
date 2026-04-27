package database

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Taterbro/backendStageZero/cmd/api/certs"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

var db *sql.DB
var data SeedData

func init() {
	log.Println("running init function")

	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(certs.CaCert)

	mysql.RegisterTLSConfig("aiven", &tls.Config{
		RootCAs: caPool,
	})
}

func Connect() {
	err := godotenv.Load()
	if err != nil {
		//log.Fatal("Error loading .env file")
		fmt.Println("couldn't load .env; proceeding since in prod")
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
	fmt.Println("database Connected!")
	Migrate(db)
	fmt.Println("Migrated successfully!")

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

	fmt.Println("seeding completed")
}
func QueryAllUsers(filters SearchFilter, limit int, offset int) ([]User, int, error) {
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
	countCommand := "SELECT COUNT(*) FROM profiles WHERE 1=1"

	args := make([]any, 0, 12)

	// Apply filters to BOTH queries
	if filters.Gender != nil {
		queryCommand += " AND gender = ?"
		countCommand += " AND gender = ?"
		args = append(args, *filters.Gender)
	}
	if filters.AgeGroup != nil {
		queryCommand += " AND age_group = ?"
		countCommand += " AND age_group = ?"
		args = append(args, *filters.AgeGroup)
	}
	if filters.CountryID != nil {
		queryCommand += " AND country_id = ?"
		countCommand += " AND country_id = ?"
		args = append(args, *filters.CountryID)
	}
	if filters.MinAge != nil {
		queryCommand += " AND age >= ?"
		countCommand += " AND age >= ?"
		args = append(args, *filters.MinAge)
	}
	if filters.MaxAge != nil {
		queryCommand += " AND age <= ?"
		countCommand += " AND age <= ?"
		args = append(args, *filters.MaxAge)
	}
	if filters.MinGenderProbability != nil {
		queryCommand += " AND gender_probability >= ?"
		countCommand += " AND gender_probability >= ?"
		args = append(args, *filters.MinGenderProbability)
	}
	if filters.MinCountryProbability != nil {
		queryCommand += " AND country_probability >= ?"
		countCommand += " AND country_probability >= ?"
		args = append(args, *filters.MinCountryProbability)
	}

	// Sorting (only for data query)
	if filters.SortBy != nil {
		if col, ok := allowedSort[*filters.SortBy]; ok {
			queryCommand += " ORDER BY " + col
			if filters.Order != nil {
				if ord, ok := allowedOrder[*filters.Order]; ok {
					queryCommand += " " + ord
				}
			}
		} else {
			return nil, 0, fmt.Errorf("invalid sort_by value")
		}
	} else {
		queryCommand += " ORDER BY created_at"
	}

	// 1. Get total count
	var totalCount int
	err := db.QueryRow(countCommand, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("queryAllUsers (count): %v", err)
	}

	// 2. Get paginated data
	dataArgs := append(args, limit, offset)

	rows, err := db.Query(queryCommand+" LIMIT ? OFFSET ?", dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("queryAllUsers (data): %v", err)
	}
	defer rows.Close()

	var users []User

	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Gender,
			&u.GenderProbability,
			&u.Age,
			&u.AgeGroup,
			&u.CountryID,
			&u.CountryName,
			&u.CountryProbability,
			&u.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("queryAllUsers (scan): %v", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("queryAllUsers (rows): %v", err)
	}

	return users, totalCount, nil
}
func DevQuery(q string) ([]User, error) {
	var users []User

	rows, err := db.Query((q))
	if err != nil {
		return nil, fmt.Errorf("dev query: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var alb User
		if err := rows.Scan(&alb.ID, &alb.Name, &alb.Gender, &alb.GenderProbability, &alb.Age, &alb.AgeGroup, &alb.CountryID, &alb.CountryName, &alb.CountryProbability, &alb.CreatedAt); err != nil {
			return nil, fmt.Errorf("queryAllUsers: %v", err)
		}
		users = append(users, alb)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("queryAllUsers: %v", err)
	}
	return users, nil
}

func QuerySingleProfileById(id string) (User, error) {
	var alb User

	row := db.QueryRow("SELECT * FROM profiles WHERE id = ?", id)
	if err := row.Scan(&alb.ID, &alb.Name, &alb.Gender, &alb.GenderProbability, &alb.Age, &alb.AgeGroup, &alb.CountryID, &alb.CountryName, &alb.CountryProbability, &alb.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return alb, fmt.Errorf("querySingleProfileById %s: no such album", id)
		}
		return alb, fmt.Errorf("querySingleProfileById %s: %v", id, err)
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
