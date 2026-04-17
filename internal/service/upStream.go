package service

import (
	"fmt"
	"net/http"

	"github.com/Taterbro/backendStageZero/internal/model"
	"github.com/Taterbro/backendStageZero/internal/utils"
)

func HandleUpstreamError(w http.ResponseWriter) {
	utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
		Status:  "error",
		Message: "Upstream error",
	})
}

func HandleEdgeCases(w http.ResponseWriter, apiName string){
	utils.WriteJson(w, http.StatusBadGateway, model.ErrorResponse{
		Status: "error",
		Message: fmt.Sprintf("%s returned an invalid response.", apiName),
	})
}