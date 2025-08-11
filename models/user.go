package models

type User struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	UserName     string `gorm:"size:100;not null" json:"user_name"`
	Enable       bool   `gorm:"default:true;not null" json:"enable"`
	Email        string `gorm:"size:450;not null" json:"email"`
	IsPublic     bool   `gorm:"default:true;not null" json:"is_public"`
	GiteaToken   string `gorm:"size:450" json:"gitea_token"`
	RefreshToken string `gorm:"size:1000" json:"refresh_token"`
	IsAdmin      bool   `gorm:"default:false;not null" json:"is_admin"`
}
