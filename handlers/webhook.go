package handlers

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/gin-gonic/gin"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/uuid"

	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/sandbox"
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
//	@Failure		503		{object}	ResponseHTTP{}
//	@Router			/api/gitea [post]
func PostGiteaHook(c *gin.Context) {
	var payload WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to parse hook",
		})
		return
	}
	log.Printf("Received hook: %+v", payload)

	// Respond immediately
	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Successfully received hook",
		Data:    payload,
	})

	// Process the hook in the background
	go func() {
		// Clone the given repository to the given directory
		log.Printf("git clone %s", "http://"+config.Config("GIT_HOST")+"/"+payload.Repository.FullName)
		codePath := fmt.Sprintf("%s/%s", config.Config("REPO_FOLDER"), payload.Repository.FullName+"/"+uuid.New().String())
		repo, err := git.PlainClone(codePath, false, &git.CloneOptions{
			URL:      "http://" + config.Config("GIT_HOST") + "/" + payload.Repository.FullName,
			Progress: os.Stdout,
		})
		if err != nil {
			log.Printf("Failed to clone repository: %v", err)
			return
		}
		os.Chmod(codePath, 0777)
		log.Printf("git show-ref --head HEAD")
		ref, err := repo.Head()
		if err != nil {
			log.Printf("Failed to get HEAD: %v", err)
			return
		}
		fmt.Println(ref.Hash())

		worktree, err := repo.Worktree()
		if err != nil {
			log.Printf("Failed to get worktree: %v", err)
			return
		}
		err = worktree.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(payload.After),
		})
		if err != nil {
			log.Printf("Failed to checkout: %v", err)
			return
		}
		ref, err = repo.Head()
		if err != nil {
			log.Printf("Failed to get HEAD: %v", err)
			return
		}
		fmt.Println(ref.Hash())

		defer os.RemoveAll(codePath)
		sandbox.SandboxPtr.RunShellCommandByRepo(payload.Repository.Parent.FullName, []byte(codePath))

		// read score from file
		score, err := os.ReadFile(fmt.Sprintf("%s/score.txt", codePath))
		if err != nil {
			log.Printf("Failed to read score: %v", err)
			return
		}
		// save score to database
		db := database.DBConn
		scoreFloat, err := strconv.ParseFloat(strings.TrimSpace(string(score)), 64)
		if err != nil {
			log.Printf("Failed to convert score to int: %v", err)
			return
		}

		// read message from file
		message, err := os.ReadFile(fmt.Sprintf("%s/message.txt", codePath))
		if err != nil {
			log.Printf("Failed to read message: %v", err)
			return
		}
		var existingQuestion models.Question
		if err := db.Where(&models.Question{GitRepoURL: payload.Repository.Parent.FullName}).First(&existingQuestion).Error; err != nil {
			existingQuestion = models.Question{
				GitRepoURL: payload.Repository.Parent.FullName,
			}
			db.Create(&existingQuestion)
		}
		var existingUser models.User
		if err := db.Where(&models.User{UserName: payload.Pusher.UserName}).First(&existingUser).Error; err != nil {
			existingUser = models.User{
				UserName: payload.Pusher.UserName,
				Email:    payload.Pusher.Email,
			}
			db.Create(&existingUser)
		}

		var existingUserQuestionRelation models.UserQuestionRelation
		if err := db.Where(&models.UserQuestionRelation{
			UserID:     existingUser.ID,
			QuestionID: existingQuestion.ID,
		}).First(&existingUserQuestionRelation).Error; err != nil {
			// If the relation does not exist, create a new one
			existingUserQuestionRelation = models.UserQuestionRelation{
				User:           existingUser,
				Question:       existingQuestion,
				GitUserRepoURL: payload.Repository.FullName,
			}
			db.Create(&existingUserQuestionRelation)
		}
		newScore := models.UserQuestionTable{
			UQR:       existingUserQuestionRelation,
			Score:     scoreFloat,
			JudgeTime: time.Now().UTC(),
			Message:   strings.TrimSpace(string(message)),
		}

		if err := db.Create(&newScore).Error; err != nil {
			log.Printf("Failed to create new score entry: %v", err)
			return
		}
	}()
}
