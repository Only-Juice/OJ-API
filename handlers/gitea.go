package handlers

import (
	"strconv"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/gin-gonic/gin"

	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/utils"
)

type BulkCreateUser struct {
	Usernames       []string `json:"usernames" validate:"required" example:"username1,username2"`
	EmailDomain     string   `json:"email_domain" validate:"required" example:"example.com"`
	DefaultPassword string   `json:"default_password" validate:"required" example:"password"`
}

type BulkCreateUserResponse struct {
	SuccessfulUsers []string          `json:"successful_users" example:"username1"`
	FailedUsers     map[string]string `json:"failed_users" example:"username1:error"`
}

func CreateUserAccessToken(username, email, password string) {
	db := database.DBConn
	client, err := gitea.NewClient(config.GetGiteaBaseURL(),
		gitea.SetBasicAuth(username, password),
	)
	if err != nil {
		utils.Errorf("Failed to create Gitea client for user %s: %v", username, err)
		return
	}

	var existingUser models.User
	if err := db.Where(&models.User{UserName: username}).First(&existingUser).Error; err != nil {
		existingUser = models.User{
			UserName:      username,
			Email:         email,
			ResetPassword: true,
		}
		db.Create(&existingUser)
	}

	if existingUser.GiteaToken == "" {
		client.DeleteAccessToken("OJ-API")
		token, _, err := client.CreateAccessToken(gitea.CreateAccessTokenOption{
			Name:   "OJ-API",
			Scopes: []gitea.AccessTokenScope{gitea.AccessTokenScopeAll},
		})
		if err != nil {
			utils.Errorf("Failed to create access token for user %s: %v", username, err)
			return
		}
		if err := utils.StoreToken(existingUser.ID, token.Token); err != nil {
			utils.Errorf("Failed to store token for user %s: %v", username, err)
			return
		}
	}
}

// Bulk create User
// @Summary	Bulk create User
// @Description Bulk create User
// @Tags			Gitea
// @Accept			json
// @Produce			json
// @Param			Usernames	body		BulkCreateUser		true	"Username + Email Domain => username1@example.com"
// @Success		200		{object}	ResponseHTTP{data=BulkCreateUserResponse} "Return successful and failed users"
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Security	BearerAuth
// @Router		/api/gitea/admin/user/bulk [post]
func PostBulkCreateUserGitea(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(403, ResponseHTTP{
			Success: false,
			Message: "Permission denied",
		})
		return
	}

	bulkUsers := new(BulkCreateUser)
	if err := c.ShouldBindJSON(bulkUsers); err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to parse bulk users",
		})
		return
	}
	token, err := utils.GetToken(jwtClaims.UserID)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to retrieve token",
		})
		return
	}
	client, err := gitea.NewClient(config.GetGiteaBaseURL(),
		gitea.SetToken(token),
	)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	successfulUsers := []map[string]string{}
	failedUsers := map[string]string{}

	for _, username := range bulkUsers.Usernames {
		password := bulkUsers.DefaultPassword
		if password == "" {
			password = utils.GenerateRandomPassword()
		}
		_, _, err := client.AdminCreateUser(gitea.CreateUserOption{
			Email:              username + "@" + bulkUsers.EmailDomain,
			Username:           username,
			Password:           password,
			MustChangePassword: func(b bool) *bool { return &b }(false),
		})
		if err != nil {
			failedUsers[username] = err.Error()
		} else {
			successfulUsers = append(successfulUsers, map[string]string{
				"username": username,
				"password": password,
			})
			db.Create(&models.User{
				UserName:      username,
				Email:         username + "@" + bulkUsers.EmailDomain,
				ResetPassword: true,
			})
		}
	}

	go func() {
		// Create access tokens sequentially after all users are created
		for _, user := range successfulUsers {
			CreateUserAccessToken(user["username"], user["username"]+"@"+bulkUsers.EmailDomain, user["password"])
			utils.SendDefaultPasswordNotification(user["username"]+"@"+bulkUsers.EmailDomain, user["username"], user["password"])
		}
	}()

	c.JSON(200, ResponseHTTP{
		Success: true,
		Data: BulkCreateUserResponse{
			SuccessfulUsers: func() []string {
				usernames := []string{}
				for _, user := range successfulUsers {
					usernames = append(usernames, user["username"])
				}
				return usernames
			}(),
			FailedUsers: failedUsers,
		},
		Message: "Bulk user creation completed",
	})
}

