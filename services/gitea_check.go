package services

import (
	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/utils"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
)

func GiteaCheckInit() {
	// 開啟時先檢查一次
	go giteaCheck()
	// 每10分鐘檢查一次 Gitea
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		for range ticker.C {
			giteaCheck()
		}
	}()
}

func giteaCheck() {
	// Find Admin user for Gitea operations
	db := database.DBConn
	var adminUser models.User
	if err := db.First(&adminUser, models.User{
		IsAdmin: true,
	}).Error; err != nil {
		utils.Errorf("Failed to find admin user: %v", err)
	}
	giteaToken, err := utils.GetToken(adminUser.ID)
	if err != nil {
		utils.Errorf("Failed to get Gitea token: %v", err)
		return
	}
	// gitea connection
	client, err := gitea.NewClient(config.GetGiteaBaseURL(), gitea.SetToken(giteaToken))
	if err != nil {
		utils.Errorf("Failed to create Gitea client: %v", err)
		return
	}
	// 找出有效的 UQR
	var uqr []models.UserQuestionRelation
	if err := db.Preload("User").Joins("JOIN questions ON questions.id = user_question_relations.question_id").
		Where("questions.start_time <= ? AND questions.end_time >= ?", time.Now(), time.Now()).
		Find(&uqr).Error; err != nil {
		utils.Errorf("Failed to find user question relations: %v", err)
		return
	}
	utils.Debugf("Found %d active UserQuestionRelations", len(uqr))
	// 檢查 Webhook 是否存在
	for _, item := range uqr {
		parts := strings.Split(item.GitUserRepoURL, "/")
		if len(parts) < 2 {
			utils.Errorf("Invalid GitUserRepoURL format: %s", item.GitUserRepoURL)
			continue
		}
		username, reponame := parts[0], parts[1]
		utils.Debugf("Checking hooks for %s/%s...", username, reponame)
		hooks, _, err := client.ListRepoHooks(username, reponame, gitea.ListHooksOptions{})
		if err != nil {
			utils.Errorf("Failed to list hooks for %s/%s: %v", username, reponame, err)
			continue
		}
		token, err := utils.GenerateAccessToken(item.User.ID, item.User.UserName, item.User.IsAdmin)
		if err != nil {
			utils.Errorf("Failed to generate token for %s/%s: %v", username, reponame, err)
			break
		}

		hookExists := false
		for _, hook := range hooks {
			if hook.Config["url"] == config.GetOJBaseURL()+"/api/gitea" {
				if hook.AuthorizationHeader != token {
					// 更新 Authorization Header
					if _, err := client.EditRepoHook(username, reponame, hook.ID, gitea.EditHookOption{
						Config: map[string]string{
							"url":          config.GetOJBaseURL() + "/api/gitea",
							"content_type": "json",
						},
						AuthorizationHeader: "Bearer " + token,
					}); err != nil {
						utils.Errorf("Failed to update hook for %s/%s: %v", username, reponame, err)
					}
				}
				hookExists = true
				break
			}
		}
		if !hookExists {
			if _, _, err := client.CreateRepoHook(username, reponame, gitea.CreateHookOption{
				Type:   "gitea",
				Active: true,
				Events: []string{"push"},
				Config: map[string]string{
					"url":          config.GetOJBaseURL() + "/api/gitea",
					"content_type": "json",
				},
				AuthorizationHeader: "Bearer " + token,
			}); err != nil {
				utils.Errorf("Failed to create hook for %s/%s: %v", username, reponame, err)
				continue
			}
		}
		// wait 100 ms
		time.Sleep(100 * time.Millisecond)
	}
}
