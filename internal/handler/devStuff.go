package handler

import (
	"log"
	"net/http"

	"github.com/Taterbro/backendStageZero/internal/database"
	"github.com/Taterbro/backendStageZero/internal/model"
	"github.com/Taterbro/backendStageZero/internal/utils"
)

func DevQuery(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	data, err := database.DevQuery(q)
	if err != nil {
		log.Println("error handling query: ", err)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "internal error",
		})
		return
	}

	utils.WriteJson(w, http.StatusOK, map[string]any{
		"data": data,
	})
}