type BulkCreateUserItem struct {
	Email    string `json:"email" validate:"required" example:"username1@example.com"`
	Username string `json:"username" validate:"required" example:"username1"`
	Password string `json:"password" validate:"required" example:"password"`
}

type BulkCreateUserRequest struct {
	User []BulkCreateUserItem `json:"user" validate:"required"`
}

// Bulk create User v2
// @Summary	Bulk create User v2
// @Description Bulk create User v2
// @Tags			Gitea
// @Accept			json
// @Produce			json
// @Param			BulkCreateUserRequest	body		BulkCreateUserRequest	true	"User Email, Username, Password"
// @Success		200		{object}	ResponseHTTP{data=BulkCreateUserResponse} "Return successful and failed users"
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Security	BearerAuth
// @Router		/api/gitea/admin/user/bulk_v2 [post]
func PostBulkCreateUserGiteav2(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	if !jwtClaims.IsAdmin {
		c.JSON(403, ResponseHTTP{
			Success: false,
			Message: "Permission denied",
		})
		return
	}

	bulkUsers := new(BulkCreateUserRequest)
	if err := c.ShouldBindJSON(bulkUsers); err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to parse bulk users",
		})
		return
	}
	token, err := utils.GetToken(jwtClaims.UserID)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to retrieve token",
		})
		return
	}
	client, err := gitea.NewClient(config.GetGiteaBaseURL(),
		gitea.SetToken(token),
	)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	successfulUsers := []string{}
	failedUsers := map[string]string{}

	for i, user := range bulkUsers.User {
		if bulkUsers.User[i].Password == "" {
			bulkUsers.User[i].Password = utils.GenerateRandomPassword()
		}
		if _, _, err := client.AdminCreateUser(gitea.CreateUserOption{
			Email:              user.Email,
			Username:           user.Username,
			Password:           bulkUsers.User[i].Password,
			MustChangePassword: func(b bool) *bool { return &b }(false),
		}); err != nil {
			failedUsers[user.Username] = err.Error()
			continue
		}

		db.Create(&models.User{
			UserName:      user.Username,
			Email:         user.Email,
			ResetPassword: true,
		})

		successfulUsers = append(successfulUsers, user.Username)
	}

	go func() {
		// Create access tokens sequentially after all users are created
		for _, user := range bulkUsers.User {
			// Only create tokens for successfully created users
			for _, successfulUser := range successfulUsers {
				if user.Username == successfulUser {
					CreateUserAccessToken(user.Username, user.Email, user.Password)
					utils.SendDefaultPasswordNotification(user.Email, user.Username, user.Password)
					break
				}
			}
		}
	}()

	c.JSON(200, ResponseHTTP{
		Success: true,
		Data: BulkCreateUserResponse{
			SuccessfulUsers: successfulUsers,
			FailedUsers:     failedUsers,
		},
		Message: "Bulk user creation completed",
	})
}

