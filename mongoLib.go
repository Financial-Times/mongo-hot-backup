package main

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type closer interface {
	Close(ctx context.Context) error
}

type mongoSession interface {
	FindAll(ctx context.Context, database, collection string) (mongoCursor, error)
	RemoveAll(ctx context.Context, database, collection string) error
	BulkWrite(ctx context.Context, database, collection string, models []mongo.WriteModel) error

	closer
}

type mongoCursor interface {
	Next(ctx context.Context) bool
	Current() []byte
	Err() error

	closer
}

type cursor struct {
	*mongo.Cursor
}

func (c *cursor) Current() []byte {
	return c.Cursor.Current
}

type mongoClient struct {
	client *mongo.Client
}

func newMongoClient(ctx context.Context, uri string, timeout time.Duration) (*mongoClient, error) {
	uri = fmt.Sprintf("mongodb://%s", uri)
	opts := options.Client().
		ApplyURI(uri).
		SetSocketTimeout(timeout)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &mongoClient{
		client: client,
	}, nil
}

func (m mongoClient) FindAll(ctx context.Context, database, collection string) (mongoCursor, error) {
	cur, err := m.client.
		Database(database).
		Collection(collection).
		Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}

	return &cursor{cur}, nil
}

func (m mongoClient) RemoveAll(ctx context.Context, database, collection string) error {
	_, err := m.client.
		Database(database).
		Collection(collection).
		DeleteMany(ctx, bson.D{})
	return err
}

func (m mongoClient) BulkWrite(ctx context.Context, database, collection string, models []mongo.WriteModel) error {

	readOpts, err := readpref.New(readpref.PrimaryMode, readpref.WithMaxStaleness(time.Second))
	if err != nil {
		return err
	}
	if err = m.client.Ping(ctx, readOpts); err != nil {
		return err
	}

	opts := options.BulkWrite().SetOrdered(false)

	_, err = m.client.
		Database(database).
		Collection(collection).
		BulkWrite(ctx, models, opts)

	return err
}

func (m mongoClient) Close(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	return m.client.Disconnect(ctx)
}
