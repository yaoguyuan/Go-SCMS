package controllers

import (
	"auth/initializers"
	"auth/models"
	"auth/utils"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Subscribe allows a reader to subscribe to an author.
func Subscribe(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the reader ID off the context
	reader, _ := c.Get("user")
	readerID := reader.(models.User).ID
	readerCredits := reader.(models.User).Credits

	// Get the author ID off the body
	var body struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		panic("Failed to get the author ID from the request body")
	}
	authorID := body.ID

	// Check if the author exists
	var author models.User
	result := initializers.DB.First(&author, authorID)
	if result.Error != nil {
		panic("Failed to find the author")
	}

	// Check if the reader has already subscribed or is the author
	if readerID == authorID {
		panic("You cannot subscribe to yourself")
	}
	result = initializers.DB.Where("reader_id = ? AND author_id = ?", readerID, authorID).First(&models.Subscribe{})
	if result.RowsAffected > 0 {
		panic("You have already subscribed to this author")
	}

	// Check if the reader has enough credits
	if readerCredits < author.Subfee {
		panic("You do not have enough credits.")
	}

	// Start a transaction to ensure atomicity
	err := initializers.DB.Transaction(func(tx *gorm.DB) error {
		// Create a new subscription
		result := tx.Create(&models.Subscribe{
			AuthorID: authorID,
			ReaderID: readerID,
		})
		if result.Error != nil {
			return errors.New("failed to create a subscription")
		}

		// Transfer credits from the reader to the author
		result = tx.Model(&models.User{}).Where("id = ?", readerID).Update("credits", readerCredits-author.Subfee)
		if result.Error != nil {
			return errors.New("failed to deduct credits from the reader")
		}
		result = tx.Model(&models.User{}).Where("id = ?", authorID).Update("credits", author.Credits+author.Subfee)
		if result.Error != nil {
			return errors.New("failed to add credits to the author")
		}

		return nil
	})
	if err != nil {
		panic(err.Error())
	}

	// Return a success response
	c.JSON(http.StatusOK, gin.H{"message": "Subscription successful"})
}

// FetchUserDiscounts fetches the discounts of an author.
func FetchUserDiscounts(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the user ID off the path
	id, ok := c.Params.Get("id")
	if !ok {
		panic("ID is required")
	}
	userID, err := strconv.Atoi(id)
	if err != nil {
		panic("Invalid ID: Type error")
	}

	// Get the discounts of the user from the database
	var discounts []models.Discount
	result := initializers.DB.Where("author_id = ? AND status = ?", uint(userID), models.Approved).Find(&discounts)
	if result.Error != nil || result.RowsAffected == 0 {
		panic("Failed to find discounts for the author")
	}

	// Filter the discounts to only include Discount, Stock, BeginTime, EndTime
	var selectedDiscounts []map[string]interface{}
	for _, discount := range discounts {
		selectedDiscounts = append(selectedDiscounts, map[string]interface{}{
			"id":        discount.ID,
			"discount":  discount.Discount,
			"stock":     discount.Stock,
			"beginTime": discount.BeginTime.Format(time.RFC3339),
			"endTime":   discount.EndTime.Format(time.RFC3339),
		})
	}

	// Return a success response with the discounts
	c.JSON(http.StatusOK, gin.H{
		"message":   "Discounts fetched successfully",
		"discounts": selectedDiscounts,
	})
}

// PostDiscount creates a new discount for the user.
func PostDiscount(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the user ID off the context
	user, _ := c.Get("user")
	userID := user.(models.User).ID
	userSubfee := user.(models.User).Subfee
	if userSubfee == 0 {
		panic("You haven't set a valid subscription fee yet")
	}

	// Get the paramters off the body: Discount, Stock, Duration
	var body struct {
		Discount uint `json:"discount" binding:"required"` // Discount percentage
		Stock    uint `json:"stock" binding:"required"`    // Stock quantity
		Duration uint `json:"duration" binding:"required"` // Duration in hours
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		panic("Failed to get the parameters from the request body: discount, stock, duration")
	}

	// Validate the parameters
	// (1) Discount should be between 0 and 100
	if body.Discount <= 0 || body.Discount >= 100 {
		panic("Invalid discount percentage: it should be between 0 and 100")
	}
	// (2) Stock should be between 0 and 1000
	if body.Stock <= 0 || body.Stock > 1000 {
		panic("Invalid stock quantity: it should be between 0 and 1000")
	}
	// (3) Duration should be between 0 and 24
	if body.Duration <= 0 || body.Duration > 24 {
		panic("Invalid duration in hours: it should be between 0 and 24")
	}

	// Create a new discount to the database
	// (1) Calculate the discount price
	discountPrice := userSubfee * body.Discount / 100
	// (2) Calculate the begin and end time
	beginTime := time.Now()
	endTime := time.Now().Add(time.Duration(body.Duration) * time.Hour)
	// (3) Create the discount
	discount := models.Discount{
		AuthorID:  userID,
		Discount:  discountPrice,
		Stock:     body.Stock,
		BeginTime: beginTime,
		EndTime:   endTime,
		Status:    models.Approved,
	}
	result := initializers.DB.Create(&discount)
	if result.Error != nil {
		panic("Failed to create a discount")
	}

	// Create a Redis key for the discount stock
	key := utils.RedisConstants.SECKILL_STOCK_KEY_PREFIX + strconv.Itoa(int(discount.ID))
	ttl := endTime.Sub(beginTime)
	initializers.RDB.Set(initializers.RDB_CTX, key, body.Stock, ttl)

	// Return a success response
	c.JSON(http.StatusOK, gin.H{
		"message": "Discount created successfully",
	})
}

