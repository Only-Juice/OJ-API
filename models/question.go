package models

import "time"

type Question struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"size:100;not null" json:"title"`
	Description string    `gorm:"size:5000;not null" json:"description"`
	GitRepoURL  string    `gorm:"size:250" json:"git_repo_url"`
	StartTime   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"start_time" example:"2006-01-02T15:04:05Z07:00" time_format:"RFC3339"`
	EndTime     time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"end_time" example:"2006-01-02T15:04:05Z07:00" time_format:"RFC3339"`
	IsActive    bool      `gorm:"not null;default:true;index:idx_questions_is_active" json:"is_active"`
}