// take a question and migrate it to a private repository in Gitea
// @Summary	Take a question and migrate it to a private repository in Gitea
// @Description Take a question and migrate it to a private repository in Gitea (migrates the entire repo content to a new private repo)
// @Tags			Gitea
// @Accept			json
// @Produce			json
// @Param			question_id	path		int		true	"Question ID"
// @Success		200		{object}	ResponseHTTP{}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Security	BearerAuth
// @Router		/api/gitea/{question_id}/question [post]
func PostCreateQuestionRepositoryGitea(c *gin.Context) {
	db := database.DBConn
	questionIDStr := c.Param("question_id")
	questionID, err := strconv.Atoi(questionIDStr)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Invalid question ID",
		})
		return
	}

	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	token, err := utils.GetToken(jwtClaims.UserID)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to retrieve token",
		})
		return
	}
	client, err := gitea.NewClient(config.GetGiteaBaseURL(),
		gitea.SetToken(token),
	)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	var existingQuestion models.Question
	if err := db.Where("id = ? AND is_active = ?", questionID, true).First(&existingQuestion).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Question not found",
		})
		return
	}

	parentRepoURLParts := strings.Split(existingQuestion.GitRepoURL, "/")
	parentRepoUsername := parentRepoURLParts[0]
	parentRepoName := parentRepoURLParts[1]

	// Check if user already has this repository
	repo, _, err := client.GetRepo(jwtClaims.Username, parentRepoName)
	if err != nil {
		// Use migrate to create a private copy of the repository
		migrateRepo, _, err := client.MigrateRepo(gitea.MigrateRepoOption{
			RepoName:    parentRepoName,
			CloneAddr:   config.GetGiteaBaseURL() + "/" + parentRepoUsername + "/" + parentRepoName + ".git",
			Service:     gitea.GitServiceType("git"),
			Private:     true,
			Description: "Private copy of " + parentRepoUsername + "/" + parentRepoName,
			Mirror:      false, // Set to false to create an independent copy
		})
		if err != nil {
			c.JSON(503, ResponseHTTP{
				Success: false,
				Message: "Failed to migrate repository: " + err.Error(),
			})
			return
		}
		repo = migrateRepo
	}

	hooks, _, err := client.ListRepoHooks(jwtClaims.Username, parentRepoName, gitea.ListHooksOptions{})
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to list repository hooks: " + err.Error(),
		})
		return
	}
	hookExists := false
	for _, hook := range hooks {
		if hook.Config["url"] == config.GetOJBaseURL()+"/api/gitea" {
			hookExists = true
			break
		}
	}

	accessToken, err := utils.GenerateAccessToken(jwtClaims.UserID, jwtClaims.Username, jwtClaims.IsAdmin)
	if err != nil {
		utils.Errorf("Failed to generate token for %s/%s: %v", jwtClaims.Username, parentRepoName, err)
		return
	}
	if !hookExists {
		if _, _, err := client.CreateRepoHook(jwtClaims.Username, parentRepoName, gitea.CreateHookOption{
			Type:   "gitea",
			Active: true,
			Events: []string{"push"},
			Config: map[string]string{
				"url":          config.GetOJBaseURL() + "/api/gitea",
				"content_type": "json",
			},
			AuthorizationHeader: "Bearer " + accessToken,
		}); err != nil {
			c.JSON(503, ResponseHTTP{
				Success: false,
				Message: "Failed to create repository hook: " + err.Error(),
			})
			return
		}
	}

	var userQuestionRelation models.UserQuestionRelation
	if err := db.Where(&models.UserQuestionRelation{
		UserID:     jwtClaims.UserID,
		QuestionID: uint(questionID),
	}).First(&userQuestionRelation).Error; err != nil || (userQuestionRelation == models.UserQuestionRelation{}) {
		db.Create(&models.UserQuestionRelation{
			UserID:         jwtClaims.UserID,
			QuestionID:     uint(questionID),
			GitUserRepoURL: jwtClaims.Username + "/" + parentRepoName,
		})
	}

	if repo != nil {
		c.JSON(200, ResponseHTTP{
			Success: true,
			Message: "Repository exists, relation updated",
		})
		return
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Repository migrated to private copy successfully",
	})
}

// Get User Profile
// @Summary	Get User Profile
// @Description Get User Profile
// @Tags			Gitea
// @Accept			json
// @Produce			json
// @Success		200		{object}	ResponseHTTP{data=gitea.User}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Security	BearerAuth
// @Router		/api/gitea/user [get]
func GetUserProfileGitea(c *gin.Context) {
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	token, err := utils.GetToken(jwtClaims.UserID)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to retrieve token",
		})
		return
	}
	client, err := gitea.NewClient(config.GetGiteaBaseURL(),
		gitea.SetToken(token),
	)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	user, _, err := client.GetMyUserInfo()
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Data:    user,
		Message: "User profile retrieved",
	})
}

type CreatePublicKey struct {
	Title    string `json:"title" validate:"required" example:"Public Key"`
	Key      string `json:"key" validate:"required" example:"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC3..."`
	ReadOnly bool   `json:"read_only" validate:"required" example:"true"`
}

// Create a public key in Gitea
// @Summary	Create a public key in Gitea
// @Description Create a public key in Gitea
// @Tags			Gitea
// @Accept			json
// @Produce			json
// @Param			CreatePublicKey	body		CreatePublicKey	true	"Public Key"
// @Success		200		{object}	ResponseHTTP{}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Security	BearerAuth
// @Router		/api/gitea/user/keys [post]
func PostCreatePublicKeyGitea(c *gin.Context) {
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	token, err := utils.GetToken(jwtClaims.UserID)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to retrieve token",
		})
		return
	}
	client, err := gitea.NewClient(config.GetGiteaBaseURL(),
		gitea.SetToken(token),
	)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	publicKey := new(CreatePublicKey)
	if err := c.ShouldBindJSON(publicKey); err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to parse public key",
		})
		return
	}

	if _, _, err := client.CreatePublicKey(gitea.CreateKeyOption{
		Key:      publicKey.Key,
		ReadOnly: publicKey.ReadOnly,
		Title:    publicKey.Title,
	}); err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Public key created successfully",
	})
}

