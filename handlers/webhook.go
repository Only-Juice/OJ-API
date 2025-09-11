package handlers

import (
	"fmt"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/gin-gonic/gin"

	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/services"
	"OJ-API/utils"
)

type WebhookPayload struct {
	Ref        string           `json:"ref"`
	Before     string           `json:"before"`
	After      string           `json:"after"`
	CompareURL string           `json:"compare_url"`
	Commits    []gitea.Commit   `json:"commits"`
	Repository gitea.Repository `json:"repository"`
	Pusher     gitea.User       `json:"pusher"`
	Sender     gitea.User       `json:"sender"`
}

// PostGiteaHook is a function to receive Gitea hook
//
//	@Summary		Receive Gitea hook
//	@Description	Receive Gitea hook
//	@Tags			Gitea
//	@Accept			json
//	@Produce		json
//	@Param			hook	body		WebhookPayload	true	"Gitea Hook"
//	@Success		200		{object}	ResponseHTTP{type=WebhookPayload}
//	@Failure		403		{object}	ResponseHTTP{}
//	@Failure		410		{object}	ResponseHTTP{}
//	@Failure		503		{object}	ResponseHTTP{}
//	@Router			/api/gitea [post]
func PostGiteaHook(c *gin.Context) {
	db := database.DBConn
	var payload WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to parse hook",
		})
		return
	}
	utils.Debugf("Received hook: %+v", payload)

	var existingUserQuestionRelation models.UserQuestionRelation
	if err := db.Where(&models.UserQuestionRelation{
		GitUserRepoURL: payload.Repository.FullName,
	}).First(&existingUserQuestionRelation).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "No user-question relation found for this repository",
		})
		return
	}

	var existingQuestion models.Question
	if err := db.Where(&models.Question{ID: existingUserQuestionRelation.QuestionID, IsActive: true}).First(&existingQuestion).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "No question found for this user-question relation",
		})
		return
	}

	// Check if current time is within the allowed testing period
	now := time.Now().UTC()
	if !existingQuestion.StartTime.IsZero() && now.Before(existingQuestion.StartTime) {
		c.JSON(403, ResponseHTTP{
			Success: false,
			Message: "Testing period has not started yet",
		})
		return
	}
	if !existingQuestion.EndTime.IsZero() && now.After(existingQuestion.EndTime) {
		c.JSON(410, ResponseHTTP{
			Success: false,
			Message: "Testing period has ended",
		})
		return
	}

	var existingUser models.User
	if err := db.Where(&models.User{UserName: payload.Pusher.UserName}).First(&existingUser).Error; err != nil {
		existingUser = models.User{
			UserName: payload.Pusher.UserName,
			Email:    payload.Pusher.Email,
		}
		db.Create(&existingUser)
	}

	newScore := models.UserQuestionTable{
		UQR:       existingUserQuestionRelation,
		Score:     -3,
		JudgeTime: time.Now().UTC(),
		Commit:    payload.After,
		Message:   "Waiting for judging...",
	}
	if err := db.Create(&newScore).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to create new score entry",
		})
		return
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Successfully received hook",
		Data:    payload,
	})

	go func() {
		// 獲取用戶 token
		token, err := utils.GetToken(existingUser.ID)
		if err != nil {
			utils.Errorf("Failed to get token: %v", err)
			db.Model(&newScore).Updates(models.UserQuestionTable{
				Score:   -2,
				Message: fmt.Sprintf("Failed to get token: %v", err),
			})
			return
		}

		// 構建 Git 倉庫 URL
		gitRepoURL := config.GetGiteaBaseURL() + "/" + payload.Repository.FullName

		// 使用 gRPC 客戶端添加任務，Git clone 將在沙箱端完成
		clientManager := services.GetSandboxClientManager()
		if err := clientManager.ReserveJob(
			existingQuestion.GitRepoURL, // parentGitFullName
			gitRepoURL,                  // gitRepoURL
			payload.Repository.FullName, // gitFullName
			payload.After,               // gitAfterHash
			existingUser.UserName,       // gitUsername
			token,                       // gitToken
			uint64(newScore.ID),         // userQuestionTableID
		); err != nil {
			db.Model(&newScore).Updates(models.UserQuestionTable{
				Score:   -2,
				Message: fmt.Sprintf("Failed to queue job: %v", err),
			})
		}
	}()
}
