package models

type ExamAndUser struct {
	ID         uint `gorm:"primarykey"`
	ExamID     uint `gorm:"not null"`
	Exam       Exam `gorm:"foreignKey:ExamID"`
	UserID     uint `gorm:"not null"`
	User       User `gorm:"foreignKey:UserID"`
	QuestionID uint `gorm:"not null"`
	Score      int  `gorm:"not null"`
}
