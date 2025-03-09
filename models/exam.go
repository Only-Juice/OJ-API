package models

import "time"

type Exam struct {
	ID          uint      `gorm:"primarykey"`
	UserID      uint      `gorm:"not null"`
	User        User      `gorm:"foreignKey:UserID"`
	Title       string    `gorm:"not null;size:50"`
	Description string    `gorm:"size:500"`
	StartTime   time.Time `gorm:"not null"`
	EndTime     time.Time `gorm:"not null"`
}
