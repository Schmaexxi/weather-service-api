package service

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/katiamach/weather-service-api/internal/logger"
	"github.com/katiamach/weather-service-api/internal/model"
	"github.com/katiamach/weather-service-api/internal/repository"
	"golang.org/x/net/html"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

var (
	dataFileNameRegExp    = `(stundenwerte_FF_+)`
	productFileNameRegExp = `(produkt+)`
)

const timeLayout = "2006010215"

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

// LoadStationsInfo gets stations information from source and saves it in a database.
func (ws *WeatherService) loadStationsInfo(ctx context.Context) error {
	stationsInfo, err := getStationsInfo()
	if err != nil {
		return fmt.Errorf("failed to get station info: %w", err)
	}

	err = ws.repo.InsertStationsInfo(ctx, stationsInfo)
	if err != nil {
		return fmt.Errorf("failed to insert stations info: %w", err)
	}

	return nil
}

// LoadStationWindStatistics creates stations wind annual statistics and saves it in a database.
func (ws *WeatherService) loadStationWindStatistics(ctx context.Context, stationID, stationName string) error {
	data, err := getStationHistoricalData()
	if err != nil {
		return fmt.Errorf("failed to get station historical data: %w", err)
	}

	yearMeasurements, err := ws.getAnnualStatistics(stationID, stationName, data)
	if err != nil {
		return fmt.Errorf("failed to process info: %w", err)
	}

	err = ws.repo.InsertAnnualStatistics(ctx, yearMeasurements)
	if err != nil {
		return fmt.Errorf("failed to insert year measurements: %w", err)
	}

	return nil
}

