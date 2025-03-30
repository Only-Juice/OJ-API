package models

type QuestionTestScript struct {
	ID         uint     `gorm:"primaryKey" json:"id"`
	QuestionID uint     `gorm:"not null" json:"question_id"`
	Question   Question `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"question"`
	TestScript string   `gorm:"size:4000;not null" json:"test_script"`
}
