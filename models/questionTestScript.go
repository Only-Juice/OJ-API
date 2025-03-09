package models

type QuestionTestScript struct {
	ID         uint     `gorm:"primaryKey"`
	QuestionID uint     `gorm:"not null"`
	Question   Question `gorm:"foreignKey:QuestionID"`
	TestScript string   `gorm:"size:4000;not null"`
}
