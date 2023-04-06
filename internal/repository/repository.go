// Package repository provides methods to initialize db and perform different db queries.
package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/katiamach/weather-service-api/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DB collections.
const (
	windStatsCollection = "windStats"
	stationsCollection  = "stations"
)

// DB errors.
var (
	ErrNoSuchStation        = errors.New("station with the given name does not exist")
	ErrNoWindDataForStation = errors.New("there is no wind data for the given station")
	ErrNoStations           = errors.New("there are no stations yet")
)

// Repository wraps database and mongo client.
type Repository struct {
	client *mongo.Client
	db     *mongo.Database
}

// New creates new repository from mongo database.
func New() (*Repository, error) {
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := NewMongoDBClient(ctxWithTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}
	db := client.Database(os.Getenv("DB_NAME"))

	err = createIndexes(ctxWithTimeout, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return &Repository{
		client: client,
		db:     db,
	}, nil
}

// CreateIndexes creates necessary indexes for collections.
func createIndexes(ctx context.Context, db *mongo.Database) error {
	indexModelStations := mongo.IndexModel{
		Keys:    bson.M{"name": 1},
		Options: options.Index().SetUnique(true),
	}

	_, err := db.Collection(stationsCollection).Indexes().CreateOne(ctx, indexModelStations)
	if err != nil {
		return fmt.Errorf("failed to create unique station name index: %w", err)
	}

	return nil
}

// Close closes mongo db connection.
func (r *Repository) Close() error {
	if err := r.client.Disconnect(context.TODO()); err != nil {
		return fmt.Errorf("failed to disconnect from mongodb: %w", err)
	}

	return nil
}

// InsertAnnualStatistics inserts annual statistics into windStats collection.
func (r *Repository) InsertAnnualStatistics(ctx context.Context, measurements []*model.WindStatistics) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	m := make([]interface{}, 0, len(measurements))
	for _, v := range measurements {
		m = append(m, v)
	}

	res, err := r.db.Collection(windStatsCollection).InsertMany(ctxWithTimeout, m)
	if err != nil {
		return err
	}
	if len(res.InsertedIDs) != len(m) {
		return errors.New("not all data was inserted")
	}

	return nil
}

// GetStationID gets station id by its name.
func (r *Repository) GetStationID(ctx context.Context, stationName string) (string, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{
		"name": stationName,
	}

	station := new(model.Station)
	err := r.db.Collection(stationsCollection).FindOne(ctxWithTimeout, filter).Decode(station)
	if err == mongo.ErrNoDocuments {
		return "", ErrNoSuchStation
	}
	if err != nil {
		return "", err
	}

	return station.ID, nil
}

// InsertStationsInfo inserts stations info into stations collection.
func (r *Repository) InsertStationsInfo(ctx context.Context, stationsInfo []*model.Station) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	m := make([]interface{}, 0, len(stationsInfo))
	for _, v := range stationsInfo {
		m = append(m, v)
	}

	res, err := r.db.Collection(stationsCollection).InsertMany(ctxWithTimeout, m)
	if err != nil {
		return err
	}
	if len(res.InsertedIDs) != len(m) {
		return errors.New("not all data was inserted")
	}

	return nil
}

// GetStationWindStatistics get wind data of the given data for the given amount of last years.
func (r *Repository) GetStationWindStatistics(ctx context.Context, stationName string, years int) ([]*model.WindStatistics, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	curYear, _, _ := time.Now().Date()

	filter := bson.M{
		"stationName": stationName,
		// data from the last <years> years
		"year": bson.M{"$gte": curYear - years},
	}

	opts := options.Find().SetSort(bson.M{"year": -1})

	windData, err := r.filterWindData(ctxWithTimeout, filter, opts)
	if err == mongo.ErrNoDocuments {
		return nil, ErrNoWindDataForStation
	}
	if err != nil {
		return nil, err
	}

	return windData, nil
}

func (r *Repository) filterWindData(ctx context.Context, filter primitive.M, opts *options.FindOptions) ([]*model.WindStatistics, error) {
	var windData []*model.WindStatistics

	cur, err := r.db.Collection(windStatsCollection).Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		wd := model.WindStatistics{}
		err := cur.Decode(&wd)
		if err != nil {
			return nil, err
		}

		windData = append(windData, &wd)
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	if len(windData) == 0 {
		return nil, mongo.ErrNoDocuments
	}

	return windData, nil
}

// GetStationsCoordinates get stations coordinates.
func (r *Repository) GetStationsCoordinates(ctx context.Context) ([]*model.Station, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	stations, err := r.filterStations(ctxWithTimeout, bson.M{}, nil)
	if err == mongo.ErrNoDocuments {
		return nil, ErrNoStations
	}
	if err != nil {
		return nil, err
	}

	return stations, nil
}

func (r *Repository) filterStations(ctx context.Context, filter primitive.M, opts *options.FindOptions) ([]*model.Station, error) {
	var stations []*model.Station

	cur, err := r.db.Collection(stationsCollection).Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		st := model.Station{}
		err := cur.Decode(&st)
		if err != nil {
			return nil, err
		}

		stations = append(stations, &st)
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	if len(stations) == 0 {
		return nil, mongo.ErrNoDocuments
	}

	return stations, nil
}

// CheckIfStatisticsExists check if statistics collection is not empty.
func (r *Repository) CheckIfStatisticsExists(ctx context.Context) (bool, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	num, err := r.db.Collection(windStatsCollection).CountDocuments(ctxWithTimeout, bson.M{})

	return num > 0, err
}
