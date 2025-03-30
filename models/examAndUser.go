package models

type ExamAndUser struct {
	ID         uint     `gorm:"primaryKey" json:"id"`
	ExamID     uint     `gorm:"not null" json:"exam_id"`
	Exam       Exam     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"exam"`
	UserID     uint     `gorm:"not null" json:"user_id"`
	User       User     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"user"`
	QuestionID uint     `gorm:"not null" json:"question_id"`
	Question   Question `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"question"`
	Score      int      `gorm:"not null" json:"score"`
}
