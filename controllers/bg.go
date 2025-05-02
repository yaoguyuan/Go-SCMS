package controllers

import (
	"auth/initializers"
	"auth/models"
	"auth/utils"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetUsers is an Admin API Endpoint that retrieves all users' information.
func GetUsers(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the pagination parameters
	params := utils.GetPaginationParams(c)

	// Get the total number of users
	var total int64
	result := initializers.DB.Model(&models.User{}).Where("role = ?", "user").Count(&total)
	if result.Error != nil {
		panic("Failed to get the total number of users")
	}

	// Get the users from the database
	var users []map[string]interface{}
	result = initializers.DB.Model(&models.User{}).Where("role = ?", "user").Offset(params.Offset).Limit(params.PageSize).Find(&users)
	if result.Error != nil {
		panic("Failed to get the users from the database")
	}

	// Get the pagination result
	pagination := utils.GetPaginationResult(params, len(users), total)

	// Return a success response with the users
	c.JSON(http.StatusOK, gin.H{
		"message":    "Users retrieved successfully",
		"users":      users,
		"pagination": pagination,
	})
}

// DelUser is an Admin API Endpoint that deletes a user by ID.
func DelUser(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			initializers.LOGGER.Error("DelUser Failed", "error", err, "sub", utils.GetSubInfo(c), "params", utils.GetParsedQuery(c))
		}
	}()

	// Get the id off the query string
	id, ok := c.GetQuery("id")
	if !ok {
		panic("Failed to get id off the query string")
	}
	userID, err := strconv.Atoi(id)
	if err != nil {
		panic("Invalid id")
	}

	// Check if the user exists in the database and return the email
	var tmp map[string]interface{}
	result := initializers.DB.Model(&models.User{}).Select("email", "avatar").Where("id = ? AND role = ?", uint(userID), "user").First(&tmp)
	if result.Error != nil {
		panic("Failed to find user in the database")
	}

	// Delete the user from the database
	result = initializers.DB.Delete(&models.User{}, uint(userID))
	if result.Error != nil {
		panic("Failed to delete user from the database")
	}

	// Delete the user's articles from the database
	result = initializers.DB.Where("user_id = ?", uint(userID)).Delete(&models.Article{})
	if result.Error != nil {
		panic("Failed to delete user's articles from the database")
	}

	// Delete the user's comments from the database
	result = initializers.DB.Where("user_id = ?", uint(userID)).Delete(&models.Comment{})
	if result.Error != nil {
		panic("Failed to delete user's comments from the database")
	}

	// Delete the user from Casbin
	initializers.E.RemoveFilteredPolicy(0, tmp["email"].(string))
	initializers.E.RemoveFilteredGroupingPolicy(0, tmp["email"].(string))

	// Delete the user's avatar file
	if tmp["avatar"].(string) != initializers.AvatarDefault {
		filePath := filepath.Join(initializers.AvatarDir, tmp["avatar"].(string))
		err := os.Remove(filePath)
		if err != nil {
			panic("Failed to delete user's avatar file")
		}
	}

	// [Prepare the object and data information for logging]
	objInfo := utils.ObjInfo{
		Op:    utils.OpDelete,
		Table: models.User{}.TableName(),
		ID:    uint(userID),
	}
	dataInfo := utils.DataInfo{
		OldData: map[string]interface{}{
			"email": tmp["email"].(string),
		},
		NewData: nil,
	}

	// Return a success response
	message := "User deleted successfully"
	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
	initializers.LOGGER.Warn(message, "sub", utils.GetSubInfo(c), "obj", objInfo, "data", dataInfo)
}

