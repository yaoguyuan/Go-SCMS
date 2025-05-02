package controllers

import (
	"auth/initializers"
	"auth/models"
	"auth/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func SignUp(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			initializers.LOGGER.Error("SignUp failed", "error", err, "ip", c.ClientIP(), "params", c.MustGet("params"))
		}
	}()

	// [Get the filtered parsed body and save it to the context]
	rawBody := utils.GetRawBody(c)
	parsedBody := utils.GetParsedBody(rawBody)
	utils.BlurMap(parsedBody, "password")
	c.Set("params", parsedBody)

	// Get the email and password off the request body
	var body struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBind(&body); err != nil {
		panic("Failed to get email and password off the request body")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(body.Password), 10)
	if err != nil {
		panic("Failed to hash password")
	}

	// Create the user in the database
	result := initializers.DB.Create(&models.User{Email: body.Email, Password: string(hashedPassword)})
	if result.Error != nil {
		panic("Failed to create user in the database")
	}

	// Create a role inheritance rule in Casbin
	initializers.E.AddGroupingPolicy(body.Email, "user")

	// Return a success response
	message := "User signed up successfully"
	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
	initializers.LOGGER.Info(message, "ip", c.ClientIP(), "email", body.Email)
}

func Login(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			initializers.LOGGER.Error("Login failed", "error", err, "ip", c.ClientIP(), "params", c.MustGet("params"))
		}
	}()

	// [Get the filtered parsed body and save it to the context]
	rawBody := utils.GetRawBody(c)
	parsedBody := utils.GetParsedBody(rawBody)
	utils.BlurMap(parsedBody, "password")
	c.Set("params", parsedBody)

	// Get the email and password off the request body
	var body struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBind(&body); err != nil {
		panic("Failed to get email and password off the request body")
	}

	// Look up the user in the database by email
	var user models.User
	result := initializers.DB.Where("email = ?", body.Email).First(&user)
	if result.Error != nil {
		panic("Login failed: user not found or incorrect password")
	}

	// Compare the password with the hashed password in the database
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)); err != nil {
		panic("Login failed: user not found or incorrect password")
	}

	// Generate a JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 30)),
	})
	JWT, err := token.SignedString([]byte(initializers.SecretKey))
	if err != nil {
		panic("Failed to generate a JWT token")
	}

	// Set cookie with the JWT token
	c.SetSameSite(http.SameSiteLaxMode)                                // Lax mode for CSRF protection
	c.SetCookie("Authorization", JWT, 3600*24*30, "", "", false, true) // HttpOnly true for XSS protection

	// Return a success response
	message := "User logged in successfully"
	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
	initializers.LOGGER.Info(message, "ip", c.ClientIP(), "email", body.Email)
}
