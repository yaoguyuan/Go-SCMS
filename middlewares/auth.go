package middlewares

import (
	"auth/initializers"
	"auth/models"
	"auth/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Using JWT for authentication
func RequireAuthentication(c *gin.Context) {
	// Get JWT off the cookie
	JWT, err := c.Cookie("Authorization")
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Decode and validate it
	token, err := jwt.ParseWithClaims(JWT, jwt.MapClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(initializers.SecretKey), nil
		})
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Find the user with the token
	id := uint(token.Claims.(jwt.MapClaims)["sub"].(float64)) // JWT will parse a figure to float64

	// First check if the user exists in the Redis cache
	var user models.User
	// Convert the user ID to string for Redis key
	idStr := strconv.Itoa(int(id))
	initializers.RDB.HGetAll(initializers.RDB_CTX, utils.CACHE_USER_KEY_PREFIX+idStr).Scan(&user)
	if user == (models.User{}) {
		// If the user is not in the cache, then fetch it from the database
		result := initializers.DB.First(&user, id)
		if result.Error != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// Finally, cache the user in Redis and set an expiration time
		initializers.RDB.HSet(initializers.RDB_CTX, utils.CACHE_USER_KEY_PREFIX+idStr, &user)
		initializers.RDB.Expire(initializers.RDB_CTX, utils.CACHE_USER_KEY_PREFIX+idStr, utils.CACHE_USER_EXPIRE_TIME)
	}

	// Attach the user to the context
	c.Set("user", user)

	// Continue
	c.Next()
}

// Using Casbin for authorization
func RequireAuthorization(c *gin.Context) {
	// Get the user, path, method off the context
	user, _ := c.Get("user")
	sub := user.(models.User).Email
	obj := c.Request.URL.Path
	act := c.Request.Method

	// Check if the user is authorized to access the resource
	ok, _ := initializers.E.Enforce(sub, obj, act)
	if !ok {
		c.AbortWithStatus(http.StatusForbidden)
		initializers.LOGGER.Warn("Authorization Failed", "email", sub, "path", obj, "method", act)
		return
	} else {
		// Continue
		c.Next()
	}
}
