package controllers

import (
	"auth/initializers"
	"auth/models"
	"auth/utils"
	"errors"
	"fmt"
	"math/bits"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Fetch retrieves the current user's information.
func Fetch(c *gin.Context) {
	// Get the user off the context
	user, _ := c.Get("user")
	userMap := map[string]interface{}{
		"id":      user.(models.User).ID,
		"email":   user.(models.User).Email,
		"address": user.(models.User).Address,
		"credits": user.(models.User).Credits,
		"subfee":  user.(models.User).Subfee,
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
		Subfee   uint   `json:"subfee"`
	}
	c.ShouldBind(&body)

	// Validate the email format
	if body.Email != "" && !utils.IsValidEmail(body.Email) {
		panic("Invalid email format")
	}

	// Validate the subfee value
	if body.Subfee > 100 {
		panic("Invalid subfee value")
	}

	// Hash the password
	var hashedPassword string
	if body.Password != "" {
		hashedPasswordBytes, _ := bcrypt.GenerateFromPassword([]byte(body.Password), 10)
		hashedPassword = string(hashedPasswordBytes)
	}

	// Update the user in the database
	result := initializers.DB.Where("id = ?", userID).Updates(&models.User{Email: body.Email, Password: hashedPassword, Address: body.Address, Subfee: body.Subfee})
	if result.Error != nil || result.RowsAffected == 0 {
		panic("Failed to update user in the database")
	}

	// Remove the Redis cache for the user
	err := initializers.RDB.Del(initializers.RDB_CTX, utils.RedisConstants.CACHE_USER_KEY_PREFIX+strconv.Itoa(int(userID))).Err()
	if err != nil {
		panic("Failed to remove user from Redis cache")
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

	// Reload the policy from the database
	initializers.E.LoadPolicy()

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

// SignIn 用于记录当前用户在当天完成签到
func SignIn(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the user off the context
	user, _ := c.Get("user")
	userID := user.(models.User).ID

	// Get the date
	currentYear, currentMonth, currentDay := time.Now().Date()

	// Using Redis Bitmap to record the sign-in status
	// (1) Set the key
	keyPrefix := utils.RedisConstants.SIGN_IN_KEY_PREFIX
	keySuffix := fmt.Sprintf(":%d%02d", currentYear, currentMonth)
	key := keyPrefix + strconv.Itoa(int(userID)) + keySuffix
	// (2) Set the bit
	if err := initializers.RDB.SetBit(initializers.RDB_CTX, key, int64(currentDay)-1, 1).Err(); err != nil {
		panic("Failed to record sign-in status")
	}

	// return a success response
	c.JSON(http.StatusOK, gin.H{
		"message": "Sign-in recorded successfully",
	})
}

// SignInCount 用于获取当前用户到当天为止的连续签到次数
func SignInCount(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the user off the context
	user, _ := c.Get("user")
	userID := user.(models.User).ID

	// call function getSignInCount
	count, err := getSignInCount(userID)
	if err != nil {
		panic(err.Error())
	}

	// Return a success response with the count
	c.JSON(http.StatusOK, gin.H{
		"message": "Sign-in count fetched successfully",
		"count":   count,
	})
}

// SignInAward 用于为当前用户发放本月连续签到奖励
func SignInAward(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the user off the context
	user, _ := c.Get("user")
	userID := user.(models.User).ID

	// Check if today is the last day of the month
	currentYear, currentMonth, currentDay := time.Now().Date()
	lastDayOfMonth := time.Date(currentYear, currentMonth+1, 0, 0, 0, 0, 0, time.UTC).Day()
	fmt.Println("Current Day:", currentDay, "Last Day of Month:", lastDayOfMonth)
	if currentDay != lastDayOfMonth {
		panic("Sign-in rewards can only be claimed on the last day of the month")
	}

	// call function getSignInCount
	count, err := getSignInCount(userID)
	if err != nil {
		panic(err.Error())
	}

	// Add the user's credits based on the sign-in count
	// (1) count >= 28 -> 100 credits
	// (2) count >= 15 -> 50 credits
	// (3) 10 credits
	award := 10 + 40*utils.BoolToUint(count >= 15) + 50*utils.BoolToUint(count >= 28)
	result := initializers.DB.Model(&models.User{}).Where("id = ?", userID).Update("credits", gorm.Expr("credits + ?", award))
	if result.Error != nil || result.RowsAffected == 0 {
		panic("Failed to award sign-in credits")
	}

	// Return a success response with the award
	c.JSON(http.StatusOK, gin.H{
		"message": "Sign-in award granted successfully",
		"award":   award,
	})
}

func getSignInCount(userID uint) (int, error) {
	// Get the date
	currentYear, currentMonth, currentDay := time.Now().Date()

	// Using Redis Bitmap to get the sign-in records till today
	// (1) Set the key
	keyPrefix := utils.RedisConstants.SIGN_IN_KEY_PREFIX
	keySuffix := fmt.Sprintf(":%d%02d", currentYear, currentMonth)
	key := keyPrefix + strconv.Itoa(int(userID)) + keySuffix
	// (2) Get the record
	result, err := initializers.RDB.BitFieldRO(initializers.RDB_CTX, key, "u"+strconv.Itoa(currentDay), "0").Result()
	if err != nil || len(result) != 1 {
		return 0, errors.New("failed to get sign-in record")
	}
	record := uint32(result[0])

	// Using Brian Kernighan's algorithm to count the continuous sign-in days
	count := bits.Len32((^record & (record + 1)) - 1)
	return count, nil
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
	result = initializers.DB.Model(&models.User{}).Select("id", "email").Offset(params.Offset).Limit(params.PageSize).Find(&users)
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

	// Look up the user in the database
	user, err := fetchUser(userID)
	if err != nil {
		panic(err.Error())
	}

	// Filter the user information
	selectedUser := make(map[string]interface{})
	selectedUser["id"] = user.ID
	selectedUser["email"] = user.Email
	selectedUser["address"] = user.Address
	selectedUser["subfee"] = user.Subfee

	// Return a success response with the user
	c.JSON(http.StatusOK, gin.H{
		"message": "User fetched successfully",
		"user":    selectedUser,
	})
}

// ************** Using Redis for Caching **************
// *** Solving Cache Penetration and Cache Breakdown ***
func fetchUser(userID int) (*models.User, error) {
	var user models.User
	// Convert the user ID to string for Redis key
	idStr := strconv.Itoa(userID)
	// Check if the user is non-existent and holds an empty value in Redis
	if initializers.RDB.HExists(initializers.RDB_CTX, utils.RedisConstants.CACHE_USER_KEY_PREFIX+idStr, "is_empty").Val() {
		return nil, errors.New("user not found")
	}
	// First-check if the user exists in the Redis cache
	initializers.RDB.HGetAll(initializers.RDB_CTX, utils.RedisConstants.CACHE_USER_KEY_PREFIX+idStr).Scan(&user)
	if user == (models.User{}) {
		// Cache Breakdown Solution
		// Try lock the mutex
		if !utils.SimpleTryLock(utils.RedisConstants.MUTEX_USER_KEY_PREFIX+idStr, utils.RedisConstants.MUTEX_USER_EXPIRE_TIME) {
			// If the lock fails, wait for a while and try again
			time.Sleep(50 * time.Millisecond)
			return fetchUser(userID)
		}
		// Double-check if the user exists in the Redis cache
		initializers.RDB.HGetAll(initializers.RDB_CTX, utils.RedisConstants.CACHE_USER_KEY_PREFIX+idStr).Scan(&user)
		if user == (models.User{}) {
			// If the user is still not found, then fetch it from the database
			time.Sleep(100 * time.Millisecond) // Simulate a delay
			result := initializers.DB.Debug().First(&user, uint(userID))
			if result.Error != nil {
				// Cache Penetration Solution
				// Cache the empty result in Redis for a short time
				initializers.RDB.HSet(initializers.RDB_CTX, utils.RedisConstants.CACHE_USER_KEY_PREFIX+idStr, "is_empty", "1")
				initializers.RDB.Expire(initializers.RDB_CTX, utils.RedisConstants.CACHE_USER_KEY_PREFIX+idStr, utils.RedisConstants.CACHE_NULL_EXPIRE_TIME)
				return nil, errors.New("user not found")
			}
			// Finally, cache the user in Redis and set an expiration time
			initializers.RDB.HSet(initializers.RDB_CTX, utils.RedisConstants.CACHE_USER_KEY_PREFIX+idStr, &user)
			initializers.RDB.Expire(initializers.RDB_CTX, utils.RedisConstants.CACHE_USER_KEY_PREFIX+idStr, utils.RedisConstants.CACHE_USER_EXPIRE_TIME)
		}
		// Unlock the mutex
		utils.SimpleUnlock(utils.RedisConstants.MUTEX_USER_KEY_PREFIX + idStr)
	}
	return &user, nil
}

// *****************************************************

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
