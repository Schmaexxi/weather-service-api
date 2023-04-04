package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/katiamach/weather-service-api/internal/logger"
	"github.com/katiamach/weather-service-api/internal/service"
	"github.com/katiamach/weather-service-api/internal/transport/rest/handler"
)

// RunAPI runs weather service API.
func RunAPI() error {
	service := service.New()
	server := handler.NewWeatherServer(service)

	r := mux.NewRouter()

	r.HandleFunc("/wind", server.GetWindInfoHandler).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		logger.Info(fmt.Sprintf("Defaulting to port %s", port))
	}

	logger.Info(fmt.Sprintf("Starting weather service api at port %s", port))

	options := setupCorsOptions()
	return http.ListenAndServe(":"+port, handlers.CORS(options...)(r))
}