// Seckill allows a reader to purchase a discount.
func Seckill(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}
	}()

	// Get the reader ID off the context
	_reader_, _ := c.Get("user")
	reader := _reader_.(models.User)

	// Get the discount ID off the body
	var body struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		panic("Failed to get the discount ID from the request body")
	}

	// Check if the discount exists and is available
	var discount models.Discount
	result := initializers.DB.First(&discount, body.ID)
	if result.Error != nil {
		panic("Failed to find the discount")
	}

	// Call the seckill function
	err := seckill(reader, discount)
	if err != nil {
		panic(err.Error())
	}

	// Return a success response
	c.JSON(http.StatusOK, gin.H{
		"message": "Discount purchased successfully",
	})
}

// *** Using Redis for Seckill Operation ***
func seckill(reader models.User, discount models.Discount) error {
	// (1) Check if the reader is the author himself/herself
	if reader.ID == discount.AuthorID {
		return errors.New("failed to purchase the discount: you cannot purchase your own discount")
	}
	// (2) Check if the reader has enough credits to purchase the discount
	if reader.Credits < discount.Discount {
		return errors.New("failed to purchase the discount: you do not have enough credits")
	}

	// Call SeckillScript to perform the seckill operation atomically
	// ret, _ := utils.SeckillScript.Run(initializers.RDB_CTX, initializers.RDB, []string{}, discount.ID, reader.ID).Int()
	ret, _ := utils.SeckillScript.Run(initializers.RDB_CTX, initializers.RDB, []string{}, reader.ID, discount.AuthorID, discount.ID, discount.Discount).Int()
	if ret == 1 {
		return errors.New("failed to purchase the discount: already subscribed")
	} else if ret == 2 {
		return errors.New("failed to purchase the discount: not within valid time range")
	} else if ret == 3 {
		return errors.New("failed to purchase the discount: no stock available")
	}

	// Asynchronously process the seckill task
	// tasks.AddSeckillTask(reader.ID, discount.AuthorID, discount.ID, discount.Discount)

	return nil
}

// *****************************************

/*
// **************** Using Optimistic Lock to prevent overselling *****************
// *** Using Pessimistic Lock to prevent one person from buying multiple times ***
func seckill(reader models.User, discount models.Discount, requestID string) error {
	// (1) Check if the reader is the author himself/herself
	if reader.ID == discount.AuthorID {
		return errors.New("failed to purchase the discount: you cannot purchase your own discount")
	}

	// 使用悲观锁实现一人一单
	key := utils.RedisConstants.LOCK_ORDER_KEY_PREFIX + strconv.Itoa(int(reader.ID)) + "-" + strconv.Itoa(int(discount.ID))
	if !utils.TryLock(key, requestID, utils.RedisConstants.LOCK_ORDER_EXPIRE_TIME) {
		return errors.New("failed to purchase the discount: you are already processing a purchase")
	}
	defer utils.Unlock(key, requestID)

	// (2) Check if the reader has already subscribed to the author
	result := initializers.DB.Where("reader_id = ? AND author_id = ?", reader.ID, discount.AuthorID).First(&models.Subscribe{})
	if result.RowsAffected > 0 {
		return errors.New("failed to purchase the discount: you have already subscribed to this author")
	}
	// (3) Check if the discount is within the valid time range
	if time.Now().Before(discount.BeginTime) || time.Now().After(discount.EndTime) {
		return errors.New("failed to purchase the discount: it is not within the valid time range")
	}
	// (4) Check if the discount stock is available
	if discount.Stock <= 0 {
		return errors.New("failed to purchase the discount: stock is not available")
	}

	// Check if the reader has enough credits
	if reader.Credits < discount.Discount {
		return errors.New("failed to purchase the discount: you do not have enough credits")
	}

	// Start a transaction to ensure atomicity
	err := initializers.DB.Transaction(func(tx *gorm.DB) error {
		// Create a new subscription
		result = tx.Create(&models.Subscribe{
			AuthorID: discount.AuthorID,
			ReaderID: reader.ID,
		})
		if result.Error != nil {
			return errors.New("failed to create a subscription")
		}

		// Transfer credits from the reader to the author
		result = tx.Model(&models.User{}).Where("id = ?", reader.ID).Update("credits", reader.Credits-discount.Discount)
		if result.Error != nil {
			return errors.New("failed to deduct credits from the reader")
		}
		result = tx.Model(&models.User{}).Where("id = ?", discount.AuthorID).Update("credits", gorm.Expr("credits + ?", discount.Discount))
		if result.Error != nil {
			return errors.New("failed to add credits to the author")
		}

		// Update the discount stock
		// 使用乐观锁解决超卖问题
		result = tx.Model(&models.Discount{}).Where("id = ? AND stock > 0", discount.ID).Update("stock", gorm.Expr("stock - 1"))
		if result.Error != nil || result.RowsAffected == 0 {
			return errors.New("failed to decrement the discount stock")
		}

		return nil
	})

	return err
}

// *******************************************************************************
*/
