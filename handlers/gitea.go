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
// @Failure		503		{object}	ResponseHTTP{}
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

	client, err := gitea.NewClient("http://"+config.Config("GIT_HOST"),
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
	client_check, err := gitea.NewClient("http://"+config.Config("GIT_HOST"),
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

	c.JSON(200, ResponseHTTP{
		Success: true,
		Data:    signedToken,
		Message: "Access token created",
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
// @Failure		401		{object}	ResponseHTTP{}
// @Failure		403		{object}	ResponseHTTP{}
// @Failure		503		{object}	ResponseHTTP{}
// @Security	BearerAuth
// @Router		/api/gitea/user/bulk [post]
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

	user := models.User{
		UserName: jwtClaims.Username,
	}
	if err := db.Where(&user).First(&user).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "User not found",
		})
		return
	}

	client, err := gitea.NewClient("http://"+config.Config("GIT_HOST"),
		gitea.SetToken(user.GiteaToken),
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

// take a question and create a repository in Gitea
// @Summary	Take a question and create a repository in Gitea
// @Description Take a question and create a repository in Gitea
// @Tags			Gitea
// @Accept			json
// @Produce			json
// @Param			question_id	path		int		true	"Question ID"
// @Success		200		{object}	ResponseHTTP{}
// @Failure		401		{object}	ResponseHTTP{}
// @Failure		403		{object}	ResponseHTTP{}
// @Failure		503		{object}	ResponseHTTP{}
// @Security	BearerAuth
// @Router		/api/gitea/question/{question_id} [post]
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
	client, err := gitea.NewClient("http://"+config.Config("GIT_HOST"),
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
	if err := db.Where(&models.Question{ID: uint(questionID)}).First(&existingQuestion).Error; err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Question not found",
		})
		return
	}

	parentRepoURLParts := strings.Split(existingQuestion.GitRepoURL, "/")
	parentRepoUsername := parentRepoURLParts[0]
	parentRepoName := parentRepoURLParts[1]

	var userQuestionRelation models.UserQuestionRelation
	if err := db.Where(&models.UserQuestionRelation{
		UserID:     jwtClaims.UserID,
		QuestionID: uint(questionID),
	}).First(&userQuestionRelation).Error; err != nil {
		db.Create(&models.UserQuestionRelation{
			UserID:         jwtClaims.UserID,
			QuestionID:     uint(questionID),
			GitUserRepoURL: jwtClaims.Username + "/" + parentRepoName,
		})
	}

	if _, _, err := client.CreateFork(parentRepoUsername, parentRepoName, gitea.CreateForkOption{
		Name: &parentRepoName,
	}); err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	client.CreateRepoHook(jwtClaims.Username, parentRepoName, gitea.CreateHookOption{
		Type:   "gitea",
		Active: true,
		Events: []string{"push"},
		Config: map[string]string{
			"url":          "http://" + config.Config("OJ_HOST") + "/api/gitea",
			"content_type": "json",
		},
	})

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Repository created",
	})
}
