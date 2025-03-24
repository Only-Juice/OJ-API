package models

type User struct {
	ID         uint   `gorm:"primaryKey"`
	UserName   string `gorm:"size:100;not null"`
	Enable     bool   `gorm:"default:true;not null"`
	Email      string `gorm:"size:450;not null"`
	IsPublic   bool   `gorm:"default:true;not null"`
	GiteaToken string `gorm:"size:450;not null"`
}
