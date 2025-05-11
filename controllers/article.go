package controllers

import (
	"auth/initializers"
	"auth/models"
	"auth/utils"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FetchArticles retrieves all articles.
func FetchArticles(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the pagination parameters
	params := utils.GetPaginationParams(c)

	// Get the total number of articles
	var total int64
	result := initializers.DB.Model(&models.Article{}).Where("status = ?", models.Approved).Count(&total)
	if result.Error != nil {
		panic("Failed to get the total number of articles")
	}

	// Get the articles from the database
	// Use JOIN to get the author's email
	var articles []map[string]interface{}
	result = initializers.DB.Model(&models.Article{}).
		Joins("JOIN users ON articles.author_id = users.id").
		Select("articles.id as id", "articles.title as title", "articles.body as body", "users.email as author").
		Where("articles.status = ?", models.Approved).
		Order("articles.created_at DESC").
		Offset(params.Offset).Limit(params.PageSize).Find(&articles)
	if result.Error != nil {
		panic("Failed to get the articles from the database")
	}

	// Convert the author's email to a string
	for i := range articles {
		if email, ok := articles[i]["author"].([]byte); ok {
			articles[i]["author"] = string(email)
		}
	}

	// Get the pagination result
	pagination := utils.GetPaginationResult(params, len(articles), total)

	// Return a success response
	c.JSON(http.StatusOK, gin.H{
		"message":    "Articles retrieved successfully",
		"articles":   articles,
		"pagination": pagination,
	})
}

// // FetchUserArticles retrieves all articles of a user.
// func FetchUserArticles(c *gin.Context) {
// 	defer func() {
// 		if err := recover(); err != nil {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": err})
// 		}
// 	}()
// 	// Get the id off the path parameter
// 	id, ok := c.Params.Get("id")
// 	if !ok {
// 		panic("ID is required")
// 	}
// 	userID, err := strconv.Atoi(id)
// 	if err != nil {
// 		panic("Invalid ID: Type error")
// 	}
// 	// Get the pagination parameters
// 	params := utils.GetPaginationParams(c)
// 	// Get the total number of articles of the user
// 	var total int64
// 	result := initializers.DB.Model(&models.Article{}).Where("author_id = ? AND status = ?", uint(userID), models.Approved).Count(&total)
// 	if result.Error != nil {
// 		panic("Failed to get the total number of articles")
// 	}
// 	// Get the articles from the database
// 	var articles []map[string]interface{}
// 	result = initializers.DB.Model(&models.Article{}).
// 		Select("id", "title", "body", "likes", "dislikes").
// 		Where("author_id = ? AND status = ?", uint(userID), models.Approved).
// 		Order("created_at DESC").
// 		Offset(params.Offset).Limit(params.PageSize).Find(&articles)
// 	if result.Error != nil {
// 		panic("Failed to get the articles from the database")
// 	}
// 	// Get the comments for each article
// 	for i := range articles {
// 		var comments []map[string]interface{}
// 		result := initializers.DB.Model(&models.Comment{}).
// 			Joins("JOIN users ON comments.author_id = users.id").
// 			Select("comments.id as id", "comments.content as content", "users.email as author").
// 			Where("comments.article_id = ? AND comments.status = ?", articles[i]["id"], models.Approved).
// 			Order("comments.created_at DESC").
// 			Find(&comments)
// 		if result.Error != nil {
// 			panic("Failed to get the comments from the database")
// 		}
// 		for j := range comments {
// 			if email, ok := comments[j]["author"].([]byte); ok {
// 				comments[j]["author"] = string(email)
// 			}
// 		}
// 		articles[i]["comments"] = comments
// 	}
// 	// Get the pagination result
// 	pagination := utils.GetPaginationResult(params, len(articles), total)
// 	// Return a success response
// 	c.JSON(http.StatusOK, gin.H{
// 		"message":    "Articles retrieved successfully",
// 		"articles":   articles,
// 		"pagination": pagination,
// 	})
// }

// FetchUserArticles retrieves all articles of a user.
func FetchUserArticles(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the user off the context
	curUser, _ := c.Get("user")
	curUserID := curUser.(models.User).ID

	// Get the id off the path parameter
	id, ok := c.Params.Get("id")
	if !ok {
		panic("ID is required")
	}
	userID, err := strconv.Atoi(id)
	if err != nil {
		panic("Invalid ID: Type error")
	}

	// First check if the user is allowed to view the articles
	// (1) Check if the user has subscribed to the author
	// (2) Check if the user is the author himself/herself
	result := initializers.DB.Where("reader_id = ? AND author_id = ?", curUserID, userID).First(&models.Subscribe{})
	if result.RowsAffected == 0 && curUserID != uint(userID) {
		panic("Not allowed to view the user's articles")
	}

	// Get the pagination parameters
	params := utils.GetPaginationParams(c)

	// Get the total number of articles of the user
	var total int64
	result = initializers.DB.Model(&models.Article{}).Where("author_id = ? AND status = ?", uint(userID), models.Approved).Count(&total)
	if result.Error != nil {
		panic("Failed to get the total number of articles")
	}

	// Get the articles from the database
	var articles []map[string]interface{}
	result = initializers.DB.Model(&models.Article{}).
		Select("id", "title", "body", "likes", "dislikes").
		Where("author_id = ? AND status = ?", uint(userID), models.Approved).
		Order("created_at DESC").
		Offset(params.Offset).Limit(params.PageSize).Find(&articles)
	if result.Error != nil {
		panic("Failed to get the articles from the database")
	}

	// We need to avoid K+1 select problem
	// Prepare the article IDs
	var articleIDs []uint
	for _, article := range articles {
		articleIDs = append(articleIDs, article["id"].(uint))
	}
	// Get all the comments for the articles
	var comments []map[string]interface{}
	result = initializers.DB.Model(&models.Comment{}).
		Joins("JOIN users ON comments.author_id = users.id").
		Select("comments.id as id", "comments.content as content", "users.email as author", "comments.article_id as article_id").
		Where("comments.article_id IN (?) AND comments.status = ?", articleIDs, models.Approved).
		Order("comments.created_at DESC").
		Find(&comments)
	if result.Error != nil {
		panic("Failed to get the comments from the database")
	}
	// Convert the author's email to a string
	for i := range comments {
		if email, ok := comments[i]["author"].([]byte); ok {
			comments[i]["author"] = string(email)
		}
	}
	// Group the comments by article ID
	commentsByArticleID := make(map[uint][]map[string]interface{})
	for _, comment := range comments {
		articleID := comment["article_id"].(uint)
		commentsByArticleID[articleID] = append(commentsByArticleID[articleID], comment)
	}
	// Map the comments to the articles
	for i := range articles {
		articleID := articles[i]["id"].(uint)
		articles[i]["comments"] = commentsByArticleID[articleID]
	}

	// Get the pagination result
	pagination := utils.GetPaginationResult(params, len(articles), total)

	// Return a success response
	c.JSON(http.StatusOK, gin.H{
		"message":    "Articles retrieved successfully",
		"articles":   articles,
		"pagination": pagination,
	})
}

// PostArticle posts an article of the current user.
func PostArticle(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			initializers.LOGGER.Error("PostArticle Failed", "error", err, "sub", utils.GetSubInfo(c), "params", c.MustGet("params"))
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

	// Check if the user exists in the database and get the email
	var email string
	result := initializers.DB.Model(&models.User{}).Select("email").Where("id = ?", userID).Find(&email)
	if result.Error != nil || result.RowsAffected == 0 {
		panic("Failed to post article")
	}

	// Check if the user is an admin
	var status uint
	if ok, _ := initializers.E.HasGroupingPolicy(email, "admin"); ok {
		status = models.Approved
	} else {
		status = models.Pending
	}

	// Get the title and body off the request
	var body struct {
		Title string `json:"title" binding:"required"`
		Body  string `json:"body" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		panic("Both title and body are required")
	}

	// Prepare the article object
	article := models.Article{
		Title:    body.Title,
		Body:     body.Body,
		AuthorID: userID,
		Status:   status,
	}

	// Start a transaction to ensure atomicity
	err := initializers.DB.Transaction(func(tx *gorm.DB) error {
		// Create the article in the database
		result = tx.Create(&article)
		if result.Error != nil {
			return errors.New("failed to post article")
		}

		// Add 10 credits to the user
		result = tx.Model(&models.User{}).Where("id = ?", userID).Update("credits", gorm.Expr("credits + 10"))
		if result.Error != nil {
			return errors.New("failed to add credits to the user")
		}

		return nil
	})
	if err != nil {
		panic(err.Error())
	}

	// [Prepare the object and data information for logging]
	objInfo := utils.ObjInfo{
		Op:    utils.OpCreate,
		Table: models.Article{}.TableName(),
		ID:    article.ID,
	}
	dataInfo := utils.DataInfo{
		OldData: nil,
		NewData: map[string]interface{}{
			"title": article.Title,
			"body":  article.Body,
		},
	}

	// Return a success response
	message := "Article posted successfully"
	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
	initializers.LOGGER.Info(message, "sub", utils.GetSubInfo(c), "obj", objInfo, "data", dataInfo)
}

// RemoveArticle removes an article of the current user.
func RemoveArticle(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			initializers.LOGGER.Error("RemoveArticle Failed", "error", err, "sub", utils.GetSubInfo(c), "params", utils.GetParsedQuery(c))
		}
	}()

	// Get the user off the context
	user, _ := c.Get("user")
	userID := user.(models.User).ID

	// Check if the user exists in the database
	result := initializers.DB.First(&models.User{}, userID)
	if result.Error != nil {
		panic("Failed to remove article")
	}

	// Get the article ID off the query string
	id, ok := c.GetQuery("id")
	if !ok {
		panic("ID is required")
	}
	articleID, err := strconv.Atoi(id)
	if err != nil {
		panic("Invalid ID: Type error")
	}

	// [Get the article from the database]
	var article models.Article
	initializers.DB.First(&article, uint(articleID))

	// Delete the article from the database
	result = initializers.DB.Where("id = ? AND author_id = ?", uint(articleID), userID).Delete(&models.Article{})
	if result.Error != nil || result.RowsAffected == 0 {
		panic("Failed to remove article")
	}

	// Delete the comments on the article from the database
	result = initializers.DB.Where("article_id = ?", uint(articleID)).Delete(&models.Comment{})
	if result.Error != nil {
		panic("Failed to remove comments on the article")
	}

	// [Prepare the object and data information for logging]
	objInfo := utils.ObjInfo{
		Op:    utils.OpDelete,
		Table: models.Article{}.TableName(),
		ID:    article.ID,
	}
	dataInfo := utils.DataInfo{
		OldData: map[string]interface{}{
			"title": article.Title,
			"body":  article.Body,
		},
		NewData: nil,
	}

	// Return a success response
	message := "Article removed successfully"
	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
	initializers.LOGGER.Info(message, "sub", utils.GetSubInfo(c), "obj", objInfo, "data", dataInfo)
}

// LikeArticle likes or dislikes an article.
func LikeArticle(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the article ID and like code off the request
	var body struct {
		ID       uint `json:"id" binding:"required"`
		LikeCode uint `json:"likecode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		panic("Both id and like status are required")
	}

	// Update the article in the database
	switch body.LikeCode {
	case models.Like:
		result := initializers.DB.Model(&models.Article{}).Where("id = ?", body.ID).Update("likes", gorm.Expr("likes + 1"))
		if result.Error != nil || result.RowsAffected == 0 {
			panic("Failed to like the article")
		}
	case models.Dislike:
		result := initializers.DB.Model(&models.Article{}).Where("id = ?", body.ID).Update("dislikes", gorm.Expr("dislikes + 1"))
		if result.Error != nil || result.RowsAffected == 0 {
			panic("Failed to dislike the article")
		}
	default:
		panic("Invalid like code: Must be either Like(1) or Dislike(2)")
	}

	// Return a success response
	c.JSON(http.StatusOK, gin.H{
		"message": "Article liked/disliked successfully",
	})
}

// PostComment posts a comment on an article.
func PostComment(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			initializers.LOGGER.Error("PostComment Failed", "error", err, "sub", utils.GetSubInfo(c), "params", c.MustGet("params"))
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

	// Check if the user exists in the database
	result := initializers.DB.First(&models.User{}, userID)
	if result.Error != nil {
		panic("Failed to post comment")
	}

	// Get the article ID and content off the request
	var body struct {
		ID      uint   `json:"id" binding:"required"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		panic("Both id and content are required")
	}

	// Prepare the comment object
	comment := models.Comment{
		Content:   body.Content,
		AuthorID:  userID,
		ArticleID: body.ID,
	}

	// Start a transaction to ensure atomicity
	err := initializers.DB.Transaction(func(tx *gorm.DB) error {
		// Create the comment in the database
		result = tx.Create(&comment)
		if result.Error != nil {
			return errors.New("failed to post comment")
		}

		// Add 5 credits to the user
		result = tx.Model(&models.User{}).Where("id = ?", userID).Update("credits", gorm.Expr("credits + 5"))
		if result.Error != nil {
			return errors.New("failed to add credits to the user")
		}

		return nil
	})
	if err != nil {
		panic(err.Error())
	}

	// [Prepare the object and data information for logging]
	objInfo := utils.ObjInfo{
		Op:    utils.OpCreate,
		Table: models.Comment{}.TableName(),
		ID:    comment.ID,
	}
	dataInfo := utils.DataInfo{
		OldData: nil,
		NewData: map[string]interface{}{
			"content": comment.Content,
		},
	}

	// Return a success response
	message := "Comment posted successfully"
	c.JSON(http.StatusOK, gin.H{
		"message": "Comment posted successfully",
	})
	initializers.LOGGER.Info(message, "sub", utils.GetSubInfo(c), "obj", objInfo, "data", dataInfo)
}

// RemoveComment removes a comment on an article.
func RemoveComment(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			initializers.LOGGER.Error("RemoveComment Failed", "error", err, "sub", utils.GetSubInfo(c), "params", utils.GetParsedQuery(c))
		}
	}()

	// Get the user off the context
	user, _ := c.Get("user")
	userID := user.(models.User).ID

	// Check if the user exists in the database
	result := initializers.DB.First(&models.User{}, userID)
	if result.Error != nil {
		panic("Failed to remove comment")
	}

	// Get the comment ID off the query string
	id, ok := c.GetQuery("id")
	if !ok {
		panic("ID is required")
	}
	commentID, err := strconv.Atoi(id)
	if err != nil {
		panic("Invalid ID: Type error")
	}

	// [Get the comment from the database]
	var comment models.Comment
	initializers.DB.First(&comment, uint(commentID))

	// Delete the comment from the database
	result = initializers.DB.Where("id = ? AND author_id = ?", uint(commentID), userID).Delete(&models.Comment{})
	if result.Error != nil || result.RowsAffected == 0 {
		panic("Failed to remove comment")
	}

	// [Prepare the object and data information for logging]
	objInfo := utils.ObjInfo{
		Op:    utils.OpDelete,
		Table: models.Comment{}.TableName(),
		ID:    comment.ID,
	}
	dataInfo := utils.DataInfo{
		OldData: map[string]interface{}{
			"content": comment.Content,
		},
		NewData: nil,
	}

	// Return a success response
	message := "Comment removed successfully"
	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
	initializers.LOGGER.Info(message, "sub", utils.GetSubInfo(c), "obj", objInfo, "data", dataInfo)
}

// GetArticles is an Admin API Endpoint that retrieves all articles.
func GetArticles(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the pagination parameters
	params := utils.GetPaginationParams(c)

	// Get the total number of articles
	var total int64
	result := initializers.DB.Model(&models.Article{}).Count(&total)
	if result.Error != nil {
		panic("Failed to get the total number of articles")
	}

	// Get the articles from the database
	var articles []map[string]interface{}
	result = initializers.DB.Model(&models.Article{}).Offset(params.Offset).Limit(params.PageSize).Find(&articles)
	if result.Error != nil {
		panic("Failed to get the articles from the database")
	}

	// Get the pagination result
	pagination := utils.GetPaginationResult(params, len(articles), total)

	// Return a success response with the articles
	c.JSON(http.StatusOK, gin.H{
		"message":    "Articles retrieved successfully",
		"articles":   articles,
		"pagination": pagination,
	})
}

// SetArticleStatus is an Admin API Endpoint that sets the status of an article.
func SetArticleStatus(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			initializers.LOGGER.Error("SetArticleStatus Failed", "error", err, "sub", utils.GetSubInfo(c), "params", c.MustGet("params"))
		}
	}()

	// [Get the filtered parsed body and save it to the context]
	rawBody := utils.GetRawBody(c)
	parsedBody := utils.GetParsedBody(rawBody)
	utils.BlurMap(parsedBody, "password")
	c.Set("params", parsedBody)

	// Get the article ID and status off the request
	var body struct {
		ID     uint `json:"id" binding:"required"`
		Status uint `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		panic("Both id and status are required")
	}

	// Validate the status
	if body.Status != models.Pending && body.Status != models.Approved && body.Status != models.Rejected {
		panic("Invalid status: Must be one of Pending(0), Approved(1), or Rejected(2)")
	}

	// [Get the article status from the database]
	var status uint
	initializers.DB.Model(&models.Article{}).Select("status").Where("id = ?", body.ID).Find(&status)

	// Set the status of the article in the database
	result := initializers.DB.Model(&models.Article{}).Where("id = ?", body.ID).Update("status", body.Status)
	if result.Error != nil || result.RowsAffected == 0 {
		panic("Failed to set the status of the article")
	}

	// [Prepare the object and data information for logging]
	objInfo := utils.ObjInfo{
		Op:    utils.OpUpdate,
		Table: models.Article{}.TableName(),
		ID:    body.ID,
	}
	dataInfo := utils.DataInfo{
		OldData: map[string]interface{}{
			"status": status,
		},
		NewData: map[string]interface{}{
			"status": body.Status,
		},
	}

	// Return a success response
	message := "Article Status set successfully"
	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
	initializers.LOGGER.Info(message, "sub", utils.GetSubInfo(c), "obj", objInfo, "data", dataInfo)
}

// GetComments is an Admin API Endpoint that retrieves all comments.
func GetComments(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the pagination parameters
	params := utils.GetPaginationParams(c)

	// Get the total number of comments
	var total int64
	result := initializers.DB.Model(&models.Comment{}).Count(&total)
	if result.Error != nil {
		panic("Failed to get the total number of comments")
	}

	// Get the comments from the database
	var comments []map[string]interface{}
	result = initializers.DB.Model(&models.Comment{}).Offset(params.Offset).Limit(params.PageSize).Find(&comments)
	if result.Error != nil {
		panic("Failed to get the comments from the database")
	}

	// Get the pagination result
	pagination := utils.GetPaginationResult(params, len(comments), total)

	// Return a success response with the comments
	c.JSON(http.StatusOK, gin.H{
		"message":    "Comments retrieved successfully",
		"comments":   comments,
		"pagination": pagination,
	})
}

// SetCommentStatus is an Admin API Endpoint that sets the status of a comment.
func SetCommentStatus(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			initializers.LOGGER.Error("SetCommentStatus Failed", "error", err, "sub", utils.GetSubInfo(c), "params", c.MustGet("params"))
		}
	}()

	// [Get the filtered parsed body and save it to the context]
	rawBody := utils.GetRawBody(c)
	parsedBody := utils.GetParsedBody(rawBody)
	utils.BlurMap(parsedBody, "password")
	c.Set("params", parsedBody)

	// Get the comment ID and status off the request
	var body struct {
		ID     uint `json:"id" binding:"required"`
		Status uint `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		panic("Both id and status are required")
	}

	// Validate the status
	if body.Status != models.Approved && body.Status != models.Rejected {
		panic("Invalid status: Must be Approved(1) or Rejected(2)")
	}

	// [Get the comment status from the database]
	var status uint
	initializers.DB.Model(&models.Comment{}).Select("status").Where("id = ?", body.ID).Find(&status)

	// Set the status of the comment in the database
	result := initializers.DB.Model(&models.Comment{}).Where("id = ?", body.ID).Update("status", body.Status)
	if result.Error != nil || result.RowsAffected == 0 {
		panic("Failed to set the status of the comment")
	}

	// [Prepare the object and data information for logging]
	objInfo := utils.ObjInfo{
		Op:    utils.OpUpdate,
		Table: models.Comment{}.TableName(),
		ID:    body.ID,
	}
	dataInfo := utils.DataInfo{
		OldData: map[string]interface{}{
			"status": status,
		},
		NewData: map[string]interface{}{
			"status": body.Status,
		},
	}

	// Return a success response
	message := "Comment Status set successfully"
	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
	initializers.LOGGER.Info(message, "sub", utils.GetSubInfo(c), "obj", objInfo, "data", dataInfo)
}
