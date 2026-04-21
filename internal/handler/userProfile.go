package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"sync"
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
	var req Request

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "Invalid request body",
		})
		return
	}

	if req.Name == "" {
		utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "Name field is required",
		})
		return
	}
	name := strings.ToLower(req.Name)
	if _, err := strconv.Atoi(name); err == nil {
		utils.WriteJson(w, http.StatusUnprocessableEntity, model.ErrorResponse{
			Status:  "error",
			Message: "name should not be a number",
		})
		return
	}

	user, ok := database.UserStore.ByName[name]
	if ok {
		utils.WriteJson(w, http.StatusOK, model.UserSuccessResponse{
			Status:  "success",
			Message: "Profile already exists",
			Data:    *user,
		})
		return
	}

	var (
		agifyData       *model.AgifyResponse
		genderData      *model.GenderizeResponse
		nationalityData *model.NationalizeResponse
	)

	g, _ := errgroup.WithContext(context.Background())

	g.Go(func() error {
		var err error
		agifyData, err = service.GetAge(name)
		return err
	})

	g.Go(func() error {
		var err error
		genderData, err = service.GetGender(name)
		return err
	})

	g.Go(func() error {
		var err error
		nationalityData, err = service.GetNation(name)
		return err
	})

	if err := g.Wait(); err != nil {
		utils.WriteJson(w, http.StatusBadGateway, model.ErrorResponse{
			Status:  "error",
			Message: "Failed to fetch external data",
		})
		return
	}

	if agifyData.Age == 0 {
		utils.WriteJson(w, http.StatusBadGateway, model.ErrorResponse{
			Status:  "error",
			Message: "Agify returned an invalid response.",
		})
		return
	}
	if genderData.Gender == "" {
		utils.WriteJson(w, http.StatusBadGateway, model.ErrorResponse{
			Status:  "error",
			Message: "Genderize returned an invalid response.",
		})
		return
	}
	if nationalityData.Country == nil {
		utils.WriteJson(w, http.StatusBadGateway, model.ErrorResponse{
			Status:  "error",
			Message: "Nationalize returned an invalid response.",
		})
	}
	ageGroup := "child"
	if agifyData.Age > 18 {
		ageGroup = "adult"
	}

	dummyUser := database.User{
		ID:                 uuid.New().String(),
		Name:               name,
		Gender:             genderData.Gender,
		GenderProbability:  float64(genderData.Probability),
		Age:                agifyData.Age,
		AgeGroup:           ageGroup,
		CountryID:          nationalityData.Country[0].CountryId,
		CountryProbability: float64(nationalityData.Country[0].Probability),
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
	}
	a := sync.RWMutex{}

	a.RLock()
	database.UserStore.AddUser(&dummyUser)
	a.RUnlock()

	fmt.Printf("le database fr fr: %v\n", database.UserStore)

	utils.WriteJson(w, http.StatusCreated, model.SuccessResponse{
		Status: "success",
		Data:   dummyUser,
	})
}

func FindUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	user := database.UserStore.GetById(id)
	if user == nil {
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
	var filters database.SearchFilter
	page := 1
	limit := 10
	gender := strings.ToLower(r.URL.Query().Get("gender"))
	countryId := strings.ToLower(r.URL.Query().Get("country_id"))
	ageGroup := strings.ToLower(r.URL.Query().Get("age_group"))
	minAge := r.URL.Query().Get("min_age")
	maxAge := r.URL.Query().Get("max_age")
	minGenderProbability := r.URL.Query().Get("min_gender_probability")
	minCountryProbability := r.URL.Query().Get("min_country_probability")
	sortBy := strings.ToLower(r.URL.Query().Get("sort_by"))
	order := strings.ToLower(r.URL.Query().Get("order"))
	qpage := r.URL.Query().Get("page")
	qLimit := r.URL.Query().Get("limit")

	if minAge != "" {
		minAgeInt, err := strconv.Atoi(minAge)
		if err != nil {
			utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
				Status:  "error",
				Message: "Invalid query parameter; min_age should be a number",
			})
			return
		} else {
			filters.MinAge = &minAgeInt
		}
	}

	if maxAge != "" {
		maxAgeInt, err := strconv.Atoi(maxAge)
		if err != nil {
			utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
				Status:  "error",
				Message: "Invalid query parameter; max_age should be a number",
			})
			return
		} else {
			filters.MaxAge = &maxAgeInt
		}
	}

	if minGenderProbability != "" {
		minGenderProbabilityInt, err := strconv.Atoi(minGenderProbability)
		if err != nil {
			utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
				Status:  "error",
				Message: "Invalid query parameter; min_gender_probability should be a number",
			})
			return
		} else {
			filters.MinGenderProbability = &minGenderProbabilityInt
		}
	}

	if qpage != "" {
		qpageInt, err := strconv.Atoi(qpage)
		if err != nil {
			utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
				Status:  "error",
				Message: "Invalid query parameter; page should be a number",
			})
			return
		} else {
			page = qpageInt
		}
	}

	if qLimit != "" {
		qLimitInt, err := strconv.Atoi(qLimit)
		if err != nil {
			utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
				Status:  "error",
				Message: "Invalid query parameter; limit should be a number",
			})
			return
		}
		if qLimitInt > 50 {
			limit = 50
		} else {
			limit = qLimitInt
		}
	}

	if minCountryProbability != "" {
		minCountryProbabilityInt, err := strconv.Atoi(minCountryProbability)
		if err != nil {
			utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
				Status:  "error",
				Message: "Invalid query parameter; min_country_probability should be a number",
			})
			return
		} else {
			filters.MinCountryProbability = &minCountryProbabilityInt
		}
	}
	if gender != "" {
		filters.Gender = &gender
	}
	if countryId != "" {
		filters.CountryID = &countryId
	}
	if ageGroup != "" {
		filters.AgeGroup = &ageGroup
	}
	if sortBy != "" {
		filters.SortBy = &sortBy
	}
	if order != "" {
		filters.Order = &order
	}
	offset := (page - 1) * limit

	users, err := database.QueryAllUsers(filters, limit, offset)
	if err == nil {

	}

	utils.WriteJson(w, http.StatusOK, model.GetUserSuccessResponse{
		Status: "success",
		Page:   page,
		Limit:  limit,
		Total:  len(users),
		Data:   users,
	})
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
