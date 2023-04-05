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
	InsertYearMeasurements(ctx context.Context, measurements []*model.AverageYearWindSpeed) error
	GetStationID(ctx context.Context, stationName string) (string, error)
	InsertStationsInfo(ctx context.Context, stationsInfo []*model.Station) error
	GetStationWindData(ctx context.Context, stationName string, years int) ([]*model.AverageYearWindSpeed, error)
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

// GetWindInfo implements wind info getting.
func (ws *WeatherService) GetWindInfo(ctx context.Context, req *model.WindRequest) ([]*model.AverageYearWindSpeed, error) {
	long, lat, err := getCoordinates(req.City)
	if err != nil {
		return nil, fmt.Errorf("failed to get coordinates: %w", err)
	}

	stationName, err := getNearestStationName(long, lat)
	if err != nil {
		return nil, fmt.Errorf("failed to get coordinates: %w", err)
	}

	stationID, err := ws.repo.GetStationID(ctx, stationName)
	if err == repository.ErrNoSuchStation {
		stationsInfo, stantionsErr := getStationsInfo()
		if stantionsErr != nil {
			return nil, fmt.Errorf("failed to get station info: %w", stantionsErr)
		}

		err = ws.repo.InsertStationsInfo(ctx, stationsInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to insert stations info: %w", err)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get station id: %w", err)
	}

	data, err := ws.repo.GetStationWindData(ctx, stationName, req.Years)
	if err == repository.ErrNoWindDataForStation {
		resp, err := http.Get(os.Getenv("HOURLY_WIND_HISTORICAL_INFO_URL"))
		if err != nil {
			return nil, fmt.Errorf("failed to get wind data from source: %w", err)
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		yearMeasurements, err := ws.processInfo(stationID, stationName, string(body))
		if err != nil {
			return nil, fmt.Errorf("failed to process info: %w", err)
		}

		err = ws.repo.InsertYearMeasurements(ctx, yearMeasurements)
		if err != nil {
			return nil, fmt.Errorf("failed to insert year measurements: %w", err)
		}

		data, err = ws.repo.GetStationWindData(ctx, stationName, req.Years)
		if err != nil {
			return nil, fmt.Errorf("failed to get station wind data: %w", err)
		}
	}
	if err != nil && err != repository.ErrNoWindDataForStation {
		return nil, fmt.Errorf("failed to get station wind data: %w", err)
	}

	return data, nil
}

func (ws *WeatherService) processInfo(stationID, stationName, htmlResponse string) ([]*model.AverageYearWindSpeed, error) {
	fileName, err := getInfoFileNameFromHTML(stationID, htmlResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to get name of the file with necessary info: %w", err)
	}

	file, err := getWindDataFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	hourMeasurements, err := readWindFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	yearMeasurements := convertToYears(stationName, hourMeasurements)
	return yearMeasurements, nil
}

// GetInfoFileNameFromHTML looks for corresponding station info file name in html result.
func getInfoFileNameFromHTML(stationID, body string) (string, error) {
	z := html.NewTokenizer(strings.NewReader(body))

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

// GetWindDataFile gets and returns a file with necessary wind data.
func getWindDataFile(fileName string) (*zip.File, error) {
	resp, err := http.Get(os.Getenv("HOURLY_WIND_HISTORICAL_INFO_URL") + fileName)
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

// ReadWindFile reads wind file content and parses it.
func readWindFile(zf *zip.File) ([]*model.WindMeasurment, error) {
	file, err := zf.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open a zip file: %w", err)
	}
	defer file.Close()

	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)

	var fileLines []string
	for fileScanner.Scan() {
		fileLines = append(fileLines, fileScanner.Text())
	}

	hourMeasurements := make([]*model.WindMeasurment, 0, len(fileLines)-1)
	for _, line := range fileLines[1:] {
		hourMeasurement, err := processFileLine(line)
		if err != nil {
			logger.Error(err)
			continue
		}

		hourMeasurements = append(hourMeasurements, hourMeasurement)
	}

	return hourMeasurements, nil
}

func processFileLine(line string) (*model.WindMeasurment, error) {
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

	return &model.WindMeasurment{
		EndDate: endDate,
		Speed:   speed,
	}, nil
}

// ConvertToYears transforms hourly data into yearly data,
// counting average wind speed for every year
func convertToYears(stationName string, hourMeasurements []*model.WindMeasurment) []*model.AverageYearWindSpeed {
	var year, previous int
	var sum float64
	var num int

	yearMeasurements := make([]*model.AverageYearWindSpeed, 0, 75)
	for i, hm := range hourMeasurements {
		year = hm.EndDate.Year()

		if previous == 0 {
			previous = year
		}

		if year == previous {
			sum += hm.Speed
			num++
			if i == len(hourMeasurements)-1 {
				avg := sum / float64(num)
				yearMeasurements = append(yearMeasurements, &model.AverageYearWindSpeed{StationName: stationName, Year: year, Speed: avg})
			}
		} else {
			sum += hm.Speed
			num++
			avg := sum / float64(num)
			yearMeasurements = append(yearMeasurements, &model.AverageYearWindSpeed{StationName: stationName, Year: previous, Speed: avg})

			sum = 0
			num = 0
			previous = year
		}
	}

	return yearMeasurements
}

func getCoordinates(city string) (float64, float64, error) {
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

func getStationsInfo() ([]*model.Station, error) {
	resp, err := http.Get(os.Getenv("STATIONS_INFO_URL"))
	if err != nil {
		return nil, fmt.Errorf("failed to get stations info from source: %w", err)
	}
	defer resp.Body.Close()

	reader := transform.NewReader(resp.Body, charmap.ISO8859_15.NewDecoder())
	fileScanner := bufio.NewScanner(reader)
	fileScanner.Split(bufio.ScanLines)

	var fileLines []string
	for fileScanner.Scan() {
		fileLines = append(fileLines, fileScanner.Text())
	}

	stationsInfo := make([]*model.Station, 0, len(fileLines)-2)
	for _, line := range fileLines[2:] {
		stationInfo, err := processStationsFileLine(line)
		if err != nil {
			logger.Error(err)
			continue
		}

		stationsInfo = append(stationsInfo, stationInfo)
	}

	return stationsInfo, nil
}

func processStationsFileLine(line string) (*model.Station, error) {
	parts := strings.Fields(line)

	// can have more than one word in name;
	// get all the data after geoLange und before Bundesland
	stationName := parts[6 : len(parts)-1]

	return &model.Station{
		ID:   parts[0],
		Name: strings.Join(stationName, " "),
	}, nil
}
