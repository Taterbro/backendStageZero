package model

type AgifyResponse struct {
	Count uint32 `json:"count"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
}