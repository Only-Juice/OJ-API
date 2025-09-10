package handlers

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"

	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/utils"
)

// Helper function to set cookies with proper CORS configuration
func setCrossDomainCookie(c *gin.Context, name, value string, maxAge int) {
	// Check if request is from HTTPS
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"

	// For development environment or non-HTTPS, use Lax instead of None
	sameSite := "Lax"
	secure := ""

	// Only use SameSite=None with Secure for HTTPS cross-origin requests
	origin := c.GetHeader("Origin")
	if isSecure && origin != "" {
		sameSite = "None"
		secure = "; Secure"
	}

	cookieStr := fmt.Sprintf("%s=%s; Max-Age=%d; Path=/; HttpOnly; SameSite=%s%s",
		name, value, maxAge, sameSite, secure)
	c.Writer.Header().Add("Set-Cookie", cookieStr)
}

type LoginRequest struct {
	Username string `json:"username" example:"username"`
	Password string `json:"password" example:"password"`
	Token    string `json:"token" example:""` // Optional token for API access
}

type LoginResponse struct {
	AccessToken   string `json:"access_token"`
	RefreshToken  string `json:"refresh_token"`
	ResetPassword bool   `json:"reset_password"`
}

// Use basic authentication or token to access the Gitea API
// @Summary	User login with username and password
// @Description Use basic authentication or token to login and get access token and refresh token
// @Tags		Auth
// @Accept		json
// @Produce	json
// @Param		LoginRequest	body		LoginRequest	true	"Login Request"
// @Success	200		{object}	ResponseHTTP{data=LoginResponse} "Return access token and refresh token"
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

	var client *gitea.Client
	var err error
	if account.Token != "" {
		client, err = gitea.NewClient(config.GetGiteaBaseURL(),
			gitea.SetToken(account.Token),
		)
	} else {
		client, err = gitea.NewClient(config.GetGiteaBaseURL(),
			gitea.SetBasicAuth(account.Username, account.Password),
		)
	}
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
	if !existingUser.Enable {
		c.JSON(401, ResponseHTTP{
			Success: false,
			Message: "User is disabled",
		})
		return
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

	var client_check *gitea.Client
	tokenString, err := utils.GetToken(existingUser.ID)
	if err == nil {
		// Check if the token is valid
		client_check, err = gitea.NewClient(config.GetGiteaBaseURL(),
			gitea.SetToken(tokenString),
		)
		if err == nil {
			_, _, err = client_check.GetMyUserInfo()
		}
	}

	if err != nil {
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

	// Store refresh token in database
	db.Model(&existingUser).Updates(models.User{
		IsAdmin:      giteaUser.IsAdmin,
		RefreshToken: refreshToken,
	})

	// Set both tokens as cookies with proper CORS configuration
	setCrossDomainCookie(c, "access_token", accessToken, 15*60)       // 15 minutes
	setCrossDomainCookie(c, "refresh_token", refreshToken, 7*24*3600) // 7 days

	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Login successfully",
		Data: LoginResponse{
			AccessToken:   accessToken,
			RefreshToken:  refreshToken,
			ResetPassword: existingUser.ResetPassword,
		},
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
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			c.JSON(401, ResponseHTTP{
				Success: false,
				Message: "Invalid authorization header format",
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
	if err := db.Where("id = ? AND refresh_token = ?", claims.UserID, refreshToken, true).First(&user).Error; err != nil {
		c.JSON(401, ResponseHTTP{
			Success: false,
			Message: "Invalid refresh token",
		})
		return
	}
	if !user.Enable {
		c.JSON(401, ResponseHTTP{
			Success: false,
			Message: "User is disabled",
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

	// Set new access token cookie with proper CORS configuration
	setCrossDomainCookie(c, "access_token", accessToken, 15*60) // 15 minutes

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
		if strings.HasPrefix(authHeader, "Bearer ") {
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

	// Clear cookies with proper CORS configuration
	setCrossDomainCookie(c, "access_token", "", -1)
	setCrossDomainCookie(c, "refresh_token", "", -1)

	internal := c.GetBool("internal")
	if !internal {
		c.JSON(200, ResponseHTTP{
			Success: true,
			Message: "Logout successful",
		})
	}
}

// getOAuth2Config returns the OAuth2 configuration for Gitea
func getOAuth2Config() *oauth2.Config {
	giteaConfig := config.GetGiteaOAuthConfig()

	return &oauth2.Config{
		ClientID:     giteaConfig.ClientID,
		ClientSecret: giteaConfig.ClientSecret,
		RedirectURL:  config.Config("OAUTH_CALLBACK_URL"),
		Scopes:       []string{"read:user"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  giteaConfig.URL + "/login/oauth/authorize",
			TokenURL: giteaConfig.URL + "/login/oauth/access_token",
		},
	}
}

// Gitea OAuth callback
// @Summary	Handle Gitea OAuth callback
// @Description Handle OAuth callback and complete authentication
// @Tags		Auth
// @Produce	json
// @Param		code	query		string	true	"Authorization code"
// @Param		state	query		string	true	"State parameter"
// @Success	200	{object}	ResponseHTTP{} "OAuth login successful"
// @Failure	400	{object}	ResponseHTTP{} "Bad request"
// @Failure	401	{object}	ResponseHTTP{} "Unauthorized"
// @Failure	500	{object}	ResponseHTTP{} "Server error"
// @Router		/api/auth/oauth/callback [get]
func OAuthCallback(c *gin.Context) {
	db := database.DBConn

	// Get authorization code
	code := c.Query("code")
	if code == "" {
		c.JSON(400, ResponseHTTP{
			Success: false,
			Message: "Missing authorization code",
		})
		return
	}

	// Exchange code for token
	oauthConfig := getOAuth2Config()

	// Create custom HTTP client that skips TLS verification if configured
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.Config("TLS_SKIP_VERIFY") == "true",
		},
	}
	httpClient := &http.Client{Transport: tr}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)

	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		c.JSON(500, ResponseHTTP{
			Success: false,
			Message: "Failed to exchange authorization code for token",
		})
		return
	}

	// Validate token expiry
	if token.Expiry.Before(time.Now()) {
		c.JSON(400, ResponseHTTP{
			Success: false,
			Message: "OAuth token has expired",
		})
		return
	}

	// Get user info from Gitea using the access token
	giteaTr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.Config("TLS_SKIP_VERIFY") == "true",
		},
	}
	giteaHttpClient := &http.Client{Transport: giteaTr}

	client, err := gitea.NewClient(config.GetGiteaOAuthConfig().URL,
		gitea.SetToken(token.AccessToken),
		gitea.SetHTTPClient(giteaHttpClient),
	)
	if err != nil {
		c.JSON(500, ResponseHTTP{
			Success: false,
			Message: "Failed to create Gitea client",
		})
		return
	}

	giteaUser, _, err := client.GetMyUserInfo()
	if err != nil {
		c.JSON(500, ResponseHTTP{
			Success: false,
			Message: "Failed to retrieve user info from Gitea",
		})
		return
	}

	// Find or create user in database
	var existingUser models.User
	if err := db.Where(&models.User{UserName: giteaUser.UserName}).First(&existingUser).Error; err != nil {
		existingUser = models.User{
			UserName: giteaUser.UserName,
			Email:    giteaUser.Email,
			IsAdmin:  giteaUser.IsAdmin,
		}
		db.Create(&existingUser)
	} else {
		// Update user info
		db.Model(&existingUser).Updates(models.User{
			Email:   giteaUser.Email,
			IsAdmin: giteaUser.IsAdmin,
		})
	}

	if existingUser.GiteaToken == "" {
		c.JSON(400, ResponseHTTP{
			Success: false,
			Message: "No Gitea token found",
		})
		return
	}

	// check Gitea Login
	var giteaToken string
	giteaToken, err = utils.GetToken(existingUser.ID)
	if err != nil {
		c.JSON(400, ResponseHTTP{
			Success: false,
			Message: "Failed to get Gitea token",
		})
		return
	}
	client, err = gitea.NewClient(config.GetGiteaBaseURL(),
		gitea.SetToken(giteaToken),
	)
	if err != nil {
		c.JSON(400, ResponseHTTP{
			Success: false,
			Message: "Failed to create Gitea client",
		})
		return
	}
	giteaUser, _, err = client.GetMyUserInfo()
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		c.Abort()
		return
	}

	// Generate JWT tokens
	accessToken, refreshToken, err := utils.GenerateTokens(existingUser.ID, existingUser.UserName, giteaUser.IsAdmin)
	if err != nil {
		c.JSON(500, ResponseHTTP{
			Success: false,
			Message: "Failed to generate tokens",
		})
		return
	}

	// Update refresh token in database
	db.Model(&existingUser).Update("refresh_token", refreshToken)

	// Set tokens as cookies with enhanced security
	setCrossDomainCookie(c, "access_token", accessToken, 15*60)       // 15 minutes
	setCrossDomainCookie(c, "refresh_token", refreshToken, 7*24*3600) // 7 days
	c.JSON(200, ResponseHTTP{
		Success: true,
		Message: "Login successful",
		Data:    giteaUser.IsAdmin,
	})
}
