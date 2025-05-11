package models

type Subscribe struct {
	ID       uint `gorm:"primaryKey"`
	AuthorID uint
	ReaderID uint
}

func (Subscribe) TableName() string {
	return "subscribes"
}
