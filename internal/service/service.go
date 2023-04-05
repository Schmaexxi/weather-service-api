package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/katiamach/weather-service-api/internal/model"
	"github.com/katiamach/weather-service-api/internal/repository"
)

// Repository provides necessary repo methods.
type Repository interface {
	InsertAnnualStatistics(ctx context.Context, measurements []*model.WindStatistics) error
	GetStationID(ctx context.Context, stationName string) (string, error)
	InsertStationsInfo(ctx context.Context, stationsInfo []*model.Station) error
	GetStationWindStatistics(ctx context.Context, stationName string, years int) ([]*model.WindStatistics, error)
}

// WeatherService provides weather service functionality.
type WeatherService struct {
	repo Repository
}

// New creates new WeatherService.
func New(repo Repository) *WeatherService {
	return &WeatherService{
		repo: repo,
	}
}

// GetWindStatistics implements retrieving year wind statistics.
func (ws *WeatherService) GetWindStatistics(ctx context.Context, req *model.WindRequest) ([]*model.WindStatistics, error) {
	stationName, err := getNearestStation(req.City)
	if err != nil {
		return nil, err
	}

	stationID, err := ws.repo.GetStationID(ctx, stationName)
	if err == repository.ErrNoSuchStation {
		err := ws.loadStationsInfo(ctx)
		if err != nil {
			return nil, err
		}
	}
	if err != nil && err != repository.ErrNoSuchStation {
		return nil, fmt.Errorf("failed to get station id: %w", err)
	}

	stats, err := ws.repo.GetStationWindStatistics(ctx, stationName, req.Years)
	if err == repository.ErrNoWindDataForStation {
		err := ws.loadStationWindStatistics(ctx, stationID, stationName)
		if err != nil {
			return nil, err
		}

		stats, err = ws.repo.GetStationWindStatistics(ctx, stationName, req.Years)
		if err != nil {
			return nil, fmt.Errorf("failed to get station wind statistics: %w", err)
		}
	}
	if err != nil && err != repository.ErrNoWindDataForStation {
		return nil, fmt.Errorf("failed to get station wind data: %w", err)
	}

	return stats, nil
}

// GetNearestStation finds nearest weather station for the given city.
func getNearestStation(city string) (string, error) {
	long, lat, err := getCityCoordinates(city)
	if err != nil {
		return "", fmt.Errorf("failed to get city coordinates: %w", err)
	}

	stationName, err := getNearestStationName(long, lat)
	if err != nil {
		return "", fmt.Errorf("failed to get coordinates: %w", err)
	}

	return stationName, nil
}

func getCityCoordinates(city string) (float64, float64, error) {
	params := fmt.Sprintf("?access_key=%s&query=%s", os.Getenv("GEO_API_ACCESS_KEY"), city)
	resp, err := http.Get(os.Getenv("GEO_API_URL") + params)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get coordinates for the given city: %w", err)
	}
	defer resp.Body.Close()

	type response struct {
		Data []struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}
	}

	var res response
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(res.Data) < 1 || res.Data[0].Longitude == 0 || res.Data[0].Latitude == 0 {
		return 0, 0, errors.New("coordinates not found, check city name")
	}

	return res.Data[0].Longitude, res.Data[0].Latitude, nil
}

func getNearestStationName(long, lat float64) (string, error) {
	params := fmt.Sprintf("?lon=%f&lat=%f&limit=1", long, lat)
	req, err := http.NewRequest("GET", os.Getenv("NEARBY_STATIONS_API_URL")+params, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create nearest station request: %w", err)
	}

	req.Header.Set("x-rapidapi-host", "meteostat.p.rapida1i.com")
	req.Header.Set("x-rapidapi-key", os.Getenv("RAPID_API_KEY"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get nearest station data from source: %w", err)
	}
	defer resp.Body.Close()

	type response struct {
		Data []struct {
			Name map[string]string `json:"name"`
		}
	}

	var res response
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(res.Data) < 1 {
		return "", fmt.Errorf("failed to find nearest station: %w", err)
	}

	name, ok := res.Data[0].Name["en"]
	if !ok {
		return "", errors.New("there is no available english name")
	}

	return name, nil
}
