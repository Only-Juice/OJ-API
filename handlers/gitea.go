package handlers

import (
	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	"encoding/json"
	"net/http"

	"code.gitea.io/sdk/gitea"
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
// @Success		200		{object}	ResponseHTTP{data=gitea.AccessToken} "Return access token"
// @Failure		503		{object}	ResponseHTTP{}
// @Router			/api/gitea/auth [post]
func PostBasicAuthenticationGitea(w http.ResponseWriter, r *http.Request) {
	account := new(BasicAuthentication)
	if err := json.NewDecoder(r.Body).Decode(account); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "Failed to parse account",
		})
		return
	}

	if client, err := gitea.NewClient("http://"+config.Config("GIT_HOST"),
		gitea.SetBasicAuth(
			account.Username, account.Password,
		),
	); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
	} else {
		client.DeleteAccessToken("OJ-API")

		if token, _, err := client.CreateAccessToken(gitea.CreateAccessTokenOption{
			Name:   "OJ-API",
			Scopes: []gitea.AccessTokenScope{gitea.AccessTokenScopeAll},
		}); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(ResponseHTTP{
				Success: false,
				Message: "Failed to create access token",
			})
		} else {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(ResponseHTTP{
				Success: true,
				Data:    token,
				Message: "Access token created",
			})
		}
	}

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
// @Failure		403		{object}	ResponseHTTP{}
// @Failure		503		{object}	ResponseHTTP{}
// @Security	AuthorizationHeaderToken
// @Router		/api/gitea/user/bulk [post]
func PostBulkCreateUserGitea(w http.ResponseWriter, r *http.Request) {
	db := database.DBConn
	user := r.Context().Value(models.UserContextKey).(*gitea.User)
	if !user.IsAdmin {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "Permission denied",
		})
		return
	}
	bulkUsers := new(BulkCreateUser)
	if err := json.NewDecoder(r.Body).Decode(bulkUsers); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(ResponseHTTP{
			Success: false,
			Message: "Failed to parse bulk users",
		})
		return
	}

	client := r.Context().Value(models.ClientContextKey).(*gitea.Client)
	successfulUsers := []string{}
	failedUsers := map[string]string{}

	for _, username := range bulkUsers.Usernames {
		if _, _, err := client.AdminCreateUser(gitea.CreateUserOption{
			Email:    username + "@" + bulkUsers.EmailDomain,
			Username: username,
			Password: bulkUsers.DefaultPassword,
		}); err != nil {
			failedUsers[username] = err.Error()
		} else {
			successfulUsers = append(successfulUsers, username)
			db.Create(&models.User{
				UserName: username,
				Email:    username + "@" + bulkUsers.EmailDomain,
			})
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ResponseHTTP{
		Success: true,
		Data: BulkCreateUserResponse{
			SuccessfulUsers: successfulUsers,
			FailedUsers:     failedUsers,
		},
		Message: "Bulk user creation completed",
	})
}
