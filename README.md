# weather-service-api

To run the app:
* specify the following environment variables:
    - `PORT` (optional)

    - `ORIGIN` (http://localhost:3000)
    - `REACT_APP_WEATHER_API_URL` (http://localhost:8080)

    - `DB_CONN_STRING` (mongodb://localhost:27017)
    - `DB_NAME`

    - `HOURLY_WIND_HISTORICAL_DATA_URL`

    - `GEO_API_URL` - url of the API for getting geo coordinates by city name (in my case `http://api.positionstack.com/v1/forward`)
    - `GEO_API_ACCESS_KEY` - access key for the API above

    - `STATIONS_INFO_URL` - url of the file with weather station information

* run `docker compose up` command to start mongo db container.  

* run `go run cmd/main.go` command to start the server.  

* run `cd frontend/ && npm install && npm start` command to start the server.  