// Package repository provides methods to initialize db and perform different db queries.
package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	windCollection     = "windStats"
	stationsCollection = "stations"
)

// DB errors.
var (
	ErrNoSuchStation        = errors.New("station with the given name does not exist")
	ErrNoWindDataForStation = errors.New("there is no wind data for the given station")
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

	res, err := r.db.Collection(windCollection).InsertMany(ctxWithTimeout, m)
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

	filter := bson.M{
		"stationName": stationName,
	}

	opts := options.Find().SetSort(bson.M{"year": -1}).
		SetLimit(int64(years))

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

	cur, err := r.db.Collection(windCollection).Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := cur.Close(ctx)
		if err != nil {
			log.Printf("failed to close cursor: %v", err)
		}
	}()

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
