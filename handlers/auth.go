package handlers

import (
	"context"
	"crypto/sha256"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/mailwilliams/recipes-api/models"
	"github.com/mailwilliams/recipes-api/utils"
	"github.com/rs/xid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"time"
)

type AuthHandler struct {
	collection *mongo.Collection
	ctx        context.Context
}

func NewAuthHandler(ctx context.Context, collection *mongo.Collection) *AuthHandler {
	return &AuthHandler{
		collection: collection,
		ctx:        ctx,
	}
}

func (handler *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		sessionToken := session.Get("token")
		if sessionToken == nil {
			utils.ErrorResponse(c, http.StatusForbidden, errors.New("not logged"))
			c.Abort()
		}
		c.Next()
	}
}

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

type JWTOutput struct {
	Token   string    `json:"token"`
	Expires time.Time `json:"expires"`
}

func (handler *AuthHandler) SignInHandler(c *gin.Context) {
	var user models.User

	if err := c.ShouldBindJSON(&user); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err)
	}

	h := sha256.New()
	curr := handler.collection.FindOne(handler.ctx, bson.M{
		"username": user.Username,
		"password": string(h.Sum([]byte(user.Password))),
	})
	if err := curr.Err(); err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, err)
		return
	}

	if err := curr.Decode(&user); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	sessionToken := xid.New().String()
	session := sessions.Default(c)
	session.Set("token", sessionToken)
	if err := session.Save(); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User signed in"})
}

func (handler *AuthHandler) RefreshHandler(c *gin.Context) {
	session := sessions.Default(c)
	sessionToken := session.Get("token")
	if sessionToken == nil {
		utils.ErrorResponse(c, http.StatusForbidden, errors.New("not logged"))
		return
	}

	sessionToken = xid.New().String()
	session.Set("token", sessionToken)
	if err := session.Save(); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User token refreshed"})
}

func (handler *AuthHandler) SignUpHandler(c *gin.Context) {
	var user models.User

	if err := c.ShouldBindJSON(&user); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	if user.Username == "" || user.Password == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, errors.New("username and password cannot be empty"))
		return
	}

	//	check if account with email address exists already
	switch err := handler.collection.FindOne(handler.ctx, bson.M{"username": user.Username}).Err(); err {
	case mongo.ErrNoDocuments:
		//	account doesn't exist with email, continue
		break
	case nil:
		//	no error, which means we found someone with that email and need to return a bad response
		utils.ErrorResponse(c, http.StatusBadRequest, errors.New("email address already in use"))
		return
	default:
		//	something broke
		utils.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	h := sha256.New()
	_, err := handler.collection.InsertOne(handler.ctx, bson.M{
		"username": user.Username,
		"password": string(h.Sum([]byte(user.Password))),
	})
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	sessionToken := xid.New().String()
	session := sessions.Default(c)
	session.Set("token", sessionToken)
	if err := session.Save(); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User signed up"})
}
