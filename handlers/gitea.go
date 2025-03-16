package handlers

import (
	"OJ-API/config"
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
// @Param			cmd	body		BasicAuthentication	true	"Basic Authentication"
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
