// Package repository provides methods to initialize db and perform different db queries.
package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/katiamach/weather-service-api/internal/model"
	"go.mongodb.org/mongo-driver/mongo"
)

// DB collections.
const (
	windCollection = "wind"
)

// Repository wraps database and mongo client.
type Repository struct {
	client         *mongo.Client
	db             *mongo.Database
	windCollection string
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

	return &Repository{
		client:         client,
		db:             db,
		windCollection: windCollection,
	}, nil
}

// Close closes mongo db connection.
func (r *Repository) Close() error {
	if err := r.client.Disconnect(context.TODO()); err != nil {
		return fmt.Errorf("failed to disconnect from mongodb: %w", err)
	}

	return nil
}

// InsertYearMeasurements inserts inserts year measurements into wind collection.
func (r *Repository) InsertYearMeasurements(ctx context.Context, measurements []*model.AverageYearWindSpeed) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	m := make([]interface{}, 0, len(measurements))
	for _, v := range measurements {
		m = append(m, v)
	}

	res, err := r.db.Collection(r.windCollection).InsertMany(ctxWithTimeout, m)
	if err != nil {
		return err
	}
	if len(res.InsertedIDs) != len(m) {
		return errors.New("not all data was inserted")
	}

	return nil
}
