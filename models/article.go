package models

import (
	"gorm.io/gorm"
)

// Status represents the status of an article.
const (
	Pending = iota
	Approved
	Rejected
)

// LikeCode represents the type of like/dislike action on an article.
const (
	Like    = 1
	Dislike = 2
)

type Article struct {
	gorm.Model
	Title    string
	Body     string
	AuthorID uint
	Status   uint `gorm:"default:0"`
	Likes    uint `gorm:"default:0"`
	Dislikes uint `gorm:"default:0"`
}

type Comment struct {
	gorm.Model
	Content   string
	AuthorID  uint
	Status    uint `gorm:"default:1"`
	ArticleID uint
}

func (Article) TableName() string {
	return "articles"
}

func (Comment) TableName() string {
	return "comments"
}
