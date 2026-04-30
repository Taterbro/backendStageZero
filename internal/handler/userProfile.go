package handler

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"time"

	"golang.org/x/sync/errgroup"

	"github.com/Taterbro/backendStageZero/internal/database"
	"github.com/Taterbro/backendStageZero/internal/model"
	"github.com/Taterbro/backendStageZero/internal/service"
	"github.com/Taterbro/backendStageZero/internal/utils"
	"github.com/google/uuid"
)

type Request struct {
	Name string `json:"name"`
}

func Seed(w http.ResponseWriter, r *http.Request) {
	database.SeedDB()
	utils.WriteJson(w, http.StatusOK, model.SuccessResponse{
		Status: "success",
		Data:   "seeded fr fr",
	})
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")
	parts := strings.Split(header, " ")
	userId, _ := utils.GetUserIDFromToken(parts[1])
	userAccount, _ := database.GetAccount(database.GetAccountType{Id: userId})
	if userAccount.Role != "admin" {
		utils.WriteJson(w, http.StatusUnauthorized, model.ErrorResponse{
			Status:  "error",
			Message: "this function cannot be performed by you",
		})
		return
	}

	var req Request

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "Invalid request body",
		})
		return
	}

	name := strings.ToLower(strings.TrimSpace(req.Name))
	existingUser, getErr := database.GetUserByName(name)
	if getErr == nil {
		utils.WriteJson(w, http.StatusOK, model.UserSuccessResponse{
			Status:  "success",
			Message: "Profile already exists",
			Data:    existingUser,
		})
		return
	}

	if name == "" {
		utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "Name field is required",
		})
		return
	}

	if _, err := strconv.Atoi(name); err == nil {
		utils.WriteJson(w, http.StatusUnprocessableEntity, model.ErrorResponse{
			Status:  "error",
			Message: "name should not be only numbers",
		})
		return
	}

	var (
		agifyData       *model.AgifyResponse
		genderData      *model.GenderizeResponse
		nationalityData *model.NationalizeResponse
	)

	g, _ := errgroup.WithContext(r.Context())

	g.Go(func() error {
		var err error
		agifyData, err = service.GetAge(name)
		if err != nil {
			return err
		}
		return err
	})

	g.Go(func() error {
		var err error
		genderData, err = service.GetGender(name)
		if err != nil {
			return err
		}
		return err
	})

	g.Go(func() error {
		var err error
		nationalityData, err = service.GetNation(name)
		if err != nil {
			return err
		}
		return err
	})

	if err := g.Wait(); err != nil {
		log.Println("go routines tweaking: ", err)
		utils.WriteJson(w, http.StatusBadGateway, model.ErrorResponse{
			Status:  "error",
			Message: "Failed to fetch external data",
		})
		return
	}

	if agifyData == nil || agifyData.Age == 0 {
		utils.WriteJson(w, http.StatusBadGateway, model.ErrorResponse{
			Status:  "error",
			Message: "Agify returned invalid data",
		})
		return
	}

	if genderData == nil || genderData.Gender == "" {
		utils.WriteJson(w, http.StatusBadGateway, model.ErrorResponse{
			Status:  "error",
			Message: "Genderize returned invalid data",
		})
		return
	}

	if nationalityData == nil || len(nationalityData.Country) == 0 {

		utils.WriteJson(w, http.StatusBadGateway, model.ErrorResponse{
			Status:  "error",
			Message: "Nationalize returned invalid data",
		})
		return
	}

	ageGroup := "child"
	if agifyData.Age >= 18 {
		ageGroup = "adult"
	}

	user := database.User{
		ID:                 uuid.New().String(),
		Name:               name,
		Gender:             genderData.Gender,
		GenderProbability:  float64(genderData.Probability),
		Age:                agifyData.Age,
		AgeGroup:           ageGroup,
		CountryID:          nationalityData.Country[0].CountryId,
		CountryName:        nationalityData.Country[0].CountryId, // replace if API provides full name
		CountryProbability: float64(nationalityData.Country[0].Probability),
		CreatedAt:          time.Now().UTC(),
	}

	err := database.AddProfile(user)

	if err != nil {
		log.Println("error is: ", err)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "Failed to create profile",
		})
		return
	}

	utils.WriteJson(w, http.StatusCreated, model.SuccessResponse{
		Status: "success",
		Data:   user,
	})
}

func FindUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	fmt.Println("id is: ", id)
	user, err := database.QuerySingleProfileById(id)
	if err != nil {
		utils.WriteJson(w, http.StatusNotFound, model.ErrorResponse{
			Status:  "error",
			Message: "Invalid user id; user not found",
		})
		return
	}
	utils.WriteJson(w, http.StatusOK, model.SuccessResponse{
		Status: "success",
		Data:   user,
	})
}

func GetAllUsers(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("X-API-Version")
	if header == "" {
		utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "API version header required",
		})
		return
	}
	if header != "1" && header != "2" {
		utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "API version is invalid",
		})
		return
	}
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	var filters database.SearchFilter
	page := 1
	limit := 10

	if q != "" {
		parsedFilters, err := ParseNaturalLanguageQuery(q)
		if err != nil {
			utils.WriteJson(w, http.StatusUnprocessableEntity, model.ErrorResponse{
				Status:  "error",
				Message: "Unable to interpret query",
			})
			return
		}
		filters = parsedFilters
	}

	if gender := strings.ToLower(r.URL.Query().Get("gender")); gender != "" {
		filters.Gender = &gender
	}

	if countryId := strings.ToLower(r.URL.Query().Get("country_id")); countryId != "" {
		filters.CountryID = &countryId
	}

	if ageGroup := strings.ToLower(r.URL.Query().Get("age_group")); ageGroup != "" {
		filters.AgeGroup = &ageGroup
	}

	if minAge := r.URL.Query().Get("min_age"); minAge != "" {
		minAgeInt, err := strconv.Atoi(minAge)
		if err != nil {
			utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
				Status:  "error",
				Message: "min_age should be a number",
			})
			return
		}
		filters.MinAge = &minAgeInt
	}

	if maxAge := r.URL.Query().Get("max_age"); maxAge != "" {
		maxAgeInt, err := strconv.Atoi(maxAge)
		if err != nil {
			utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
				Status:  "error",
				Message: "max_age should be a number",
			})
			return
		}
		filters.MaxAge = &maxAgeInt
	}

	if minGenderProbability := r.URL.Query().Get("min_gender_probability"); minGenderProbability != "" {
		val, err := strconv.Atoi(minGenderProbability)
		if err != nil {
			utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
				Status:  "error",
				Message: "min_gender_probability should be a number",
			})
			return
		}
		filters.MinGenderProbability = &val
	}

	if minCountryProbability := r.URL.Query().Get("min_country_probability"); minCountryProbability != "" {
		val, err := strconv.Atoi(minCountryProbability)
		if err != nil {
			utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
				Status:  "error",
				Message: "min_country_probability should be a number",
			})
			return
		}
		filters.MinCountryProbability = &val
	}

	if sortBy := strings.ToLower(r.URL.Query().Get("sort_by")); sortBy != "" {
		filters.SortBy = &sortBy
	}

	if order := strings.ToLower(r.URL.Query().Get("order")); order != "" {
		filters.Order = &order
	}

	if qpage := r.URL.Query().Get("page"); qpage != "" {
		p, err := strconv.Atoi(qpage)
		if err != nil || p < 1 {
			utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
				Status:  "error",
				Message: "page should be a valid number",
			})
			return
		}
		page = p
	}

	if qLimit := r.URL.Query().Get("limit"); qLimit != "" {
		l, err := strconv.Atoi(qLimit)
		if err != nil || l < 1 {
			utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
				Status:  "error",
				Message: "limit should be a valid number",
			})
			return
		}
		if l > 50 {
			l = 50
		}
		limit = l
	}

	offset := (page - 1) * limit

	users, total, err := database.QueryAllUsers(filters, limit, offset)
	if err != nil {
		if strings.Contains(err.Error(), "invalid") {
			utils.WriteJson(w, http.StatusUnprocessableEntity, model.ErrorResponse{
				Status:  "error",
				Message: "Invalid query parameters",
			})
			return
		}

		utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "Unable to interpret query",
		})
		return
	}

	if header == "1" {
		utils.WriteJson(w, http.StatusOK, model.GetUserSuccessResponse{
			Status: "success",
			Page:   page,
			Limit:  limit,
			Total:  total,
			Data:   users,
		})
		return
	}
	if header == "2" {
		totalPages := (total + limit - 1) / limit // proper ceiling division

		next := ""
		prev := ""

		if page < totalPages {
			u := *r.URL // copy URL struct

			q := u.Query() // clone query params
			q.Set("page", strconv.Itoa(page+1))
			u.Scheme = "https"
			u.Host = r.Host
			u.RawQuery = q.Encode()
			next = u.String()
		}

		if page > 1 {
			u := *r.URL // copy URL struct

			q := u.Query() // clone query params
			q.Set("page", strconv.Itoa(page-1))
			u.Scheme = "https"
			u.Host = r.Host
			u.RawQuery = q.Encode()
			prev = u.String()
		}

		utils.WriteJson(w, http.StatusOK, map[string]any{
			"status":      "success",
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
			"next":        next,
			"prev":        prev,
			"data":        users,
		})
		return
	}

}

