package tasks

import (
	"auth/initializers"
	"auth/models"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// SeckillTask represents a task for processing a seckill operation.
type SeckillTask struct {
	ReaderID   uint
	AuthorID   uint
	DiscountID uint
	Credits    uint
}

// SeckillTaskQueue is a channel used to queue seckill tasks for processing.
var SeckillTaskQueue = make(chan SeckillTask, 100)

// InitSeckillProcessor initializes the seckill task processor.
func InitSeckillProcessor() {
	go func() {
		for task := range SeckillTaskQueue {
			err := processSeckillTask(task)
			if err != nil {
				// Here you would implement the logic to handle the error.
				// This could include logging the error, or retrying the task.

				fmt.Printf("Failed to process seckill task for ReaderID %d, AuthorID %d, DiscountID %d: %v", task.ReaderID, task.AuthorID, task.DiscountID, err)
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

// AddSeckillTask adds a seckill task to the queue.
func AddSeckillTask(readerID, authorID, discountID, credits uint) {
	SeckillTaskQueue <- SeckillTask{
		ReaderID:   readerID,
		AuthorID:   authorID,
		DiscountID: discountID,
		Credits:    credits,
	}
}
