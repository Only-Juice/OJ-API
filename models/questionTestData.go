package models

type QuestionTestData struct {
	ID         uint     `gorm:"primaryKey"`
	QuestionID uint     `gorm:"not null"`
	Question   Question `gorm:"foreignKey:QuestionID"`
	TestData   string   `gorm:"size:200;not null"`
}
