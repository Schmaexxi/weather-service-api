package repository

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// NewMongoDBClient initializes new mongoDB client and resturns needed database.
func NewMongoDBClient(ctx context.Context) (*mongo.Client, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	connectionURI := fmt.Sprintf("%s/%s", os.Getenv("DB_CONN_STRING"), os.Getenv("DB_NAME"))

	clientOptions := options.Client().ApplyURI(connectionURI)
	client, err := mongo.Connect(ctxWithTimeout, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	err = client.Ping(ctxWithTimeout, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("failed to ping: %w", err)
	}

	return client, nil
}
