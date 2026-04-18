package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Taterbro/backendStageZero/internal/model"
)

func GetAge(name string) (*model.AgifyResponse, error) {
	apiURL := fmt.Sprintf("https://api.agify.io?name=%s", name)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result model.AgifyResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	fmt.Printf("agify data: %v", result)
	return &result, nil
}