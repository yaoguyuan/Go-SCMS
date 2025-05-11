package models

import "time"

type Discount struct {
	ID        uint `gorm:"primaryKey"`
	AuthorID  uint
	Discount  uint
	Stock     uint
	BeginTime time.Time
	EndTime   time.Time
	Status    uint
}

func (Discount) TableName() string {
	return "discounts"
}
