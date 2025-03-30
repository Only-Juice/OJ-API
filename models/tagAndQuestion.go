package models

type TagAndQuestion struct {
	ID         uint     `gorm:"primaryKey" json:"id"`
	TagID      uint     `gorm:"not null" json:"tag_id"`
	Tag        Tag      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"tag"`
	QuestionID uint     `gorm:"not null" json:"question_id"`
	Question   Question `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"question"`
}
