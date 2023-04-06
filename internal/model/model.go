package model

import "time"

// WindRequest contains wind request parameters.
type WindRequest struct {
	City  string `json:"city"`
	Years int    `json:"years"`
}

// HourlyStatistics contains hourly file wind statistics.
type HourlyStatistics struct {
	EndDate time.Time
	Speed   float64
}

// WindStatistics contains annual statistics of wind speed.
type WindStatistics struct {
	StationName string  `bson:"stationName,omitempty"`
	Year        int     `bson:"year,omitempty"`
	Speed       float64 `bson:"speed,omitempty"`
}

// Station contains weather station info.
type Station struct {
	ID        string  `bson:"id,omitempty"`
	Name      string  `bson:"name,omitempty"`
	Latitude  float64 `bson:"latitude,omitempty"`
	Longitude float64 `bson:"longitude,omitempty"`
}
