package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Taterbro/backendStageZero/internal/model"
)

func GetGender(name string) (*model.GenderizeResponse, error) {
	baseURL := "https://api.genderize.io"
	params := url.Values{}
	params.Set("name", name)

	apiURL := baseURL + "?" + params.Encode()
	var apiError model.ApiError
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	// body, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	return nil, err
	// }
	//log.Println("genderize response body: ", string(body))

	if resp.Status != "200 OK" {
		json.NewDecoder(resp.Body).Decode(&apiError)
		return nil, fmt.Errorf("genderize failed: %s", apiError.Error)
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
