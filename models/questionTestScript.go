package models

type QuestionTestScript struct {
	ID            uint     `gorm:"primaryKey" json:"id"`
	QuestionID    uint     `gorm:"not null" json:"question_id"`
	Question      Question `gorm:"foreignKey:QuestionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"question"`
	CompileScript string   `gorm:"size:4000;not null" json:"compile_script"`
	ExecuteScript string   `gorm:"size:4000;not null" json:"execute_script"`
	ScoreScript   string   `gorm:"size:8000;not null" json:"score_script"`
	Memory        uint     `gorm:"not null;default:262144" json:"memory"`
	StackMemory   uint     `gorm:"not null;default:8192" json:"stack_memory"`
	Time          uint     `gorm:"not null;default:1000" json:"time"`
	WallTime      uint     `gorm:"not null;default:3000" json:"wall_time"`
	FileSize      uint     `gorm:"not null;default:10240" json:"file_size"`
	Processes     uint     `gorm:"not null;default:10" json:"processes"`
	OpenFiles     uint     `gorm:"not null;default:64" json:"open_files"`
	ScoreMap      string   `gorm:"size:8000;not null" json:"score_map"`
}
