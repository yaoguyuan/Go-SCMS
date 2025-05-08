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

	// Validate the email format
	if !utils.IsValidEmail(body.Email) {
		panic("Invalid email format")
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

// SendCode sends a verification code to the user's email address
func SendCode(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the email off the request body
	var body struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBind(&body); err != nil {
		panic("Failed to get email off the request body")
	}

	// Generate a verification code
	code := utils.GenerateVerificationCode(6)

	// Save the verification code to Redis
	err := initializers.RDB.Set(initializers.RDB_CTX, utils.LOGIN_CODE_KEY_PREFIX+body.Email, code, utils.LOGIN_CODE_EXPIRE_TIME).Err()
	if err != nil {
		panic("Failed to save verification code to Redis")
	}

	// Send the verification code to the user's email address
	if err := utils.SendVerificationCode(body.Email, code); err != nil {
		panic("Failed to send verification code")
	}

	// Return a success response
	message := "Verification code sent successfully"
	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
}

// VerifyCode verifies the verification code sent to the user's email address
// If verification is successful, the user is logged in with a JWT token
func VerifyCode(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the email and code off the request body
	var body struct {
		Email string `json:"email" binding:"required"`
		Code  string `json:"code" binding:"required"`
	}
	if err := c.ShouldBind(&body); err != nil {
		panic("Failed to get email and code off the request body")
	}

	// Fetch the cached verification code from Redis
	cachedCode, err := initializers.RDB.Get(initializers.RDB_CTX, utils.LOGIN_CODE_KEY_PREFIX+body.Email).Result()
	if err != nil {
		panic("Login failed: verification code expired or not sent")
	}

	// Check if the given verification code is correct
	if body.Code != cachedCode {
		panic("Login failed: incorrect verification code")
	}

	// Look up the user in the database by email
	var user models.User
	result := initializers.DB.Where("email = ?", body.Email).First(&user)
	if result.Error != nil {
		panic("Login failed: user not found")
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
}
