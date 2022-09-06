package db

import (
	"context"
	"fadel-blog-services/configs/helpers"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var DB, Ctx = connectDB()

func connectDB() (*mongo.Database, context.Context) {
	// Open mongodb connection
	uri := helpers.GetEnvVariable("DB_URL")
	// Declare Context type object for managing multiple API requests
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}

	// Ping the primary / Check the connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		panic(err)
	}

	db := client.Database("fadel-blog")
	fmt.Println("database connected...")
	return db, ctx
}
