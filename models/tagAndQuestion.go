package models

type TagAndQuestion struct {
	ID         uint     `gorm:"primaryKey"`
	TagID      uint     `gorm:"not null"`
	Tag        Tag      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	QuestionID uint     `gorm:"not null"`
	Question   Question `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}
