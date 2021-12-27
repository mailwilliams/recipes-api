/*	Recipes API

This is a sample recipes API. You can find out more about the
API at https://github.com/mailwilliams/recipes-api

Schemes: http
Host: localhost:8080
BasePath: /
Version: 1.0.0
Contact:
	Liam Williams
	<liamwilliams1218@gmail.com>

Consumes:
	- application/json

Produces:
	- application/json

swagger:meta
*/
package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/mailwilliams/recipes-api/handlers"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"os"
)

var (
	ctx            context.Context
	err            error
	client         *mongo.Client
	recipesHandler *handlers.RecipesHandler
)

func main() {
	router := gin.Default()
	router.GET(
		"/recipes",
		recipesHandler.ListRecipesHandler,
	)
	router.GET(
		"/recipes/search",
		recipesHandler.SearchRecipesHandler,
	)
	router.POST(
		"/recipes",
		recipesHandler.NewRecipeHandler,
	)
	router.PUT(
		"/recipes/:id",
		recipesHandler.UpdateRecipeHandler,
	)
	router.DELETE(
		"/recipes/:id",
		recipesHandler.DeleteRecipeHandler,
	)
	_ = router.Run()
}

func init() {
	ctx = context.Background()

	client, err = mongo.Connect(
		ctx,
		options.Client().ApplyURI(os.Getenv("MONGO_URI")),
	)
	if err = client.Ping(context.TODO(), readpref.Primary()); err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to MongoDB")

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	status := redisClient.Ping()
	fmt.Println(status)
	
	recipesHandler = handlers.NewRecipesHandler(
		ctx,
		client.
			Database(os.Getenv("MONGO_DATABASE")).
			Collection("recipes"),
		redisClient,
	)
}