// ListMyPublicKeys list all the public keys of current user
// @Summary	List all public keys
// @Description List all public keys
// @Tags			Gitea
// @Accept			json
// @Produce			json
// @Success		200		{object}	ResponseHTTP{data=[]gitea.PublicKey}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Security	BearerAuth
// @Router		/api/gitea/user/keys [get]
func ListMyPublicKeysGitea(c *gin.Context) {
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	token, err := utils.GetToken(jwtClaims.UserID)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to retrieve token",
		})
		return
	}
	client, err := gitea.NewClient(config.GetGiteaBaseURL(),
		gitea.SetToken(token),
	)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	keys, _, err := client.ListMyPublicKeys(gitea.ListPublicKeysOptions{})
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Data:    keys,
		Message: "Public keys retrieved successfully",
	})
}

type DeletePublicKey struct {
	ID int64 `json:"id" validate:"required" example:"1"`
}

// DeletePublicKey delete public key with key id
// @Summary	Delete a public key
// @Description Delete a public key
// @Tags			Gitea
// @Accept			json
// @Produce			json
// @Param			DeletePublicKey	body		DeletePublicKey	true	"Public Key ID"
// @Success		200		{object}	ResponseHTTP{}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Security	BearerAuth
// @Router		/api/gitea/user/keys [delete]
func DeletePublicKeyGitea(c *gin.Context) {
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	token, err := utils.GetToken(jwtClaims.UserID)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to retrieve token",
		})
		return
	}
	client, err := gitea.NewClient(config.GetGiteaBaseURL(),
		gitea.SetToken(token),
	)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	deleteKey := new(DeletePublicKey)
	if err := c.ShouldBindJSON(deleteKey); err != nil {
		c.JSON(400, ResponseHTTP{
			Success: false,
			Message: "Failed to parse request body",
		})
		return
	}

	_, err = client.DeletePublicKey(deleteKey.ID)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Public key deleted successfully",
	})
}

// Gets the metadata of all the entries of the root dir
// @Summary	Get root directory metadata
// @Description Gets the metadata of all the entries of the root dir
// @Tags			Gitea
// @Accept			json
// @Produce			json
// @Param			owner		path		string		true	"owner of the repo"
// @Param			repo		path		string		true	"name of the repo"
// @Param			filepath	path		string		false	"The path to the file or directory. Use empty string for root directory." default(/)
// @Param			ref			query		string		false	"The name of the commit/branch/tag. Default to the repository’s default branch."
// @Success		200		{object}	ResponseHTTP{data=gitea.ContentsResponse}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Security	BearerAuth
// @Router		/api/gitea/repos/{owner}/{repo}/dir/{filepath} [get]
func ListRepoDirGitea(c *gin.Context) {
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	token, err := utils.GetToken(jwtClaims.UserID)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to retrieve token",
		})
		return
	}
	client, err := gitea.NewClient(config.GetGiteaBaseURL(),
		gitea.SetToken(token),
	)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	owner := c.Param("owner")
	repo := c.Param("repo")
	filepath := c.Param("filepath")
	ref := c.Query("ref")

	contents, resp, err := client.ListContents(owner, repo, ref, filepath)
	if err != nil {
		c.JSON(resp.StatusCode, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Data:    contents,
		Message: "Root directory metadata retrieved successfully",
	})
}

// Get File Content
// @Summary	Get File Content
// @Description Get File Content
// @Tags			Gitea
// @Accept			json
// @Produce			json
// @Param			owner		path		string		true	"owner of the repo"
// @Param			repo		path		string		true	"name of the repo"
// @Param			filepath	path		string		true	"The path to the file."
// @Param			ref			query		string		false	"The name of the commit/branch/tag. Default to the repository’s default branch."
// @Success		200		{object}	ResponseHTTP{data=gitea.ContentsResponse}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Security	BearerAuth
// @Router		/api/gitea/repos/{owner}/{repo}/file/{filepath} [get]
func GetRepoFileGitea(c *gin.Context) {
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)
	token, err := utils.GetToken(jwtClaims.UserID)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to retrieve token",
		})
		return
	}
	client, err := gitea.NewClient(config.GetGiteaBaseURL(),
		gitea.SetToken(token),
	)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	owner := c.Param("owner")
	repo := c.Param("repo")
	filepath := c.Param("filepath")
	ref := c.Query("ref")

	content, resp, err := client.GetContents(owner, repo, ref, filepath)
	if err != nil {
		c.JSON(resp.StatusCode, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Data:    content,
		Message: "File content retrieved successfully",
	})
}
