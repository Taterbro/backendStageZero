package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"golang.org/x/sync/errgroup"

	"github.com/Taterbro/backendStageZero/internal/database"
	"github.com/Taterbro/backendStageZero/internal/model"
	"github.com/Taterbro/backendStageZero/internal/service"
	"github.com/Taterbro/backendStageZero/internal/utils"
)

type Request struct{
	Name string `json:"name"`
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
    name := req.Name
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
        agifyData      *model.AgifyResponse
        genderData     *model.GenderizeResponse
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

    if agifyData.Age==0{
		utils.WriteJson(w, http.StatusBadGateway, model.ErrorResponse{
		Status: "error",
		Message: "Agify returned an invalid response.",
	})
	return
	}
	if genderData.Gender == ""{
		utils.WriteJson(w, http.StatusBadGateway, model.ErrorResponse{
		Status: "error",
		Message: "Genderize returned an invalid response.",
	})
	return
	}
	if nationalityData.Country==nil{
		utils.WriteJson(w, http.StatusBadGateway, model.ErrorResponse{
		Status: "error",
		Message: "Nationalize returned an invalid response.",
	})
	}

	dummyUser := database.User{
		ID:                 1,
		Name:               name,
		Gender:             genderData.Gender,
		GenderProbability:  float64(genderData.Probability),
		SampleSize:         1234,
		Age:               agifyData.Age,
		AgeGroup:           "adult",
		CountryID:          "EEHEHEHE",
		CountryProbability: 0.85,
		CreatedAt:          "2026-04-01T12:00:00Z",
	}

	utils.WriteJson(w, http.StatusOK, model.UserSuccessResponse{
            Status:  "success",
            Data: dummyUser,
        })
}