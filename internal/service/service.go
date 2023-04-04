package service

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/katiamach/weather-service-api/internal/logger"
	"github.com/katiamach/weather-service-api/internal/model"
)

var (
	dataFileNameRegExp    = `(stundenwerte_FF_+)`
	productFileNameRegExp = `(produkt+)`
)

const timeLayout = "2006010215"

// WeatherService provides weather service functionality.
type WeatherService struct{}

// New creates new WeatherService.
func New() *WeatherService {
	return &WeatherService{}
}

// GetWindInfo implements wind info getting.
func (ws *WeatherService) GetWindInfo(ctx context.Context, req *model.WindRequest) error {
	// check if exists in db

	resp, err := http.Get(os.Getenv("HOURLY_WIND_HISTORICAL_INFO_URL"))
	if err != nil {
		return fmt.Errorf("failed to get wind data from source: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	err = ws.processInfo(string(body))
	if err != nil {
		return fmt.Errorf("failed to process info: %w", err)
	}

	return nil
}

func (ws *WeatherService) processInfo(htmlResponse string) error {
	fileName, err := getInfoFileNameFromHTML(htmlResponse)
	if err != nil {
		return fmt.Errorf("failed to get name of the file with necessary info: %w", err)
	}

	file, err := getWindDataFile(fileName)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	hourly, err := readWindFile(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	yearly := convertToYears(hourly)

	_ = yearly
	return nil
}

// GetInfoFileNameFromHTML looks for corresponding station info file name in html result.
func getInfoFileNameFromHTML(body string) (string, error) {
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
				isDataFile, err := regexp.MatchString(dataFileNameRegExp, tokenData)
				if err != nil {
					logger.Error(fmt.Errorf("failed to check regexp in data file name: %v", err))
					continue
				}
				if isDataFile {
					// Just return when found the first corresponding file.
					// Later filter for station id.
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

	hourly := make([]*model.WindMeasurment, 0, len(fileLines))
	for _, line := range fileLines[1:] {
		hourMeasurement, err := processFileLine(line)
		if err != nil {
			logger.Error(err)
			continue
		}

		hourly = append(hourly, hourMeasurement)
	}

	return hourly, nil
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
func convertToYears(hourly []*model.WindMeasurment) []*model.AverageYearWindSpeed {
	var year, previous int
	var sum float64
	var num int

	yearly := make([]*model.AverageYearWindSpeed, 0, 75)
	for _, hm := range hourly {
		year = hm.EndDate.Year()

		if previous == 0 {
			previous = year
		}

		if year == previous {
			sum += hm.Speed
			num++
		} else {
			sum += hm.Speed
			num++
			avg := sum / float64(num)
			yearly = append(yearly, &model.AverageYearWindSpeed{Year: year, Speed: avg})

			sum = 0
			num = 0
			previous = year
		}
	}

	return yearly
}
