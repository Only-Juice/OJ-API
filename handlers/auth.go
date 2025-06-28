package handlers

import (
	"net/http"

	"code.gitea.io/sdk/gitea"
	"github.com/gin-gonic/gin"

	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/utils"
)

type LoginRequest struct {
	Username string `json:"username" example:"username"`
	Password string `json:"password" example:"password"`
	Token    string `json:"token" example:""` // Optional token for API access
}

// Use basic authentication or token to access the Gitea API
// @Summary	User login with username and password
// @Description Use basic authentication or token to login and get access token and refresh token
// @Tags		Auth
// @Accept		json
// @Produce	json
// @Param		LoginRequest	body		LoginRequest	true	"Login Request"
// @Success	200		{object}	ResponseHTTP{} "Return access token and refresh token"
// @Failure	503
// @Router		/api/auth/login [post]
func AuthBasic(c *gin.Context) {
	db := database.DBConn
	account := new(LoginRequest)
	if err := c.ShouldBindJSON(account); err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to parse account",
		})
		return
	}

	client, err := gitea.NewClient("http://"+config.Config("GIT_HOST"),
		gitea.SetBasicAuth(account.Username, account.Password),
		gitea.SetToken(account.Token),
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

	accessToken, refreshToken, err := utils.GenerateTokens(existingUser.ID, existingUser.UserName, giteaUser.IsAdmin)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to generate tokens",
		})
		return
	}

	db.Model(&existingUser).Updates(models.User{
		IsAdmin: giteaUser.IsAdmin,
	})

	// Set both tokens as cookies
	c.SetCookie("access_token", accessToken, 15*60, "/", "", false, true)       // 15 minutes
	c.SetCookie("refresh_token", refreshToken, 7*24*3600, "/", "", false, true) // 7 days

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Login successfully",
	})
}

// Refresh access token using refresh token
// @Summary	Refresh access token
// @Description Use refresh token to get a new access token
// @Tags		Auth
// @Accept		json
// @Produce	json
// @Success	200	{object}	ResponseHTTP{} "Return new access token"
// @Failure	401	{object}	ResponseHTTP{} "Invalid refresh token"
// @Failure	503	{object}	ResponseHTTP{} "Server error"
// @Router		/api/auth/refresh [post]
func RefreshToken(c *gin.Context) {
	db := database.DBConn

	// Get refresh token from cookie or header
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		// Try to get from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, ResponseHTTP{
				Success: false,
				Message: "No refresh token provided",
			})
			return
		}
		refreshToken = authHeader[7:] // Remove "Bearer " prefix
	}

	// Validate refresh token
	claims, err := utils.ValidateRefreshToken(refreshToken)
	if err != nil {
		c.JSON(401, ResponseHTTP{
			Success: false,
			Message: "Invalid refresh token",
		})
		return
	}

	// Check if user exists and refresh token matches
	var user models.User
	if err := db.Where("id = ? AND refresh_token = ?", claims.UserID, refreshToken).First(&user).Error; err != nil {
		c.JSON(401, ResponseHTTP{
			Success: false,
			Message: "Invalid refresh token",
		})
		return
	}

	// Generate new access token
	accessToken, err := utils.GenerateAccessToken(user.ID, user.UserName, user.IsAdmin)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to generate access token",
		})
		return
	}

	// Set new access token cookie
	c.SetCookie("access_token", accessToken, 15*60, "/", "", false, true) // 15 minutes

	c.JSON(200, ResponseHTTP{
		Success: true,
		Data: gin.H{
			"access_token": accessToken,
		},
		Message: "Access token refreshed successfully",
	})
}

// Logout user and invalidate tokens
// @Summary	Logout user
// @Description Logout user and invalidate refresh token
// @Tags		Auth
// @Accept		json
// @Produce	json
// @Success	200	{object}	ResponseHTTP{} "Logout successful"
// @Failure	401	{object}	ResponseHTTP{} "Unauthorized"
// @Router		/api/auth/logout [post]
func Logout(c *gin.Context) {
	db := database.DBConn

	// Get refresh token
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			refreshToken = authHeader[7:] // Remove "Bearer " prefix
		}
	}

	if refreshToken != "" {
		// Validate and get user info
		claims, err := utils.ValidateRefreshToken(refreshToken)
		if err == nil {
			// Clear refresh token from database
			db.Model(&models.User{}).Where("id = ?", claims.UserID).Update("refresh_token", "")
		}
	}

	// Clear cookies
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Logout successful",
	})
}
