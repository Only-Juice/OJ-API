package models

type UserQuestionRelation struct {
	ID             uint     `gorm:"primaryKey"`
	UserID         uint     `gorm:"not null"`
	User           User     `gorm:"foreignKey:UserID"`
	QuestionID     uint     `gorm:"not null"`
	Question       Question `gorm:"foreignKey:QuestionID"`
	GitUserRepoURL string   `gorm:"size:150;not null"`
}
