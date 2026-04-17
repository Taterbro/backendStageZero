package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Taterbro/backendStageZero/internal/model"
)

func GetAge(name string) (*model.AgifyResponse, error) {
	apiURL := fmt.Sprintf("https://api.agify.io?name=%s", name)

	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Printf("error from get age: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var result model.AgifyResponse
	body, _ := io.ReadAll(resp.Body)
	err = json.NewDecoder(resp.Body).Decode(&result)
	fmt.Printf("result from agify is: %v",result)
	fmt.Println("raw agify body is: ",string(body))
	if err != nil {
		return nil, err
	}

	return &result, nil
}