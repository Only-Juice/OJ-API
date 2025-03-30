package models

type ExamQuestion struct {
	ExamID     uint     `gorm:"primaryKey" json:"exam_id"`
	Exam       Exam     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"exam"`
	QuestionID uint     `gorm:"primaryKey" json:"question_id"`
	Question   Question `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"question"`
	Point      int      `gorm:"not null" json:"point"` // Points for the question in the exam
}
