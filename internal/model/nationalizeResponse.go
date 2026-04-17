package model

type NationalizeResponse struct {
	Count   uint32 `json:"count"`
	Name    string `json:"name"`
	Country []Country
}

type Country struct {
	CountryId   string  `json:"country_id"`
	Probability float32 `json:"probability"`
}