func ParseNaturalLanguageQuery(q string) (database.SearchFilter, error) {
	q = normalizeQuery(q)

	var filters database.SearchFilter
	matched := false

	// gender
	if hasAnyWord(q, "male", "males") && hasAnyWord(q, "female", "females") {
		// "male and female" means no gender filter
		matched = true
	} else if hasAnyWord(q, "male", "males") {
		g := "male"
		filters.Gender = &g
		matched = true
	} else if hasAnyWord(q, "female", "females") {
		g := "female"
		filters.Gender = &g
		matched = true
	}

	// age groups
	if hasAnyWord(q, "teenager", "teenagers") {
		ag := "teenager"
		filters.AgeGroup = &ag
		matched = true
	}
	if hasAnyWord(q, "adult", "adults") {
		ag := "adult"
		filters.AgeGroup = &ag
		matched = true
	}

	// "young" => 16-24
	if hasAnyWord(q, "young") {
		min := 16
		max := 24
		filters.MinAge = &min
		filters.MaxAge = &max
		matched = true
	}

	// country: "from nigeria"
	if m := regexp.MustCompile(`\bfrom\s+([a-z ]+)`).FindStringSubmatch(q); len(m) == 2 {
		country := strings.TrimSpace(m[1])
		if alias, ok := CountryAliases[country]; ok {
			country = alias
		}
		code, ok := CountryCodes[country]
		if !ok {
			return filters, errors.New("unable to interpret query")
		}
		filters.CountryID = &code
		matched = true
	}

	// "above 30", "over 30", "older than 30"
	if m := regexp.MustCompile(`\b(?:above|over|older than)\s+(\d+)\b`).FindStringSubmatch(q); len(m) == 2 {
		n, _ := strconv.Atoi(m[1])
		filters.MinAge = &n
		matched = true
	}

	// "below 30", "under 30", "younger than 30"
	if m := regexp.MustCompile(`\b(?:below|under|younger than)\s+(\d+)\b`).FindStringSubmatch(q); len(m) == 2 {
		n, _ := strconv.Atoi(m[1])
		filters.MaxAge = &n
		matched = true
	}

	// "between 18 and 25"
	if m := regexp.MustCompile(`\bbetween\s+(\d+)\s+and\s+(\d+)\b`).FindStringSubmatch(q); len(m) == 3 {
		min, _ := strconv.Atoi(m[1])
		max, _ := strconv.Atoi(m[2])
		filters.MinAge = &min
		filters.MaxAge = &max
		matched = true
	}

	// "17+" or "age 17+"
	if m := regexp.MustCompile(`\b(\d+)\+\b`).FindStringSubmatch(q); len(m) == 2 {
		n, _ := strconv.Atoi(m[1])
		filters.MinAge = &n
		matched = true
	}

	if !matched {
		return filters, errors.New("unable to interpret query")
	}
	return filters, nil
}

func normalizeQuery(q string) string {
	q = strings.ToLower(strings.TrimSpace(q))
	q = regexp.MustCompile(`[^\w\s]+`).ReplaceAllString(q, " ")
	q = strings.Join(strings.Fields(q), " ")
	return q
}

