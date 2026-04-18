package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Taterbro/backendStageZero/internal/model"
)

func GetGender(name string) (*model.GenderizeResponse, error) {
	apiURL := fmt.Sprintf("https://api.genderize.io?name=%s", name)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result model.GenderizeResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	fmt.Printf("genderize data: %v", result)


	return &result, nil
}