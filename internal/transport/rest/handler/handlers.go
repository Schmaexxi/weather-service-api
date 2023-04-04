package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/katiamach/weather-service-api/internal/logger"
	"github.com/katiamach/weather-service-api/internal/model"
)

//go:generate mockgen -source=handlers.go -destination=mock/mock.go WeatherService

// WeatherService provides weather service methods.
type WeatherService interface {
	GetWindInfo(ctx context.Context, req *model.WindRequest) error
}

// WeatherServer is a server for weather info processing.
type WeatherServer struct {
	service WeatherService
}

// NewWeatherServer creates new WeatherServer.
func NewWeatherServer(service WeatherService) *WeatherServer {
	return &WeatherServer{service}
}

// GetWindInfoHandler handles GetWindInfo request.
func (s *WeatherServer) GetWindInfoHandler(w http.ResponseWriter, r *http.Request) {
	// validation
	err := s.service.GetWindInfo(r.Context(), &model.WindRequest{})
	if err != nil {
		logger.Error(fmt.Errorf("failed to get wind info: %v", err))
		respondErr(w, http.StatusInternalServerError, err)
		return
	}
	respond(w, http.StatusOK, http.NoBody)
}
