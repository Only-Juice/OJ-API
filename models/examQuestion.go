package models

type ExamQuestion struct {
	ExamID     uint     `gorm:"primarykey"`
	Exam       Exam     `gorm:"foreignKey:ExamID"`
	QuestionID uint     `gorm:"primarykey"`
	Question   Question `gorm:"foreignKey:QuestionID"`
	Point      int      `gorm:"not null"` // Points for the question in the exam
}
