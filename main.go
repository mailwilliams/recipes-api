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
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	ctx        context.Context
	err        error
	client     *mongo.Client
	collection *mongo.Collection
)

type Recipe struct {
	ID           primitive.ObjectID `json:"id" bson:"_id"`
	Name         string             `json:"name" bson:"name"`
	Tags         []string           `json:"tags" bson:"tags"`
	Ingredients  []string           `json:"ingredients" bson:"ingredients"`
	Instructions []string           `json:"instructions" bson:"instructions"`
	PublishedAt  time.Time          `json:"publishedAt" bson:"publishedAt"`
}

func NewRecipeHandler(c *gin.Context) {
	var recipe Recipe

	if err := c.ShouldBindJSON(&recipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	recipe.ID = primitive.NewObjectID()
	recipe.PublishedAt = time.Now()
	_, err = collection.InsertOne(ctx, recipe)
	if err != nil {
		fmt.Println(err)
		errorResponse(
			c,
			http.StatusInternalServerError,
			errors.New("error while inserting a new recipe"),
		)
		return
	}
	c.JSON(http.StatusOK, recipe)
}

func ListRecipesHandler(c *gin.Context) {
	cur, err := collection.Find(ctx, bson.M{})
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, err)
		return
	}
	defer func() {
		_ = cur.Close(ctx)
	}()
	recipes := make([]Recipe, 0)
	for cur.Next(ctx) {
		var recipe Recipe
		err = cur.Decode(&recipe)
		if err != nil {
			errorResponse(c, http.StatusInternalServerError, err)
			return
		}
		recipes = append(recipes, recipe)
	}
	c.JSON(http.StatusOK, recipes)
}

func SearchRecipesHandler(c *gin.Context) {
	tag := c.Query("tag")
	cur, err := collection.Find(ctx, bson.M{
		"tags": tag,
	})
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, err)
		return
	}
	defer func() {
		_ = cur.Close(ctx)
	}()
	recipes := make([]Recipe, 0)
	for cur.Next(ctx) {
		var recipe Recipe
		err = cur.Decode(&recipe)
		if err != nil {
			errorResponse(c, http.StatusInternalServerError, err)
			return
		}
		recipes = append(recipes, recipe)
	}
	c.JSON(http.StatusOK, recipes)
}

func UpdateRecipeHandler(c *gin.Context) {
	id := c.Param("id")
	var recipe Recipe

	if err := c.ShouldBindJSON(&recipe); err != nil {
		errorResponse(c, http.StatusBadRequest, err)
		return
	}

	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, err)
		return
	}

	_, err = collection.UpdateOne(ctx, bson.M{
		"_id": objectId,
	}, bson.D{{"$set", bson.D{
		bson.E{Key: "name", Value: recipe.Name},
		bson.E{Key: "instructions", Value: recipe.Instructions},
		bson.E{Key: "ingredients", Value: recipe.Ingredients},
		bson.E{Key: "tags", Value: recipe.Tags},
	}}})
	if err != nil {
		fmt.Println(err)
		errorResponse(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Recipe has been updated"})
}

func DeleteRecipeHandler(c *gin.Context) {
	id := c.Param("id")

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, err)
		return
	}

	_, err = collection.DeleteOne(ctx, bson.M{
		"_id": objectID,
	})
	if err != nil {
		fmt.Println(err)
		errorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Recipe has been deleted",
	})
}

func main() {
	router := gin.Default()
	router.GET("/recipes", ListRecipesHandler)
	router.GET("/recipes/search", SearchRecipesHandler)
	router.POST("/recipes", NewRecipeHandler)
	router.PUT("/recipes/:id", UpdateRecipeHandler)
	router.DELETE("/recipes/:id", DeleteRecipeHandler)
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
	collection = client.Database(os.Getenv("MONGO_DATABASE")).Collection("recipes")
	log.Println("Connected to MongoDB")
}

func errorResponse(c *gin.Context, statusCode int, err error) {
	c.JSON(statusCode, gin.H{
		"error": err.Error(),
	})
}
