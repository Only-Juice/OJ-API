package models

import "time"

type Question struct {
	ID          uint      `gorm:"primaryKey"`
	Title       string    `gorm:"size:100;not null"`
	Description string    `gorm:"size:5000;not null"`
	GitRepoURL  string    `gorm:"size:250"`
	StartTime   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	EndTime     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}
