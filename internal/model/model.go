package model

import "time"

// WindRequest contains wind request parameters.
type WindRequest struct {
	City string `json:"city"`
	Year int    `json:"year"`
}

type WindMeasurment struct {
	StationID string    `bson:"stationID,omitempty"`
	EndDate   time.Time `bson:"endDate,omitempty"`
	Speed     float64   `bson:"speed,omitempty"`
}
