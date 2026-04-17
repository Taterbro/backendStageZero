package database

type User struct {
	ID                 uint32  `json:"id"`
	Name               string  `json:"name"`
	Gender             string  `json:"gender"`
	GenderProbability  float64 `json:"gender_probability"`
	SampleSize         uint32  `json:"sample_size"`
	Age                uint32  `json:"age"`
	AgeGroup           string  `json:"age_group"`
	CountryID          string  `json:"country_id"`
	CountryProbability float64 `json:"country_probability"`
	CreatedAt          string  `json:"created_at"`
}

type Store struct {
	ById   map[uint32]*User
	ByName map[string]*User
}

var UserStore Store

func (s *Store) AddUser(user *User) {
	s.ById[user.ID] = user
	s.ByName[user.Name] = user
}

func (s *Store) GetById(id uint32) *User {
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

func (s *Store) DeleteUser(id uint32) {
	name := s.ById[id].Name
	delete(s.ById, id)
	delete(s.ByName, name)
}