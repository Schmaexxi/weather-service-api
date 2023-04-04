package main

import (
	"fmt"

	"github.com/katiamach/weather-service-api/internal/api"
	"github.com/katiamach/weather-service-api/internal/logger"
)

func main() {
	err := api.RunAPI()
	if err != nil {
		logger.Fatal(fmt.Errorf("failed to run weather api: %v", err))
	}
}
