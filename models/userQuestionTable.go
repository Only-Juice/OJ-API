package models

import "time"

type UserQuestionTable struct {
	ID        uint                 `gorm:"primaryKey" json:"id"`
	UQRID     uint                 `gorm:"not null" json:"uqr_id"`
	UQR       UserQuestionRelation `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"uqr"`
	Score     float64              `gorm:"not null" json:"score"`
	JudgeTime time.Time            `gorm:"not null;default:CURRENT_TIMESTAMP" json:"judge_time" example:"2006-01-02T15:04:05Z07:00" time_format:"RFC3339"`
	Message   string               `gorm:"not null" json:"message"`
	CreatedAt time.Time            `gorm:"autoCreateTime" json:"created_at"`
	Commit    string               `gorm:"size:150;not null" json:"commit"`
}
