package models

type UserQuestionRelation struct {
	ID             uint     `gorm:"primaryKey"`
	UserID         uint     `gorm:"not null"`
	User           User     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	QuestionID     uint     `gorm:"not null"`
	Question       Question `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	GitUserRepoURL string   `gorm:"size:150;not null"`
}
