package models

type Announcement struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	UserID      uint   `gorm:"not null" json:"user_id"`
	User        User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"user"`
	Title       string `gorm:"not null;size:50" json:"title"`
	Description string `gorm:"size:50" json:"description"`
}
