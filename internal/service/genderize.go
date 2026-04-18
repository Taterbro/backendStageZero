package service

import (
	"encoding/json"
	"fmt"
	"io"
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
	body, _ := io.ReadAll(resp.Body)
	err = json.NewDecoder(resp.Body).Decode(&result)
	fmt.Printf("result from genderize is: %v",result)
	fmt.Println("raw genderize body is: ",string(body))
	if err != nil {
		return nil, err
	}

	return &result, nil
}