func getStationHistoricalData() (string, error) {
	resp, err := http.Get(os.Getenv("HOURLY_WIND_HISTORICAL_DATA_URL"))
	if err != nil {
		return "", fmt.Errorf("failed to get wind data from source: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

func (ws *WeatherService) getAnnualStatistics(stationID, stationName, htmlData string) ([]*model.WindStatistics, error) {
	fileName, err := getStationStatisticsFileName(stationID, htmlData)
	if err != nil {
		return nil, fmt.Errorf("failed to get name of the file with necessary station statistics: %w", err)
	}

	file, err := getWindStatisticsFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	hourlyStatistics, err := getHourlyStatistics(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	annualStatistics := countAnnualStatistics(stationName, hourlyStatistics)
	return annualStatistics, nil
}

// GetStationStatisticsFileName looks for corresponding station statistics file name in html result.
func getStationStatisticsFileName(stationID, htmlResult string) (string, error) {
	z := html.NewTokenizer(strings.NewReader(htmlResult))

	var isLink bool

	for {
		tokenType := z.Next()
		tokenData := z.Token().Data

		switch tokenType {
		case html.ErrorToken:
			return "", errors.New("info file not found")
		case html.StartTagToken:
			if tokenData == "a" {
				isLink = true
			}
		case html.TextToken:
			if isLink {
				isDataFile, err := regexp.MatchString(dataFileNameRegExp+stationID, tokenData)
				if err != nil {
					logger.Error(fmt.Errorf("failed to check regexp in data file name: %v", err))
					continue
				}
				if isDataFile {
					return tokenData, err
				}

				isLink = false
			}
		default:
			continue
		}
	}
}

// GetWindStatisticsFile retrieves the file with necessary wind statistics.
func getWindStatisticsFile(fileName string) (*zip.File, error) {
	resp, err := http.Get(os.Getenv("HOURLY_WIND_HISTORICAL_DATA_URL") + fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get wind data by station from source: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create a zip reader: %w", err)
	}

	for _, file := range zipReader.File {
		isProductFile, err := regexp.MatchString(productFileNameRegExp, file.Name)
		if err != nil {
			logger.Error(fmt.Errorf("failed to check regexp in product file name: %v", err))
			continue
		}

		if isProductFile {
			return file, nil
		}
	}

	return nil, errors.New("there is no product file")
}

// GetHourlyStatistics reads wind file content and retrieves hourly statistics.
func getHourlyStatistics(zf *zip.File) ([]*model.HourlyStatistics, error) {
	file, err := zf.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open a file: %w", err)
	}
	defer file.Close()

	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)

	var fileLines []string
	for fileScanner.Scan() {
		fileLines = append(fileLines, fileScanner.Text())
	}

	hourlyStatistics := make([]*model.HourlyStatistics, 0, len(fileLines)-1)
	for _, line := range fileLines[1:] {
		hs, err := parseHourlyStatistics(line)
		if err != nil {
			logger.Error(err)
			continue
		}

		hourlyStatistics = append(hourlyStatistics, hs)
	}

	return hourlyStatistics, nil
}

// parseHourlyStatistics parses stations hourly statistics file line.
func parseHourlyStatistics(line string) (*model.HourlyStatistics, error) {
	lineNoSpaces := strings.ReplaceAll(line, " ", "")
	parts := strings.Split(lineNoSpaces, ";")

	endDate, err := time.Parse(timeLayout, parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse end date value: %w", err)
	}

	speed, err := strconv.ParseFloat(parts[3], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse speed value: %w", err)
	}

	return &model.HourlyStatistics{
		EndDate: endDate,
		Speed:   speed,
	}, nil
}

// CountAnnualStatistics transforms hourly statistics into annual one, calculating average wind speed for every year.
func countAnnualStatistics(stationName string, hourlyStatistics []*model.HourlyStatistics) []*model.WindStatistics {
	var year, previous int
	var sum float64
	var num int

	// find approximate number of years in statistics
	approxSize := len(hourlyStatistics)/24/30/12 + 1

	annualStatistics := make([]*model.WindStatistics, 0, approxSize)
	for i, hm := range hourlyStatistics {
		year = hm.EndDate.Year()

		if previous == 0 {
			previous = year
		}

		// may be -999 - unknown
		if hm.Speed < 0 {
			continue
		}

		// summarize all hourly speed values for every year including the value of the next year first day for 00h
		// (contains speed measurement for the hour before 00h)
		if year == previous {
			sum += hm.Speed
			num++
			if i == len(hourlyStatistics)-1 {
				avg := sum / float64(num)
				annualStatistics = append(annualStatistics, &model.WindStatistics{StationName: stationName, Year: year, Speed: avg})
			}
		} else {
			sum += hm.Speed
			num++
			avg := sum / float64(num)
			annualStatistics = append(annualStatistics, &model.WindStatistics{StationName: stationName, Year: previous, Speed: avg})

			sum = 0
			num = 0
			previous = year
		}
	}

	return annualStatistics
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

// GetStationsInfo retrieves stations information from the source file.
func getStationsInfo() ([]*model.Station, error) {
	resp, err := http.Get(os.Getenv("STATIONS_INFO_URL"))
	if err != nil {
		return nil, fmt.Errorf("failed to get stations info from source: %w", err)
	}
	defer resp.Body.Close()

	// for german specific characters
	reader := transform.NewReader(resp.Body, charmap.ISO8859_15.NewDecoder())

	fileScanner := bufio.NewScanner(reader)
	fileScanner.Split(bufio.ScanLines)

	var fileLines []string
	for fileScanner.Scan() {
		fileLines = append(fileLines, fileScanner.Text())
	}

	// first two lines contain no useful data
	stationsInfo := make([]*model.Station, 0, len(fileLines)-2)
	for _, line := range fileLines[2:] {
		stationInfo, err := parseStationsInfo(line)
		if err != nil {
			logger.Error(err)
			continue
		}

		stationsInfo = append(stationsInfo, stationInfo)
	}

	return stationsInfo, nil
}

// ParseStationsInfo parses stations info file line.
func parseStationsInfo(line string) (*model.Station, error) {
	parts := strings.Fields(line)

	// Station name can have more than one word;
	// get all the data after geoLange und before Bundesland columns
	stationName := parts[6 : len(parts)-1]

	return &model.Station{
		ID:   parts[0],
		Name: strings.Join(stationName, " "),
	}, nil
}
