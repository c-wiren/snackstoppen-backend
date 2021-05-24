package model

type Brand struct {
	ID    string  `json:"id"`
	Count int     `json:"count"`
	Image *string `json:"image"`
	Name  string  `json:"name"`
}
