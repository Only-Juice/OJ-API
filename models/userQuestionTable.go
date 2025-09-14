package models

import "time"

type UserQuestionTable struct {
	ID        uint                 `gorm:"primaryKey" json:"id"`
	UQRID     uint                 `gorm:"not null;index:idx_uqt_uqr_score_created,priority:1" json:"uqr_id"`
	UQR       UserQuestionRelation `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"uqr"`
	Score     float64              `gorm:"not null;index:idx_uqt_uqr_score_created,priority:2" json:"score"`
	JudgeTime time.Time            `gorm:"not null;default:CURRENT_TIMESTAMP" json:"judge_time" example:"2006-01-02T15:04:05Z07:00" time_format:"RFC3339"`
	Message   string               `gorm:"not null" json:"message"`
	Commit    string               `gorm:"size:150;not null;default:''" json:"commit"`
	CreatedAt time.Time            `gorm:"autoCreateTime;index:idx_uqt_uqr_score_created,priority:3" json:"created_at"`
}
