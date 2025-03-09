package models

type Sandbox struct {
	SourceGitRepo string `json:"source_git_url" example:"user_name/repo_name" validate:"required"`
	Script        string `json:"script" example:"#!/bin/bash\n\necho 'Hello, World!'" validate:"required"`
}
