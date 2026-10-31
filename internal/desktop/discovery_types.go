package desktop

// CategoryCardDTO describes one of Vocabulary.com's 9 root curated unit categories.
type CategoryCardDTO struct {
	UnitID      string `json:"unit_id"`
	Tag         string `json:"tag"`
	Title       string `json:"title"`
	Description string `json:"description"`
}
