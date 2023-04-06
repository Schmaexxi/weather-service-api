package api

import (
	"os"

	"github.com/gorilla/handlers"
)

func setupCorsOptions() []handlers.CORSOption {
	credentials := handlers.AllowCredentials()
	methods := handlers.AllowedMethods([]string{"POST", "OPTIONS"})
	origins := handlers.AllowedOrigins([]string{os.Getenv("ORIGIN")})
	headers := handlers.AllowedHeaders([]string{"Content-Type"})

	options := []handlers.CORSOption{credentials, methods, origins, headers}
	return options
}
