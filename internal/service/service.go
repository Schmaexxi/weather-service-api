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
	"github.com/umahmood/haversine"
)

var (
	ErrNoDataInThisPeriod = errors.New("unfortunately, there is no data available for the nearest weather station for this period")
	ErrCityNotFound       = errors.New("city not found, please, check city name")
)

// Repository provides necessary repo methods.
type Repository interface {
	InsertAnnualStatistics(ctx context.Context, measurements []*model.WindStatistics) error
	GetStationID(ctx context.Context, stationName string) (string, error)
	InsertStationsInfo(ctx context.Context, stationsInfo []*model.Station) error
	GetStationWindStatistics(ctx context.Context, stationName string, years int) ([]*model.WindStatistics, error)
	GetStationsCoordinates(ctx context.Context) ([]*model.Station, error)
	CheckIfStatisticsExists(ctx context.Context) (bool, error)
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
	stationName, err := ws.getNearestStation(ctx, req.City)
	if err == ErrCityNotFound {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	stationID, err := ws.repo.GetStationID(ctx, stationName)
	if err != nil {
		return nil, fmt.Errorf("failed to get station id: %w", err)
	}

	statsExists, err := ws.repo.CheckIfStatisticsExists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check if statistics exists: %w", err)
	}

	if !statsExists {
		err := ws.loadStationWindStatistics(ctx, stationID, stationName)
		if err != nil {
			return nil, err
		}
	}

	stats, err := ws.repo.GetStationWindStatistics(ctx, stationName, req.Years)
	if err == repository.ErrNoWindDataForStation {
		return nil, ErrNoDataInThisPeriod
	}
	if err != nil && err != repository.ErrNoWindDataForStation {
		return nil, fmt.Errorf("failed to get station wind data: %w", err)
	}

	return stats, nil
}

// GetNearestStation finds nearest weather station for the given city.
func (ws *WeatherService) getNearestStation(ctx context.Context, city string) (string, error) {
	lat, lon, err := getCityCoordinates(city)
	if err != nil {
		return "", fmt.Errorf("failed to get city coordinates: %w", err)
	}

	stationName, err := ws.getNearestStationName(ctx, lat, lon)
	if err != nil {
		return "", fmt.Errorf("failed to get city coordinates: %w", err)
	}

	return stationName, nil
}

// GetCityCoordinates uses an open API to get city coordinates.
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
		return 0, 0, ErrCityNotFound
	}

	return res.Data[0].Latitude, res.Data[0].Longitude, nil
}

// GetNearestStationName finds nearest weather station in db for the given city.
func (ws *WeatherService) getNearestStationName(ctx context.Context, lat, lon float64) (string, error) {
	cityCoords := haversine.Coord{Lat: lat, Lon: lon}

	stations, err := ws.repo.GetStationsCoordinates(ctx)
	if err == repository.ErrNoStations {
		err := ws.loadStationsInfo(ctx)
		if err != nil {
			return "", err
		}

		stations, err = ws.repo.GetStationsCoordinates(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get station coordinates: %w", err)
		}
	}
	if err != nil && err != repository.ErrNoStations {
		return "", fmt.Errorf("failed to get station coordinates: %w", err)
	}

	nearestStation := findNearestStation(cityCoords, stations)

	return nearestStation.Name, nil
}

func findNearestStation(cityCoords haversine.Coord, stations []*model.Station) *model.Station {
	var minDistance float64
	minIndex := len(stations) + 1

	for i, st := range stations {
		stCoords := haversine.Coord{Lat: st.Latitude, Lon: st.Longitude}

		_, km := haversine.Distance(cityCoords, stCoords)
		if i == 0 || km < minDistance {
			minDistance = km
			minIndex = i
		}
	}

	return stations[minIndex]
}
