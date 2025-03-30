package models

type QuestionTestScript struct {
	ID         uint     `gorm:"primaryKey"`
	QuestionID uint     `gorm:"not null"`
	Question   Question `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	TestScript string   `gorm:"size:4000;not null"`
}
