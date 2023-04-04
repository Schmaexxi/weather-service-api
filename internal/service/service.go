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
	fileName, err := ws.getInfoFileNameFromHTML(htmlResponse)
	if err != nil {
		return fmt.Errorf("failed to get name of the file with necessary info: %w", err)
	}

	file, err := ws.getWindDataFile(fileName)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	hourMeasurements, err := ws.readWindFile(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	_ = hourMeasurements
	return nil
}

// GetInfoFileNameFromHTML looks for corresponding station info file name in html result.
func (ws *WeatherService) getInfoFileNameFromHTML(body string) (string, error) {
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
func (ws *WeatherService) getWindDataFile(fileName string) (*zip.File, error) {
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
func (ws *WeatherService) readWindFile(zf *zip.File) ([]*model.WindMeasurment, error) {
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

	hourMeasurements := make([]*model.WindMeasurment, 0, len(fileLines))
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
