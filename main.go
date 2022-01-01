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
	"github.com/gin-contrib/sessions"
	redisStore "github.com/gin-contrib/sessions/redis"
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
	authHandler    *handlers.AuthHandler
	recipesHandler *handlers.RecipesHandler
)

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

	authHandler = handlers.NewAuthHandler(
		ctx,
		client.
			Database(os.Getenv("MONGO_DATABASE")).
			Collection("users"),
	)
	recipesHandler = handlers.NewRecipesHandler(
		ctx,
		client.
			Database(os.Getenv("MONGO_DATABASE")).
			Collection("recipes"),
		redisClient,
	)
}

func main() {
	router := gin.Default()

	store, _ := redisStore.NewStore(10, "tcp", "localhost:6379", "", []byte("secret"))
	router.Use(sessions.Sessions("recipes-api", store))

	router.GET(
		"/recipes",
		recipesHandler.ListRecipesHandler,
	)
	router.POST(
		"/signup",
		authHandler.SignUpHandler,
	)
	router.POST(
		"/signin",
		authHandler.SignInHandler,
	)
	router.POST(
		"/refresh",
		authHandler.RefreshHandler,
	)

	authorized := router.Group("/")
	authorized.Use(authHandler.AuthMiddleware())
	{
		authorized.GET(
			"/recipes/search",
			recipesHandler.SearchRecipesHandler,
		)
		authorized.GET(
			"/recipes/:id",
			recipesHandler.GetRecipeByIDHandler,
		)
		authorized.POST(
			"/recipes",
			recipesHandler.NewRecipeHandler,
		)
		authorized.PUT(
			"/recipes/:id",
			recipesHandler.UpdateRecipeHandler,
		)
		authorized.DELETE(
			"/recipes/:id",
			recipesHandler.DeleteRecipeHandler,
		)
	}

	_ = router.Run()
}
