package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Taterbro/backendStageZero/internal/model"
)

func GetNation(name string) (*model.NationalizeResponse, error) {
	apiURL := fmt.Sprintf("https://api.nationalize.io?name=%s", name)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result model.NationalizeResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	

	return &result, nil
}