package tasks

import (
	"auth/initializers"
	"auth/models"
	"auth/utils"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// SeckillTask represents a task for processing a seckill operation.
type SeckillTask struct {
	ReaderID   uint
	AuthorID   uint
	DiscountID uint
	Credits    uint
}

// InitSeckillProcessor initializes the seckill task processor.
func InitSeckillProcessor() {
	go func() {
		for {
			// (1) Read a new task
			// XREADGROUP GROUP g1 c1 COUNT 1 BLOCK 2000 STREAMS stream.orders >
			result, err := initializers.RDB.XReadGroup(initializers.RDB_CTX, &redis.XReadGroupArgs{
				Group:    "g1",
				Consumer: "c1",
				Count:    1,
				Block:    2000 * time.Millisecond,
				Streams:  []string{"stream.orders", ">"},
			}).Result()
			// If no new task is available, continue to the next iteration
			if err == redis.Nil {
				continue
			}

			// (2) Parse the task
			taskID := result[0].Messages[0].ID
			taskData := result[0].Messages[0].Values
			task := SeckillTask{
				ReaderID:   utils.StrToUint(taskData["readerID"].(string)),
				AuthorID:   utils.StrToUint(taskData["authorID"].(string)),
				DiscountID: utils.StrToUint(taskData["discountID"].(string)),
				Credits:    utils.StrToUint(taskData["credits"].(string)),
			}

			// (3) Process the task
			if err := processSeckillTask(task); err != nil {
				// If an error occurs, log it and process the task again in the pending-list
				fmt.Printf("Error processing task %v: %v\n", task, err)
				handlePendingList()
			} else {
				// If no error occurs, acknowledge the task to remove it from the pending-list
				// XACK stream.orders g1 <taskID>
				initializers.RDB.XAck(initializers.RDB_CTX, "stream.orders", "g1", taskID)
			}
		}
	}()
	fmt.Println("SeckillTaskProcessor running...")
}

// processSeckillTask processes a seckill task.
func processSeckillTask(task SeckillTask) error {
	// Here you would implement the logic to process the seckill task.
	// This might include creating a subscription, transferring credits, and updating the stock.

	err := initializers.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Create a subscription for the reader to the author
		result := tx.Create(&models.Subscribe{
			AuthorID: task.AuthorID,
			ReaderID: task.ReaderID,
		})
		if result.Error != nil {
			return errors.New("failed to create a subscription")
		}

		// 2. Transfer credits from the reader to the author
		result = tx.Model(&models.User{}).Where("id = ?", task.ReaderID).Update("credits", gorm.Expr("credits - ?", task.Credits))
		if result.Error != nil {
			return errors.New("failed to transfer credits from the reader")
		}
		result = tx.Model(&models.User{}).Where("id = ?", task.AuthorID).Update("credits", gorm.Expr("credits + ?", task.Credits))
		if result.Error != nil {
			return errors.New("failed to transfer credits to the author")
		}

		// 3. Update the stock for the discount
		result = tx.Model(&models.Discount{}).Where("id = ? AND stock > 0", task.DiscountID).Update("stock", gorm.Expr("stock - 1"))
		if result.Error != nil || result.RowsAffected == 0 {
			return errors.New("failed to update the stock")
		}

		return nil
	})

	return err
}

// handlePendingList handles the pending list.
func handlePendingList() {
	// Here you would implement the logic to handle the pending list.
	// This should be logically similar to the task processor.

	for {
		// (1) Read a new task
		// XREADGROUP GROUP g1 c1 COUNT 1 STREAMS stream.orders 0
		result, err := initializers.RDB.XReadGroup(initializers.RDB_CTX, &redis.XReadGroupArgs{
			Group:    "g1",
			Consumer: "c1",
			Count:    1,
			Streams:  []string{"stream.orders", "0"},
		}).Result()
		// If no task is in the pending-list, break the loop
		if err == redis.Nil {
			break
		}

		// (2) Parse the task
		taskID := result[0].Messages[0].ID
		taskData := result[0].Messages[0].Values
		task := SeckillTask{
			ReaderID:   utils.StrToUint(taskData["readerID"].(string)),
			AuthorID:   utils.StrToUint(taskData["authorID"].(string)),
			DiscountID: utils.StrToUint(taskData["discountID"].(string)),
			Credits:    utils.StrToUint(taskData["credits"].(string)),
		}

		// (3) Process the task
		if err := processSeckillTask(task); err != nil {
			// If an error occurs, log it and process the task again in the pending-list
			fmt.Printf("Error processing task %v: %v\n", task, err)
		} else {
			// If no error occurs, acknowledge the task to remove it from the pending-list
			// XACK stream.orders g1 <taskID>
			initializers.RDB.XAck(initializers.RDB_CTX, "stream.orders", "g1", taskID)
		}
	}
}
