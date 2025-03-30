package models

type Announcement struct {
	ID          uint   `gorm:"primaryKey"`
	UserID      uint   `gorm:"not null"`
	User        User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Title       string `gorm:"not null;size:50"`
	Description string `gorm:"size:50"`
}
