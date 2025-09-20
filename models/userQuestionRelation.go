package models

type UserQuestionRelation struct {
	ID             uint     `gorm:"primaryKey" json:"id"`
	UserID         uint     `gorm:"not null;index:idx_uqr_question_user" json:"user_id"`
	User           User     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"user"`
	QuestionID     uint     `gorm:"not null;index:idx_uqr_question_user" json:"question_id"`
	Question       Question `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"question"`
	GitUserRepoURL string   `gorm:"size:150;not null" json:"git_user_repo_url"`
}
