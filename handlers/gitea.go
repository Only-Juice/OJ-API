package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/gin-gonic/gin"

	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/utils"
)

type BasicAuthentication struct {
	Username string `json:"username" validate:"required" example:"username"`
	Password string `json:"password" validate:"required" example:"password"`
}

// Use basic authentication to access the Gitea API
// @Summary	Use basic authentication to access the Gitea API
// @Description Use basic authentication to access the Gitea API
// @Tags			Gitea
// @Accept			json
// @Produce		json
// @Param			BasicAuthentication	body		BasicAuthentication	true	"Basic Authentication"
// @Success		200		{object}	ResponseHTTP{} "Return access token"
// @Failure		503
// @Router			/api/gitea/auth [post]
func PostBasicAuthenticationGitea(c *gin.Context) {
	db := database.DBConn
	account := new(BasicAuthentication)
	if err := c.ShouldBindJSON(account); err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to parse account",
		})
		return
	}

	client, err := gitea.NewClient(config.GetGiteaBaseURL(),
		gitea.SetBasicAuth(account.Username, account.Password),
	)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	giteaUser, _, err := client.GetMyUserInfo()
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		c.Abort()
		return
	}

	var existingUser models.User
	if err := db.Where(&models.User{UserName: giteaUser.UserName}).First(&existingUser).Error; err != nil {
		existingUser = models.User{
			UserName: giteaUser.UserName,
			Email:    giteaUser.Email,
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
			c.JSON(503, ResponseHTTP{
				Success: false,
				Message: err.Error(),
			})
			return
		}
		if err := utils.StoreToken(existingUser.ID, token.Token); err != nil {
			c.JSON(503, ResponseHTTP{
				Success: false,
				Message: err.Error(),
			})
			return
		}
	}

	fail := false
	tokenString, err := utils.GetToken(existingUser.ID)
	if err != nil {
		fail = true
	}

	// Check if the token is valid
	client_check, err := gitea.NewClient(config.GetGiteaBaseURL(),
		gitea.SetToken(tokenString),
	)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}
	_, _, err = client_check.GetMyUserInfo()
	if err != nil || fail {
		client.DeleteAccessToken("OJ-API")
		newToken, _, err := client.CreateAccessToken(gitea.CreateAccessTokenOption{
			Name:   "OJ-API",
			Scopes: []gitea.AccessTokenScope{gitea.AccessTokenScopeAll},
		})
		if err != nil {
			c.JSON(503, ResponseHTTP{
				Success: false,
				Message: err.Error(),
			})
			return
		}
		if err := utils.StoreToken(existingUser.ID, newToken.Token); err != nil {
			c.JSON(503, ResponseHTTP{
				Success: false,
				Message: err.Error(),
			})
			return
		}
	}

	signedToken, err := utils.GenerateJWT(existingUser.ID, existingUser.UserName, giteaUser.IsAdmin)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to generate JWT",
		})
		return
	}
	db.Model(&existingUser).Updates(models.User{
		IsAdmin: giteaUser.IsAdmin,
	})

	c.JSON(200, ResponseHTTP{
		Success: true,
		Data:    signedToken,
		Message: "JWT generated successfully",
	})
}

type BulkCreateUser struct {
	Usernames       []string `json:"usernames" validate:"required" example:"username1,username2"`
	EmailDomain     string   `json:"email_domain" validate:"required" example:"example.com"`
	DefaultPassword string   `json:"default_password" validate:"required" example:"password"`
}

type BulkCreateUserResponse struct {
	SuccessfulUsers []string          `json:"successful_users" example:"username1"`
	FailedUsers     map[string]string `json:"failed_users" example:"username1:error"`
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
	successfulUsers := []string{}
	failedUsers := map[string]string{}

	for _, username := range bulkUsers.Usernames {
		_, _, err := client.AdminCreateUser(gitea.CreateUserOption{
			Email:    username + "@" + bulkUsers.EmailDomain,
			Username: username,
			Password: bulkUsers.DefaultPassword,
		})
		if err != nil {
			failedUsers[username] = err.Error()
		} else {
			successfulUsers = append(successfulUsers, username)
			db.Create(&models.User{
				UserName: username,
				Email:    username + "@" + bulkUsers.EmailDomain,
			})
		}
	}

	c.JSON(200, ResponseHTTP{
		Success: true,
		Data: BulkCreateUserResponse{
			SuccessfulUsers: successfulUsers,
			FailedUsers:     failedUsers,
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

	for _, user := range bulkUsers.User {
		if _, _, err := client.AdminCreateUser(gitea.CreateUserOption{
			Email:              user.Email,
			Username:           user.Username,
			Password:           user.Password,
			MustChangePassword: func(b bool) *bool { return &b }(false),
		}); err != nil {
			failedUsers[user.Username] = err.Error()
			continue
		}

		db.Create(&models.User{
			UserName: user.Username,
			Email:    user.Email,
		})

		successfulUsers = append(successfulUsers, user.Username)
	}
	c.JSON(200, ResponseHTTP{
		Success: true,
		Data: BulkCreateUserResponse{
			SuccessfulUsers: successfulUsers,
			FailedUsers:     failedUsers,
		},
		Message: "Bulk user creation completed",
	})
}

// take a question and create a repository in Gitea
// @Summary	Take a question and create a repository in Gitea
// @Description Take a question and create a repository in Gitea
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

	repo, _, err := client.GetRepo(jwtClaims.Username, parentRepoName)
	if err != nil {
		if _, _, err := client.CreateFork(parentRepoUsername, parentRepoName, gitea.CreateForkOption{
			Name: &parentRepoName,
		}); err != nil {
			c.JSON(503, ResponseHTTP{
				Success: false,
				Message: err.Error(),
			})
			return
		}
		repo = nil
	}

	hooks, _, err := client.ListRepoHooks(jwtClaims.Username, parentRepoName, gitea.ListHooksOptions{})
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to list repository hooks: " + err.Error(),
		})
		return
	}

	scheme := "http"
	if config.Config("USE_TLS") == "true" {
		scheme = "https"
	}
	hookExists := false
	for _, hook := range hooks {
		if hook.Config["url"] == scheme+"://"+config.Config("OJ_HOST")+"/api/gitea" {
			hookExists = true
			break
		}
	}

	if !hookExists {

		if _, _, err := client.CreateRepoHook(jwtClaims.Username, parentRepoName, gitea.CreateHookOption{
			Type:   "gitea",
			Active: true,
			Events: []string{"push"},
			Config: map[string]string{
				"url":          scheme + "://" + config.Config("OJ_HOST") + "/api/gitea",
				"content_type": "json",
			},
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
		Message: "Repository created",
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
