package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/sandbox"

	"code.gitea.io/sdk/gitea"
)

type Sandbox struct {
	SourceGitRepo string `json:"source_git_url" example:"user_name/repo_name" validate:"required"`
	Script        string `json:"script" example:"#!/bin/bash\n\necho 'Hello, World!'" validate:"required"`
}

// Specify the shell command for the corresponding repo
// @Summary		Specify the shell command for the corresponding repo
// @Description	Specify the shell command for the corresponding repo
// @Tags			Sandbox
// @Accept			json
// @Produce		json
// @Param			cmd	body		Sandbox	true	"Shell command"
// @Success		200		{object}	ResponseHTTP{data=models.QuestionTestScript}
// @Failure		401		{object}	ResponseHTTP{}
// @Failure		503		{object}	ResponseHTTP{}
// @Router			/api/sandbox [post]
// @Security		AuthorizationHeaderToken
func PostSandboxCmd(w http.ResponseWriter, r *http.Request) {
	db := database.DBConn
	giteaUser := r.Context().Value(models.UserContextKey).(*gitea.User)
	if !giteaUser.IsAdmin {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	cmd := new(Sandbox)
	if err := json.NewDecoder(r.Body).Decode(cmd); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "Failed to parse shell command",
		})
		return
	}

	var existingCmd models.QuestionTestScript
	if err := db.Joins("Question").
		Where("git_repo_url = ?", cmd.SourceGitRepo).
		Take(&existingCmd).Error; err != nil {
		// If the script does not exist, create a new one
		existingCmd = models.QuestionTestScript{
			Question:   models.Question{GitRepoURL: cmd.SourceGitRepo},
			TestScript: cmd.Script,
		}
		db.Create(&existingCmd)
	} else {
		// If the script exists, update it
		existingCmd.TestScript = cmd.Script
		db.Save(&existingCmd)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ResponseHTTP{
		Success: true,
		Message: fmt.Sprintf("Success set shell command for %v.", cmd.SourceGitRepo),
		Data:    existingCmd,
	})
}

type StatusResponse struct {
	AvailableCount  int `json:"available_count"`
	WaitingCount    int `json:"waiting_count"`
	ProcessingCount int `json:"processing_count"`
}

// GetSandboxStatus godoc
//
// @Summary Get the current available sandbox count and waiting count
// @Description Get the current available sandbox count and waiting count
// @Tags Sandbox
// @Produce json
// @Success		200		{object}	ResponseHTTP{data=StatusResponse}
// @Failure		500		{object}	ResponseHTTP{}
// @Router /api/sandbox/status [get]
func GetSandboxStatus(w http.ResponseWriter, r *http.Request) {
	if sandbox.SandboxPtr == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "Sandbox instance not initialized",
		})
		return
	}

	status := StatusResponse{
		AvailableCount:  sandbox.SandboxPtr.AvailableCount(),
		WaitingCount:    sandbox.SandboxPtr.WaitingCount(),
		ProcessingCount: sandbox.SandboxPtr.ProcessingCount(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ResponseHTTP{
		Success: true,
		Message: "",
		Data:    status,
	})
}
