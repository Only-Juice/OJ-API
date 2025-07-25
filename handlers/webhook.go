package handlers

import (
	"fmt"
	"log"
	"os"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/gin-gonic/gin"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/uuid"

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
	log.Printf("Received hook: %+v", payload)

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
	now := time.Now()
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
		UQR:     existingUserQuestionRelation,
		Score:   -3,
		Message: "Waiting for judging...",
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
		// Clone the given repository to the given directory
		log.Printf("git clone %s", "http://"+config.Config("GIT_HOST")+"/"+payload.Repository.FullName)
		codePath := fmt.Sprintf("%s/%s", config.Config("REPO_FOLDER"), payload.Repository.FullName+"/"+uuid.New().String())
		token, err := utils.GetToken(existingUser.ID)
		var cloneOptions *git.CloneOptions
		if err != nil {
			log.Printf("Failed to get token: %v", err)
			db.Model(&newScore).Updates(models.UserQuestionTable{
				Score:   -2,
				Message: fmt.Sprintf("Failed to get token: %v", err),
			})
			return
		} else {
			cloneOptions = &git.CloneOptions{
				URL: "http://" + config.Config("GIT_HOST") + "/" + payload.Repository.FullName,
				Auth: &http.BasicAuth{
					Username: existingUser.UserName,
					Password: token,
				},
				Progress: os.Stdout,
			}
		}

		repo, err := git.PlainClone(codePath, false, cloneOptions)
		if err != nil {
			db.Model(&newScore).Updates(models.UserQuestionTable{
				Score:   -2,
				Message: fmt.Sprintf("Failed to clone repository: %v", err),
			})
			return
		}
		os.Chmod(codePath, 0777) // Need to confirm if this is necessary
		log.Printf("git show-ref --head HEAD")
		ref, err := repo.Head()
		if err != nil {
			db.Model(&newScore).Updates(models.UserQuestionTable{
				Score:   -2,
				Message: fmt.Sprintf("Failed to get HEAD: %v", err),
			})
			return
		}
		fmt.Println(ref.Hash())

		worktree, err := repo.Worktree()
		if err != nil {
			db.Model(&newScore).Updates(models.UserQuestionTable{
				Score:   -2,
				Message: fmt.Sprintf("Failed to get worktree: %v", err),
			})
			return
		}
		err = worktree.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(payload.After),
		})
		if err != nil {
			db.Model(&newScore).Updates(models.UserQuestionTable{
				Score:   -2,
				Message: fmt.Sprintf("Failed to checkout: %v", err),
			})
			return
		}
		ref, err = repo.Head()
		if err != nil {
			db.Model(&newScore).Updates(models.UserQuestionTable{
				Score:   -2,
				Message: fmt.Sprintf("Failed to get HEAD: %v", err),
			})
			return
		}
		fmt.Println(ref.Hash())

		// 使用 gRPC 客戶端添加任務
		clientManager := services.GetSandboxClientManager()
		if err := clientManager.ReserveJob(existingQuestion.GitRepoURL, []byte(codePath), uint64(newScore.ID)); err != nil {
			db.Model(&newScore).Updates(models.UserQuestionTable{
				Score:   -2,
				Message: fmt.Sprintf("Failed to queue job: %v", err),
			})
		}
	}()
}
