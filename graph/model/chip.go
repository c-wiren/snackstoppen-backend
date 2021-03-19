package model

type Chip struct {
	ID          int     `json:"id"`
	Brand       *Brand  `json:"brand"`
	Category    string  `json:"category"`
	Image       *string `json:"image"`
	Ingredients *string `json:"ingredients"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Subcategory *string `json:"subcategory"`
}
