package models

import "time"

type Exam struct {
	ID          uint      `gorm:"primarykey"`
	OwnerID     uint      `gorm:"not null"`
	Owner       User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Title       string    `gorm:"not null;size:50"`
	Description string    `gorm:"size:500"`
	StartTime   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	EndTime     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}
