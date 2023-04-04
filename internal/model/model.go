package model

import "time"

// WindRequest contains wind request parameters.
type WindRequest struct {
	City string `json:"city"`
	Year int    `json:"year"`
}

// WindMeasurment represent hourly file wind info.
type WindMeasurment struct {
	EndDate time.Time
	Speed   float64
}

// AverageYearWindSpeed represent every measured year with corresponding average speed.
type AverageYearWindSpeed struct {
	Year  int     `bson:"year,omitempty"`
	Speed float64 `bson:"speed,omitempty"`
}
