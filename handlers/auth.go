package handlers

import (
	"context"
	"crypto/sha256"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/mailwilliams/recipes-api/models"
	"github.com/mailwilliams/recipes-api/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"os"
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
		tokenValue := c.GetHeader("Authorization")
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenValue, claims,
			func(token *jwt.Token) (interface{}, error) {
				return []byte(os.Getenv("JWT_SECRET")), nil
			},
		)
		if err != nil || token == nil || !token.Valid {
			c.AbortWithStatus(http.StatusUnauthorized)
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

	expirationTime := time.Now().Add(10 * time.Minute)
	claims := &Claims{
		Username: user.Username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, JWTOutput{
		Token:   tokenString,
		Expires: expirationTime,
	})
}

func (handler *AuthHandler) RefreshHandler(c *gin.Context) {
	tokenValue := c.GetHeader("Authorization")
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenValue, claims,
		func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		},
	)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, err)
		return
	}
	if token == nil || !token.Valid {
		utils.ErrorResponse(c, http.StatusUnauthorized, errors.New("invalid token"))
		return
	}
	if time.Unix(claims.ExpiresAt, 0).Sub(time.Now()) > 30*time.Second {
		utils.ErrorResponse(c, http.StatusInternalServerError, errors.New("token not expired yet"))
		return
	}
	expirationTime := time.Now().Add(5 * time.Minute)
	claims.ExpiresAt = expirationTime.Unix()
	token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(os.Getenv("JWT_SECRET"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, JWTOutput{
		Token:   tokenString,
		Expires: expirationTime,
	})
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

	expirationTime := time.Now().Add(10 * time.Minute)
	claims := &Claims{
		Username: user.Username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusCreated, JWTOutput{
		Token:   tokenString,
		Expires: expirationTime,
	})
}
