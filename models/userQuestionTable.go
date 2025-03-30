package models

import "time"

type UserQuestionTable struct {
	ID        uint                 `gorm:"primaryKey"`
	UQRID     uint                 `gorm:"not null"`
	UQR       UserQuestionRelation `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Score     float64              `gorm:"not null"`
	JudgeTime time.Time            `gorm:"not null"`
	Message   string
}
