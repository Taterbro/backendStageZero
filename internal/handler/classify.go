package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Taterbro/backendStageZero/internal/model"
	"github.com/Taterbro/backendStageZero/internal/service"
	"github.com/Taterbro/backendStageZero/internal/utils"
)

func ClassifyHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")

	if name == "" {
		utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "name parameter is required",
		})
		return
	}

	if _, err := strconv.Atoi(name); err == nil {
		utils.WriteJson(w, http.StatusUnprocessableEntity, model.ErrorResponse{
			Status:  "error",
			Message: "name should not be a number",
		})
		return
	}

	result, err := service.GetGender(name)
	if err != nil {
		service.HandleUpstreamError(w)
		return
	}

	if result.Count == 0 || result.Gender == "" {
		utils.WriteJson(w, http.StatusNotFound, model.ErrorResponse{
			Status:  "error",
			Message: "No prediction available",
		})
		return
	}

	isConfident := result.Probability >= 0.7 && result.Count >= 100

	response := model.ResponseData{
		Name:        name,
		Gender:      result.Gender,
		Probability: result.Probability,
		SampleSize:  result.Count,
		IsConfident: isConfident,
		ProcessedAt: time.Now().UTC().Format(time.RFC3339),
	}

	utils.WriteJson(w, http.StatusOK, model.SuccessResponse{
		Status: "success",
		Data:   response,
	})
}

