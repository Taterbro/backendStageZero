package handler

import (
	"net/http"

	"github.com/Taterbro/backendStageZero/internal/database"
	"github.com/Taterbro/backendStageZero/internal/utils"
)

func DevQuery(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	data, err := database.DevQuery(q)
	if err != nil {
		utils.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
			"data": err,
		})
	}

	utils.WriteJson(w, http.StatusOK, map[string]interface{}{
		"data": data,
	})
}
