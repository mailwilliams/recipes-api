package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/mailwilliams/recipes-api/models"
	"github.com/mailwilliams/recipes-api/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"time"
)

type RecipesHandler struct {
	collection *mongo.Collection
	ctx        context.Context
}

func NewRecipesHandler(ctx context.Context, collection *mongo.Collection) *RecipesHandler {
	return &RecipesHandler{
		collection: collection,
		ctx:        ctx,
	}
}

func (handler *RecipesHandler) ListRecipesHandler(c *gin.Context) {
	cur, err := handler.collection.Find(handler.ctx, bson.M{})
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	defer func(ctx context.Context) {
		_ = cur.Close(ctx)
	}(handler.ctx)

	recipes := make([]models.Recipe, 0)
	for cur.Next(handler.ctx) {
		var recipe models.Recipe
		if err = cur.Decode(&recipe); err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}
		recipes = append(recipes, recipe)
	}
	c.JSON(http.StatusOK, recipes)
}

func (handler *RecipesHandler) NewRecipeHandler(c *gin.Context) {
	var recipe models.Recipe

	if err := c.ShouldBindJSON(&recipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	recipe.ID = primitive.NewObjectID()
	recipe.PublishedAt = time.Now()
	_, err := handler.collection.InsertOne(handler.ctx, recipe)
	if err != nil {
		fmt.Println(err)
		utils.ErrorResponse(
			c,
			http.StatusInternalServerError,
			errors.New("error while inserting a new recipe"),
		)
		return
	}
	c.JSON(http.StatusOK, recipe)
}

func (handler *RecipesHandler) SearchRecipesHandler(c *gin.Context) {
	tag := c.Query("tag")
	cur, err := handler.collection.Find(handler.ctx, bson.M{
		"tags": tag,
	})
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	defer func() {
		_ = cur.Close(handler.ctx)
	}()
	recipes := make([]models.Recipe, 0)
	for cur.Next(handler.ctx) {
		var recipe models.Recipe
		err = cur.Decode(&recipe)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, err)
			return
		}
		recipes = append(recipes, recipe)
	}
	c.JSON(http.StatusOK, recipes)
}

func (handler *RecipesHandler) UpdateRecipeHandler(c *gin.Context) {
	id := c.Param("id")
	var recipe models.Recipe

	if err := c.ShouldBindJSON(&recipe); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err)
		return
	}

	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	_, err = handler.collection.UpdateOne(handler.ctx, bson.M{
		"_id": objectId,
	}, bson.D{{"$set", bson.D{
		bson.E{Key: "name", Value: recipe.Name},
		bson.E{Key: "instructions", Value: recipe.Instructions},
		bson.E{Key: "ingredients", Value: recipe.Ingredients},
		bson.E{Key: "tags", Value: recipe.Tags},
	}}})
	if err != nil {
		fmt.Println(err)
		utils.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Recipe has been updated"})
}

func (handler *RecipesHandler) DeleteRecipeHandler(c *gin.Context) {
	id := c.Param("id")

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	_, err = handler.collection.DeleteOne(handler.ctx, bson.M{
		"_id": objectID,
	})
	if err != nil {
		fmt.Println(err)
		utils.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Recipe has been deleted",
	})
}
