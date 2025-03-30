package models

type ExamQuestion struct {
	ExamID     uint     `gorm:"primaryKey"`
	Exam       Exam     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	QuestionID uint     `gorm:"primaryKey"`
	Question   Question `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Point      int      `gorm:"not null"` // Points for the question in the exam
}
