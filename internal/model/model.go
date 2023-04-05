package model

import "time"

// WindRequest contains wind request parameters.
type WindRequest struct {
	City string `json:"city"`
	Year int    `json:"year"`
}

// WindMeasurment contains hourly file wind info.
type WindMeasurment struct {
	EndDate time.Time
	Speed   float64
}

// AverageYearWindSpeed contains every measured year with corresponding average speed.
type AverageYearWindSpeed struct {
	StationName string  `bson:"stationName,omitempty"`
	Year        int     `bson:"year,omitempty"`
	Speed       float64 `bson:"speed,omitempty"`
}

// Station contains weather station info.
type Station struct {
	ID   string `bson:"id,omitempty"`
	Name string `bson:"name,omitempty"`
}
