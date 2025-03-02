package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"OJ-API/database"
	"OJ-API/models"
)

// Specify the shell command for the corresponding repo
// @Summary		Specify the shell command for the corresponding repo
// @Description	Specify the shell command for the corresponding repo
// @Tags			Sandbox
// @Accept			json
// @Produce		json
// @Param			cmd	body		models.Sandbox	true	"Shell command"
// @Success		200		{object}	ResponseHTTP{data=models.Sandbox}
// @Failure		503		{object}	ResponseHTTP{}
// @Router			/api/sandbox [post]
func PostSandboxCmd(w http.ResponseWriter, r *http.Request) {
	db := database.DBConn

	cmd := new(models.Sandbox)
	if err := json.NewDecoder(r.Body).Decode(cmd); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "Failed to parse shell command",
		})
		return
	}

	var existingCmd models.Sandbox
	if err := db.Where("source_git_repo = ?", cmd.SourceGitRepo).First(&existingCmd).Error; err != nil {
		// If the script does not exist, create a new one
		db.Create(cmd)
		existingCmd = *cmd
	} else {
		// If the script exists, update it
		existingCmd.Script = cmd.Script
		db.Save(&existingCmd)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ResponseHTTP{
		Success: true,
		Message: fmt.Sprintf("Success set shell command for %v.", cmd.SourceGitRepo),
		Data:    existingCmd,
	})
}
