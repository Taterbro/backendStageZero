package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func Connect(){
	var db *sql.DB

	err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file")
    }

	cfg := mysql.NewConfig()
    cfg.User = os.Getenv("DBUSER")
    cfg.Passwd = os.Getenv("DBPASS")
    cfg.Net = "tcp"
    cfg.Addr = "127.0.0.1:3306"
    cfg.DBName = "insighta_labs"

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
		if value.isFilterValid(gender, ageGroup, countryId){
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
	if gender != "" && !strings.EqualFold(u.Gender, gender){
		return  false
	}

	if ageGroup != "" && !strings.EqualFold(u.AgeGroup, ageGroup){
		return  false
	}

	if countryId != "" && !strings.EqualFold(u.CountryID, countryId){
		return  false
	}

	return true
}