func hasAnyWord(q string, words ...string) bool {
	for _, word := range words {
		pattern := `\b` + regexp.QuoteMeta(word) + `\b`
		if regexp.MustCompile(pattern).FindStringIndex(q) != nil {
			return true
		}
	}
	return false
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	targetId := r.PathValue("id")
	exists := database.UserStore.GetById(targetId)
	if exists == nil {
		utils.WriteJson(w, http.StatusNotFound, model.ErrorResponse{
			Status:  "error",
			Message: "Invalid user id; user not found",
		})
		return
	}
	database.UserStore.DeleteUser(targetId)

	utils.WriteJson(w, http.StatusOK, model.SuccessResponse{
		Status: "success",
		Data:   nil,
	})

}

func ExportProfilesCSV(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("X-API-Version")
	if header == "" {
		utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "API version header required",
		})
		return
	}

	if header != "1" && header != "2" {
		utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "API version is invalid",
		})
		return
	}

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format != "csv" {
		utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "format must be csv",
		})
		return
	}

	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))

	var filters database.SearchFilter
	limit := 10
	offset := 0

	if q != "" {
		parsedFilters, err := ParseNaturalLanguageQuery(q)
		if err != nil {
			utils.WriteJson(w, http.StatusUnprocessableEntity, model.ErrorResponse{
				Status:  "error",
				Message: "Unable to interpret query",
			})
			return
		}
		filters = parsedFilters
	}

	if gender := strings.ToLower(r.URL.Query().Get("gender")); gender != "" {
		filters.Gender = &gender
	}

	if countryId := strings.ToLower(r.URL.Query().Get("country_id")); countryId != "" {
		filters.CountryID = &countryId
	}

	if ageGroup := strings.ToLower(r.URL.Query().Get("age_group")); ageGroup != "" {
		filters.AgeGroup = &ageGroup
	}

	if minAge := r.URL.Query().Get("min_age"); minAge != "" {
		val, err := strconv.Atoi(minAge)
		if err != nil {
			utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
				Status:  "error",
				Message: "min_age should be a number",
			})
			return
		}
		filters.MinAge = &val
	}

	if maxAge := r.URL.Query().Get("max_age"); maxAge != "" {
		val, err := strconv.Atoi(maxAge)
		if err != nil {
			utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
				Status:  "error",
				Message: "max_age should be a number",
			})
			return
		}
		filters.MaxAge = &val
	}

	if minGenderProbability := r.URL.Query().Get("min_gender_probability"); minGenderProbability != "" {
		val, err := strconv.Atoi(minGenderProbability)
		if err != nil {
			utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
				Status:  "error",
				Message: "min_gender_probability should be a number",
			})
			return
		}
		filters.MinGenderProbability = &val
	}

	if minCountryProbability := r.URL.Query().Get("min_country_probability"); minCountryProbability != "" {
		val, err := strconv.Atoi(minCountryProbability)
		if err != nil {
			utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
				Status:  "error",
				Message: "min_country_probability should be a number",
			})
			return
		}
		filters.MinCountryProbability = &val
	}

	if sortBy := strings.ToLower(r.URL.Query().Get("sort_by")); sortBy != "" {
		filters.SortBy = &sortBy
	}

	if order := strings.ToLower(r.URL.Query().Get("order")); order != "" {
		filters.Order = &order
	}

	users, _, err := database.QueryAllUsers(filters, limit, offset)
	if err != nil {
		utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "Unable to export profiles",
		})
		return
	}

	filename := fmt.Sprintf(
		"profiles_%s.csv",
		time.Now().Format("20060102_150405"),
	)

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set(
		"Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"`, filename),
	)

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{
		"id",
		"name",
		"gender",
		"gender_probability",
		"age",
		"country_id",
		"country_probability",
		"created_at",
		"updated_at",
	})

	for _, user := range users {
		writer.Write([]string{
			user.ID,
			user.Name,
			user.Gender,
			strconv.FormatFloat(user.GenderProbability, 'f', 2, 64),
			strconv.Itoa(user.Age),
			user.CountryID,
			strconv.FormatFloat(user.CountryProbability, 'f', 2, 64),
			user.CreatedAt.Format(time.RFC3339),
		})
	}
}