// GetDenials is an Admin API Endpoint that retrieves all denials in Casbin.
func GetDenials(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the pagination parameters
	params := utils.GetPaginationParams(c)

	// Get the total number of denials
	allDenials, _ := initializers.E.GetFilteredPolicy(3, "deny")
	total := len(allDenials)

	// Get the denials for the current page
	var denials [][]string
	if params.Offset+params.PageSize > total {
		denials = allDenials[params.Offset:]
	} else {
		denials = allDenials[params.Offset : params.Offset+params.PageSize]
	}

	// Get the pagination result
	pagination := utils.GetPaginationResult(params, len(denials), int64(total))

	// Return a success response with the denials
	c.JSON(http.StatusOK, gin.H{
		"message":    "Denials retrieved successfully",
		"denials":    denials,
		"pagination": pagination,
	})
}

// AddDenial is an Admin API Endpoint that adds a denial in Casbin.
func AddDenial(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			initializers.LOGGER.Error("AddDenial Failed", "error", err, "sub", utils.GetSubInfo(c), "params", c.MustGet("params"))
		}
	}()

	// [Get the filtered parsed body and save it to the context]
	rawBody := utils.GetRawBody(c)
	parsedBody := utils.GetParsedBody(rawBody)
	utils.BlurMap(parsedBody, "password")
	c.Set("params", parsedBody)

	// Get the email, subpath, and method off the request body
	var body struct {
		Email   string `json:"email" binding:"required"`
		Subpath string `json:"subpath" binding:"required"`
		Method  string `json:"method" binding:"required"`
	}
	if err := c.ShouldBind(&body); err != nil {
		panic("Failed to get email, subpath, and method off the request body")
	}

	// Check if the user exists in the database
	result := initializers.DB.Where("email = ? AND role = ?", body.Email, "user").First(&models.User{})
	if result.Error != nil {
		panic("Failed to find user in the database")
	}

	// Check if the subpath and method are valid
	if !strings.HasPrefix(body.Subpath, "/") {
		panic("Invalid subpath")
	}
	path := "/api/ui" + body.Subpath
	methods := []string{"GET", "POST", "PUT", "DELETE", "ANY"}
	if !slices.Contains(methods, body.Method) {
		panic("Invalid method")
	}

	// Add the denial in Casbin
	ok, _ := initializers.E.AddPolicy(body.Email, path, body.Method, "deny")
	if !ok {
		panic("Failed to add denial in Casbin, which may already exist")
	}

	// Return a success response
	message := "Denial added successfully"
	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
	initializers.LOGGER.Warn(message, "sub", utils.GetSubInfo(c), "rule", body)
}

// DelDenial is an Admin API Endpoint that deletes a denial in Casbin.
func DelDenial(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			initializers.LOGGER.Error("DelDenial Failed", "error", err, "sub", utils.GetSubInfo(c), "params", utils.GetParsedQuery(c))
		}
	}()

	// Get the email, path, and method off the qury string
	email, ok1 := c.GetQuery("email")
	path, ok2 := c.GetQuery("path")
	method, ok3 := c.GetQuery("method")
	if !(ok1 && ok2 && ok3) {
		panic("Failed to get email, path, and method off the query string")
	}

	// Delete the denail in Casbin
	ok, _ := initializers.E.RemovePolicy(email, path, method, "deny")
	if !ok {
		panic("Failed to delete denial in Casbin, which may not exist")
	}

	// Return a success response
	message := "Denial deleted successfully"
	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
	initializers.LOGGER.Warn(message, "sub", utils.GetSubInfo(c), "rule", map[string]interface{}{
		"email":  email,
		"path":   path,
		"method": method,
	})
}

// DownloadLogFile is an Admin API Endpoint that downloads the log file.
func DownloadLogFile(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the log file path
	logFilePath := filepath.Join(initializers.LogFileDir, initializers.LogFileDefault)

	// Check if the log file exists
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		panic("Log file does not exist")
	}

	// Ensure log file download
	c.Header("Content-Disposition", "attachment; filename="+initializers.LogFileDefault)
	c.Header("Content-Type", "application/force-download")
	// Disable the browser cache
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	// Return the log file as a download
	c.File(logFilePath)
}
