package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)
type GenderizeResponse struct {
	Count       uint32 `json:"count"`
	Name        string `json:"name"`
	Gender      string `json:"gender"`
	Probability float32  `json:"probability"`
}
type ResponseData struct {
	Name        string  `json:"name"`
	Gender      string  `json:"gender"`
	Probability float32 `json:"probability"`
	SampleSize  uint32  `json:"sample_size"`
	IsConfident bool    `json:"is_confident"`
	ProcessedAt string  `json:"processed_at"`
}
type SuccessResponse struct {
	Status string       `json:"status"`
	Data   ResponseData `json:"data"`
}
type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}


func writeJson(w http.ResponseWriter, status int, payload any){
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func requestHandler(w http.ResponseWriter,r *http.Request){
	name := r.URL.Query().Get("name")
	if name == ""{
		writeJson(w,http.StatusBadRequest,ErrorResponse{
			Status:"error", Message: "name parameter is required",
		})
		return
	}
	_, err := strconv.Atoi(name)
	if err == nil{
		writeJson(w,http.StatusUnprocessableEntity,ErrorResponse{
			Status:"error", Message: "name parameter should not be a number",
		})
		return
	}

	apiUrl := fmt.Sprintf("https://api.genderize.io?name=%s", name)
	resp, err := http.Get(apiUrl)
	if err != nil{
		writeJson(w,http.StatusInternalServerError,ErrorResponse{
			Status:"error", Message: "Upstream error from genderize.",
		})
		return
	}
	defer resp.Body.Close()

	var result GenderizeResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil{
		fmt.Println("Failed to decode value from genderize!")
		writeJson(w,http.StatusInternalServerError,ErrorResponse{
			Status:"error", Message: "Internal Server Error",
		})
		return
	}

	if result.Count == 0 || result.Gender==""{
		writeJson(w,http.StatusInternalServerError,ErrorResponse{
			Status:"error", Message: "No prediction available for the provided name",
		})
		return
	}

	isConfident := false 
	now := time.Now().UTC()
	processedAt := now.Format(time.RFC3339)
	if result.Probability >= 0.7 && result.Count >= 100{ isConfident=true}

	responseData := ResponseData{
		Name: name,
		Gender: result.Gender,
		Probability: result.Probability,
		SampleSize: result.Count,
		IsConfident: isConfident,
		ProcessedAt: processedAt,
	} 
	
	writeJson(w,http.StatusOK,SuccessResponse{
		Status: "success",
		Data: responseData,
	})
}

func main(){
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/classify", requestHandler)
	server := &http.Server{
		Handler: mux,
		Addr: ":8080",
	}

	log.Println("Server running on http://localhost:8080")
	log.Fatal(server.ListenAndServe())
}