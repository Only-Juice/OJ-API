package handlers

import (
	"fmt"

	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/sandbox"
	"OJ-API/utils"

	"github.com/gin-gonic/gin"
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
func PostSandboxCmd(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(401, ResponseHTTP{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	cmd := new(Sandbox)
	if err := c.ShouldBindJSON(cmd); err != nil {
		c.JSON(503, ResponseHTTP{
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

	c.JSON(200, ResponseHTTP{
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
func GetSandboxStatus(c *gin.Context) {
	if sandbox.SandboxPtr == nil {
		c.JSON(500, ResponseHTTP{
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

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "",
		Data:    status,
	})
}
