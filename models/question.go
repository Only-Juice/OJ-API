package models

type Question struct {
	ID          uint   `gorm:"primaryKey"`
	Title       string `gorm:"size:100;not null"`
	Description string `gorm:"size:5000;not null"`
	GitRepoURL  string `gorm:"size:250"`
}
