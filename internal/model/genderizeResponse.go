package model

type GenderizeResponse struct {
	Count       uint32  `json:"count"`
	Name        string  `json:"name"`
	Gender      string  `json:"gender"`
	Probability float32 `json:"probability"`
}