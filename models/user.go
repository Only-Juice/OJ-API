package models

type User struct {
	ID       uint   `gorm:"primaryKey"`
	UID      string `gorm:"size:15;not null"`
	Name     string `gorm:"size:50;not null"`
	IsAdmin  bool   `gorm:"default:false;not null"`
	Password string `gorm:"size:25;not null"`
	Enable   bool   `gorm:"default:true;not null"`
	Email    string `gorm:"size:450;not null"`
	IsPublic bool   `gorm:"default:true;not null"`
	Image    string `gorm:"size:250"`
}
