package models

import "gorm.io/gorm"

type Score struct {
	gorm.Model
	UserName string `json:"user_name" example:"user_name" validate:"required"`
	RepoName string `json:"repo_name" example:"repo_name" validate:"required"`
	Score float64 `json:"score" example:"100" validate:"required"`
	Message string `json:"message" example:"Scored successfully" validate:"required"`
}