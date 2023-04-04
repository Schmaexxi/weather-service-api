package model

// WindRequest contains wind request parameters.
type WindRequest struct {
	City string `json:"city"`
	Year int    `json:"year"`
}
