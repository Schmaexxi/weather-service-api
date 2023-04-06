package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"

	"github.com/katiamach/weather-service-api/backend/internal/model"
	"github.com/katiamach/weather-service-api/backend/internal/repository"
	"github.com/umahmood/haversine"
)

var (
	ErrNoStatisticsInThisPeriod = errors.New("unfortunately, there is no statistics available for the nearest weather station for this period")
	ErrCityNotFound             = errors.New("city not found, please, check city name")
)

// Repository provides necessary repo methods.
type Repository interface {
	InsertAnnualStatistics(ctx context.Context, measurements []*model.WindStatistics) error
	GetStationID(ctx context.Context, stationName string) (string, error)
	InsertStationsInfo(ctx context.Context, stationsInfo []*model.Station) error
	GetStationWindStatistics(ctx context.Context, stationName string, years int) ([]*model.WindStatistics, error)
	GetStationsCoordinates(ctx context.Context) ([]*model.Station, error)
	CheckIfStatisticsExists(ctx context.Context, stationName string) (bool, error)
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
	possibleStations, err := ws.getPossibleNearestStations(ctx, req)
	if err == ErrCityNotFound {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	var stationName string

	for _, ps := range possibleStations {
		stationID, err := ws.repo.GetStationID(ctx, ps.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get station id: %w", err)
		}

		statsExists, err := ws.repo.CheckIfStatisticsExists(ctx, ps.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to check if statistics exists: %w", err)
		}

		if statsExists {
			stationName = ps.Name
			break
		}

		err = ws.loadStationWindStatistics(ctx, stationID, ps.Name)
		if errors.Is(err, errStatsFileNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}

		stationName = ps.Name
		break
	}

	if stationName == "" {
		return nil, ErrNoStatisticsInThisPeriod
	}

	stats, err := ws.repo.GetStationWindStatistics(ctx, stationName, req.Years)
	if err != nil && err != repository.ErrNoWindDataForStation {
		return nil, fmt.Errorf("failed to get station wind data: %w", err)
	}

	return stats, nil
}

// GetPossibleNearestStations finds possible nearest weather station for the given city.
func (ws *WeatherService) getPossibleNearestStations(ctx context.Context, req *model.WindRequest) ([]*model.Station, error) {
	lat, lon, err := getCityCoordinates(req.City)
	if err != nil {
		return nil, fmt.Errorf("failed to get city coordinates: %w", err)
	}

	cityCoords := haversine.Coord{Lat: lat, Lon: lon}

	stations, err := ws.repo.GetStationsCoordinates(ctx)
	if err == repository.ErrNoStations {
		err := ws.loadStationsInfo(ctx)
		if err != nil {
			return nil, err
		}

		stations, err = ws.repo.GetStationsCoordinates(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get station coordinates: %w", err)
		}
	}
	if err != nil && err != repository.ErrNoStations {
		return nil, fmt.Errorf("failed to get station coordinates: %w", err)
	}

	possibleStations := findPossibleNearestStations(cityCoords, stations, req.Years)

	return possibleStations, nil
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

func findPossibleNearestStations(cityCoords haversine.Coord, stations []*model.Station, years int) []*model.Station {
	stationByDistance := make(map[float64]*model.Station, len(stations))

	for _, st := range stations {
		stCoords := haversine.Coord{Lat: st.Latitude, Lon: st.Longitude}
		_, kmDistance := haversine.Distance(cityCoords, stCoords)

		stationByDistance[kmDistance] = st
	}

	distances := make([]float64, 0, len(stationByDistance))
	for d := range stationByDistance {
		distances = append(distances, d)
	}

	// sort distances from min to max
	sort.Float64s(distances)

	possibleStations := make([]*model.Station, 0, len(stations))

	// range starting with the station with min distance and so on
	for _, d := range distances {
		// check if the station contains data in given period (at least 1 statistic)
		if stationByDistance[d].EndDate.Year() > repository.LastMeasuredYear-years {
			possibleStations = append(possibleStations, stationByDistance[d])
		}
	}

	return possibleStations
}
