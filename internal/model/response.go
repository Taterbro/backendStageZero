package model

import "github.com/Taterbro/backendStageZero/internal/database"

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
type UserSuccessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    database.User `json:"data"`
}
type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}