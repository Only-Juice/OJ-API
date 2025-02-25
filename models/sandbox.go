package models

import "gorm.io/gorm"

// Sandbox is a model for sandbox
type Sandbox struct {
	gorm.Model
	SourceGitRepo string `json:"source_git_url" example:"user_name/repo_name" validate:"required"`
	Script string `json:"script" example:"#!/bin/bash\n\necho 'Hello, World!'" validate:"required"`
}