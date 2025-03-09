package models

import "time"

type UserQuestionTable struct {
	ID        uint                 `gorm:"primaryKey"`
	UQRID     uint                 `gorm:"not null"`
	UQR       UserQuestionRelation `gorm:"foreignKey:UQRID"`
	Score     float64              `gorm:"not null"`
	JudgeTime time.Time            `gorm:"not null"`
	Message   string
}
