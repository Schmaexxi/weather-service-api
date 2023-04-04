package service

import (
	"context"

	"github.com/katiamach/weather-service-api/internal/model"
)

// WeatherService provides weather service functionality.
type WeatherService struct{}

// New creates new WeatherService.
func New() *WeatherService {
	return &WeatherService{}
}

// GetWindInfo implements wind info getting.
func (ws *WeatherService) GetWindInfo(ctx context.Context, req *model.WindRequest) error {
	return nil
}
