package controllers

import (
	"auth/initializers"
	"auth/models"
	"auth/utils"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Fetch retrieves the current user's information.
func Fetch(c *gin.Context) {
	// Get the user off the context
	user, _ := c.Get("user")
	userMap := map[string]interface{}{
		"id":      user.(models.User).ID,
		"email":   user.(models.User).Email,
		"address": user.(models.User).Address,
	}

	// Return a success response with the user
	c.JSON(http.StatusOK, gin.H{
		"message": "User fetched successfully",
		"user":    userMap,
	})
}

// Modify updates the current user's information.
func Modify(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			initializers.LOGGER.Error("Modify failed", "error", err, "sub", utils.GetSubInfo(c), "params", c.MustGet("params"))
		}
	}()

	// [Get the filtered parsed body and save it to the context]
	rawBody := utils.GetRawBody(c)
	parsedBody := utils.GetParsedBody(rawBody)
	utils.BlurMap(parsedBody, "password")
	c.Set("params", parsedBody)

	// Get the user off the context
	user, _ := c.Get("user")
	userID := user.(models.User).ID
	userEmail := user.(models.User).Email
	userRole := user.(models.User).Role
	userAddress := user.(models.User).Address

	// Get the email and password off the request body
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Address  string `json:"address"`
	}
	c.ShouldBind(&body)

	// Hash the password
	var hashedPassword string
	if body.Password != "" {
		hashedPasswordBytes, _ := bcrypt.GenerateFromPassword([]byte(body.Password), 10)
		hashedPassword = string(hashedPasswordBytes)
	}

	// Update the user in the database
	result := initializers.DB.Where("id = ?", userID).Updates(&models.User{Email: body.Email, Password: hashedPassword, Address: body.Address})
	if result.Error != nil || result.RowsAffected == 0 {
		panic("Failed to update user in the database")
	}

	// [Get the updated user from the database]
	var new_user models.User
	initializers.DB.Where("id = ?", userID).First(&new_user)

	// Update the authorization rules in Casbin
	oldPolicies, _ := initializers.E.GetFilteredPolicy(0, userEmail)
	var newPolicies [][]string
	for _, policy := range oldPolicies {
		newPolicies = append(newPolicies, []string{body.Email, policy[1], policy[2]})
	}
	initializers.E.UpdatePolicies(oldPolicies, newPolicies)

	// Update the role inheritance rule in Casbin
	initializers.E.UpdateGroupingPolicy([]string{userEmail, userRole}, []string{body.Email, userRole})

	// [Prepare the object and data information for logging]
	objInfo := utils.ObjInfo{
		Op:    utils.OpUpdate,
		Table: models.User{}.TableName(),
		ID:    userID,
	}
	dataInfo := utils.DataInfo{
		OldData: map[string]interface{}{
			"email":    userEmail,
			"password": utils.BlurStr(),
			"address":  userAddress,
		},
		NewData: map[string]interface{}{
			"email":    new_user.Email,
			"password": utils.BlurStr(),
			"address":  new_user.Address,
		},
	}

	// Return a success response
	message := "User modified successfully"
	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
	initializers.LOGGER.Info(message, "sub", utils.GetSubInfo(c), "obj", objInfo, "data", dataInfo)
}

// FetchUsers retrieves all users' information.
func FetchUsers(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the pagination parameters
	params := utils.GetPaginationParams(c)

	// Get the total number of users
	var total int64
	result := initializers.DB.Model(&models.User{}).Count(&total)
	if result.Error != nil {
		panic("Failed to get the total number of users")
	}

	// Get the users from the database
	var users []map[string]interface{}
	result = initializers.DB.Model(&models.User{}).Select("id", "email", "address").Offset(params.Offset).Limit(params.PageSize).Find(&users)
	if result.Error != nil {
		panic("Failed to get the users from the database")
	}

	// Get the pagination result
	pagination := utils.GetPaginationResult(params, len(users), total)

	// Return a success response with the users
	c.JSON(http.StatusOK, gin.H{
		"message":    "Users fetched successfully",
		"users":      users,
		"pagination": pagination,
	})
}

// FetchUser retrieves a user's information.
func FetchUser(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the id off the path parameter
	id, ok := c.Params.Get("id")
	if !ok {
		panic("ID is required")
	}
	userID, err := strconv.Atoi(id)
	if err != nil {
		panic("Invalid ID: Type error")
	}

	// Get the user from the database
	var user map[string]interface{}
	result := initializers.DB.Model(&models.User{}).Select("id", "email", "address").Where("id = ?", uint(userID)).First(&user)
	if result.Error != nil {
		panic("Invalid ID: User not found")
	}

	// Return a success response with the user
	c.JSON(http.StatusOK, gin.H{
		"message": "User fetched successfully",
		"user":    user,
	})
}

// GetAvatar retrieves a user's avatar.
func GetAvatar(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the id off the path parameter
	id, ok := c.Params.Get("id")
	if !ok {
		panic("ID is required")
	}
	userID, err := strconv.Atoi(id)
	if err != nil {
		panic("Invalid ID: Type error")
	}

	// Look up the user in the database and get the avatar file name
	var fileName string
	result := initializers.DB.Model(&models.User{}).Select("avatar").Where("id = ?", uint(userID)).First(&fileName)
	if result.Error != nil {
		panic("Invalid ID: User not found")
	}

	// Check if the file exists
	filePath := filepath.Join(initializers.AvatarDir, fileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		panic("Avatar file not found")
	}

	// Return the file as a response
	c.File(filePath)
}

// UploadAvatar uploads the avatar of the current user.
func UploadAvatar(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the user off the context
	user, _ := c.Get("user")
	userID := user.(models.User).ID

	// Check if the user exists in the database
	result := initializers.DB.First(&models.User{}, userID)
	if result.Error != nil {
		panic("Failed to upload avatar file")
	}

	// Get the avatar file off the request body
	file, err := c.FormFile("avatar")
	if err != nil {
		panic("Failed to get avatar file from request")
	}

	// Validate the file type and size
	fileExt := strings.ToLower(filepath.Ext(file.Filename))
	if fileExt != ".jpg" && fileExt != ".jpeg" && fileExt != ".png" {
		panic("Invalid avatar file: Only .jpg, .jpeg, and .png are allowed")
	}
	if file.Size > 2*1024*1024 {
		panic("Invalid avatar file: File size exceeds 2MB")
	}

	// Generate a unique file name
	fileID := uuid.New().String()
	fileName := fileID + fileExt
	filePath := filepath.Join(initializers.AvatarDir, fileName)

	// Save the file to the server
	err = c.SaveUploadedFile(file, filePath)
	if err != nil {
		panic("Failed to save avatar file")
	}

	// Remove the old avatar file if it exists
	var oldFileName string
	initializers.DB.Model(&models.User{}).Select("avatar").Where("id = ?", userID).First(&oldFileName)
	if oldFileName != initializers.AvatarDefault {
		oldFilePath := filepath.Join(initializers.AvatarDir, oldFileName)
		err := os.Remove(oldFilePath)
		if err != nil {
			panic("Failed to remove old avatar file")
		}
	}

	// Update avatar file name in the database
	result = initializers.DB.Model(&models.User{}).Where("id = ?", userID).Update("avatar", fileName)
	if result.Error != nil {
		panic("Failed to update avatar file name in the database")
	}

	// Return a success response
	c.JSON(http.StatusOK, gin.H{
		"message": "Avatar uploaded successfully",
	})
}
