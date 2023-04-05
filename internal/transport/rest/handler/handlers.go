package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/katiamach/weather-service-api/internal/logger"
	"github.com/katiamach/weather-service-api/internal/model"
)

//go:generate mockgen -source=handlers.go -destination=mock/mock.go WeatherService

// WeatherService provides weather service methods.
type WeatherService interface {
	GetWindStatistics(ctx context.Context, req *model.WindRequest) ([]*model.WindStatistics, error)
}

// WeatherServer is a server for weather data processing.
type WeatherServer struct {
	service WeatherService
}

// NewWeatherServer creates new WeatherServer.
func NewWeatherServer(service WeatherService) *WeatherServer {
	return &WeatherServer{service}
}

// GetWindStatisticsHandler handles GetWindStatistics request.
func (s *WeatherServer) GetWindStatisticsHandler(w http.ResponseWriter, r *http.Request) {
	windReq, err := validateQueryParams(r.URL.Query())
	if err != nil {
		logger.Error(err)
		respondErr(w, http.StatusBadRequest, err)
		return
	}

	statistics, err := s.service.GetWindStatistics(r.Context(), windReq)
	if err != nil {
		logger.Error(fmt.Errorf("failed to get wind statistics: %v", err))
		respondErr(w, http.StatusInternalServerError, err)
		return
	}

	respond(w, http.StatusOK, statistics)
}

func validateQueryParams(params url.Values) (*model.WindRequest, error) {
	city := params.Get("city")
	if city == "" {
		return nil, errors.New("city parameter not provided in query")
	}

	yearsStr := params.Get("years")
	if yearsStr == "" {
		return nil, errors.New("years parameter not provided in query")
	}

	years, err := strconv.Atoi(yearsStr)
	if err != nil {
		return nil, fmt.Errorf("years parameter is not a number: %w", err)
	}

	return &model.WindRequest{City: city, Years: years}, nil
}
