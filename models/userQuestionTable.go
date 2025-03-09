package models

import "time"

type UserQuestionTable struct {
	ID       uint                 `gorm:"primaryKey"`
	UQRID    uint                 `gorm:"not null"`
	UQR      UserQuestionRelation `gorm:"foreignKey:UQRID"`
	Score    float32              `gorm:"not null"`
	PushTime time.Time            `gorm:"not null"`
	TXTPASS  string               `gorm:"size:4000"`
}
