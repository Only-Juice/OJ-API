package models

type ExamAndUser struct {
	ID         uint     `gorm:"primaryKey"`
	ExamID     uint     `gorm:"not null"`
	Exam       Exam     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	UserID     uint     `gorm:"not null"`
	User       User     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	QuestionID uint     `gorm:"not null"`
	Question   Question `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Score      int      `gorm:"not null"`
}
