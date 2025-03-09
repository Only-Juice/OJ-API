package models

type TagAndQuestion struct {
	ID         uint     `gorm:"primaryKey"`
	TagID      uint     `gorm:"not null"`
	Tag        Tag      `gorm:"foreignKey:TagID"`
	QuestionID uint     `gorm:"not null"`
	Question   Question `gorm:"foreignKey:QuestionID"`
}
