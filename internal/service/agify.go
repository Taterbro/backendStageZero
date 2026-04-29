package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Taterbro/backendStageZero/internal/model"
)

func GetAge(name string) (*model.AgifyResponse, error) {
	baseURL := "https://api.agify.io"
	params := url.Values{}
	params.Set("name", name)

	apiURL := baseURL + "?" + params.Encode()
	var apiError model.ApiError
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	if resp.Status != "200 OK" {
		json.NewDecoder(resp.Body).Decode(&apiError)
		return nil, fmt.Errorf("agify failed: %s", apiError.Error)
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
