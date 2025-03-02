package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"code.gitea.io/sdk/gitea"
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
//	@Tags			WebHook
//	@Accept			json
//	@Produce		json
//	@Param			hook	body		WebhookPayload	true	"Gitea Hook"
//	@Success		200		{object}	ResponseHTTP{type=WebhookPayload}
//	@Failure		503		{object}	ResponseHTTP{}
//	@Router			/api/gitea [post]
func PostGiteaHook(w http.ResponseWriter, r *http.Request) {
	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Failed to parse hook", http.StatusServiceUnavailable)
		return
	}
	log.Printf("Received hook: %+v", payload)

	// Respond immediately
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ResponseHTTP{
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
		defer os.RemoveAll(codePath)
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

		sandbox.SandboxPtr.RunShellCommandByRepo(payload.Repository.Parent.FullName, []byte(codePath))

		// read score from file
		score, err := os.ReadFile(fmt.Sprintf("%s/score.txt", codePath))
		if err != nil {
			log.Printf("Failed to read score: %v", err)
			return
		}
		// log.Printf("Score: %s", score)
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
		// log.Printf("Message: %s", message)

		// Create a new score entry in the database
		newScore := models.Score{
			UserName: payload.Repository.Owner.UserName,
			RepoName: payload.Repository.Name,
			Score:    scoreFloat,
			Message:  strings.TrimSpace(string(message)),
		}

		if err := db.Create(&newScore).Error; err != nil {
			log.Printf("Failed to create new score entry: %v", err)
			return
		}
	}()
}
