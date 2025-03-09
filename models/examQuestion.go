package models

type ExamQuestion struct {
	ExamID     uint     `gorm:"primarykey"`
	Exam       Exam     `gorm:"foreignKey:ExamID"`
	QuestionID uint     `gorm:"primarykey"`
	Question   Question `gorm:"foreignKey:QuestionID"`
	Score      int      `gorm:"not null"`
}
