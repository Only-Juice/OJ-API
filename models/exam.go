package models

import "time"

type Exam struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	OwnerID     uint      `gorm:"not null" json:"owner_id"`
	Owner       User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"owner"`
	Title       string    `gorm:"not null;size:50" json:"title"`
	Description string    `gorm:"size:500" json:"description"`
	StartTime   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"start_time" example:"2006-01-02T15:04:05Z07:00" time_format:"RFC3339"`
	EndTime     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"end_time" example:"2006-01-02T15:04:05Z07:00" time_format:"RFC3339"`
}
