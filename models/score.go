package models

import "time"

// Deprecated in the future
type Score struct {
	Score     float64   `json:"score" example:"100" validate:"required"`
	Message   string    `json:"message" example:"Scored successfully" validate:"required"`
	JudgeTime time.Time `json:"judge_time" example:"2021-07-01T00:00:00Z" validate:"required"`
}
