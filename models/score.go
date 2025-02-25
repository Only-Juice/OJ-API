package models

import "gorm.io/gorm"

type Score struct {
	gorm.Model
	GitRepo string `json:"git_repo" example:"user_name/repo_name" validate:"required"`
	Score float64 `json:"score" example:"100" validate:"required"`
	Message string `json:"message" example:"Scored successfully" validate:"required"`
}