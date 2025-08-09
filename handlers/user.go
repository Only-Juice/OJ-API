package handlers

import (
	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/utils"
	"net/http"

	"code.gitea.io/sdk/gitea"
	"github.com/gin-gonic/gin"
)

type IsPublicRequest struct {
	IsPublic bool `json:"is_public"`
}

// Post User is_public setting
// @Summary Set user is_public
// @Description Update the user's is_public setting
// @Tags User
// @Accept json
// @Produce json
// @Param is_public body handlers.IsPublicRequest true "is_public"
// @Success 200 {object} ResponseHTTP{data=models.User}
// @Failure 400 {object} ResponseHTTP{}
// @Failure 401 {object} ResponseHTTP{}
// @Failure 404 {object} ResponseHTTP{}
// @Failure 503 {object} ResponseHTTP{}
// @Router /api/user/is_public [post]
// @Security BearerAuth
func PostUserIsPublic(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)

	// Find the user by ID
	var user models.User
	if err := db.First(&user, jwtClaims.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "User not found",
		})
		return
	}

	// Bind the request body to IsPublicRequest
	var req IsPublicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// Update the user's is_public field
	user.IsPublic = req.IsPublic
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ResponseHTTP{
			Success: false,
			Message: "Failed to update user",
		})
		return
	}

	// Respond with success
	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Message: "User is_public updated successfully",
		Data: models.User{
			ID:       user.ID,
			UserName: user.UserName,
			Enable:   user.Enable,
			Email:    user.Email,
			IsPublic: user.IsPublic,
		},
	})
}

type GetUserData struct {
	ID       uint   `json:"id"`
	UserName string `json:"user_name"`
	Enable   bool   `json:"enable"`
	Email    string `json:"email"`
	IsPublic bool   `json:"is_public"`
	IsAdmin  bool   `json:"is_admin"`
}

// Get User Info
// @Summary Get user info
// @Description Get user info by ID
// @Tags User
// @Accept json
// @Produce json
// @Success 200 {object} ResponseHTTP{data=GetUserData}
// @Failure 400 {object} ResponseHTTP{}
// @Failure 401 {object} ResponseHTTP{}
// @Failure 404 {object} ResponseHTTP{}
// @Failure 503 {object} ResponseHTTP{}
// @Router /api/user [get]
// @Security BearerAuth
func GetUser(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)

	// Find the user by ID
	var user models.User
	if err := db.First(&user, jwtClaims.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "User not found",
		})
		return
	}

	// Respond with user info
	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Message: "User info retrieved successfully",
		Data: GetUserData{
			ID:       user.ID,
			UserName: user.UserName,
			Enable:   user.Enable,
			Email:    user.Email,
			IsPublic: user.IsPublic,
			IsAdmin:  user.IsAdmin,
		},
	})
}

type ChangeUserPasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// Change User Password
// @Summary Change user password
// @Description Change user password
// @Tags User
// @Accept json
// @Produce json
// @Param request body ChangeUserPasswordRequest true "ChangeUserPasswordRequest"
// @Success 200 {object} ResponseHTTP{}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Security	BearerAuth
// @Router		/api/user/change_password [post]
func ChangeUserPassword(c *gin.Context) {
	db := database.DBConn
	jwtClaims := c.Request.Context().Value(models.JWTClaimsKey).(*utils.JWTClaims)

	request := ChangeUserPasswordRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	client_check, err := gitea.NewClient(config.GetGiteaBaseURL(),
		gitea.SetBasicAuth(jwtClaims.Username, request.OldPassword),
	)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	_, _, err = client_check.GetMyUserInfo()
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to authenticate with Gitea",
		})
		return
	}

	// Find Admin user
	var adminUser models.User
	if err := db.First(&adminUser, models.User{
		IsAdmin: true,
	}).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "User not found",
		})
		return
	}
	// Update password
	token, err := utils.GetToken(adminUser.ID)
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
	_, err = client.AdminEditUser(jwtClaims.Username, gitea.EditUserOption{
		LoginName: jwtClaims.Username,
		Password:  request.NewPassword,
	})
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	// Get user email for notification
	var user models.User
	if err := db.First(&user, models.User{UserName: jwtClaims.Username}).Error; err == nil {
		// Send password change notification email
		if err := utils.SendPasswordChangeNotification(user.Email, user.UserName, utils.GetClientInfo(c)); err != nil {
			// Log error but don't fail the request
			utils.Warnf("Failed to send password change notification email to %s: %v", user.Email, err)
		}
	}

	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Message: "Password changed successfully",
	})
}

type ForgetPasswordRequest struct {
	Email string `json:"email" binding:"required,email" example:"username@example.com"`
}

// Forget Password
// @Summary Forget password
// @Description Forget password
// @Tags User
// @Accept json
// @Produce json
// @Param request body ForgetPasswordRequest true "ForgetPasswordRequest"
// @Success 200 {object} ResponseHTTP{}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Router		/api/user/forget_password [post]
func ForgetPassword(c *gin.Context) {
	db := database.DBConn

	request := ForgetPasswordRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// Find the user by email
	var user models.User
	if err := db.Where("email = ?", request.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "User not found",
		})
		return
	}

	// Generate a reset token
	resetToken, err := utils.GenerateResetToken(user.ID)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to generate reset token",
		})
		return
	}

	// Send reset email
	if err := utils.SendResetEmail(user.Email, resetToken); err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Message: "Reset email sent successfully",
	})
}

type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// Reset Password Page (GET)
// @Summary Show password reset page
// @Description Show password reset form page
// @Tags User
// @Accept html
// @Produce html
// @Param token query string true "Reset token"
// @Success 200 {string} string "HTML page"
// @Failure		400
// @Router		/api/user/reset_password [get]
func ResetPasswordPage(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusBadRequest, `
<!DOCTYPE html>
<html lang="zh-TW">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>密碼重置 - 橘測評OJ</title>
	<style>
		body { font-family: 'Arial', sans-serif; margin: 0; padding: 20px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); min-height: 100vh; display: flex; align-items: center; justify-content: center; }
		.container { background: white; padding: 40px; border-radius: 10px; box-shadow: 0 15px 35px rgba(0,0,0,0.1); max-width: 400px; width: 100%; text-align: center; }
		.error { color: #e74c3c; font-size: 18px; margin-bottom: 20px; }
		.logo { font-size: 24px; font-weight: bold; color: #667eea; margin-bottom: 20px; }
	</style>
</head>
<body>
	<div class="container">
		<div class="logo">橘測評 OJ</div>
		<div class="error">❌ 無效的重置連結</div>
		<p>重置代碼遺失或無效，請重新申請密碼重置。</p>
	</div>
</body>
</html>`)
		return
	}

	// Validate reset token
	_, err := utils.ValidateResetToken(token)
	if err != nil {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusUnauthorized, `
<!DOCTYPE html>
<html lang="zh-TW">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>密碼重置 - 橘測評OJ</title>
	<style>
		body { font-family: 'Arial', sans-serif; margin: 0; padding: 20px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); min-height: 100vh; display: flex; align-items: center; justify-content: center; }
		.container { background: white; padding: 40px; border-radius: 10px; box-shadow: 0 15px 35px rgba(0,0,0,0.1); max-width: 400px; width: 100%; text-align: center; }
		.error { color: #e74c3c; font-size: 18px; margin-bottom: 20px; }
		.logo { font-size: 24px; font-weight: bold; color: #667eea; margin-bottom: 20px; }
	</style>
</head>
<body>
	<div class="container">
		<div class="logo">橘測評 OJ</div>
		<div class="error">❌ 無效或過期的重置連結</div>
		<p>重置代碼無效或已過期，請重新申請密碼重置。</p>
	</div>
</body>
</html>`)
		return
	}

	// Show password reset form
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, `
<!DOCTYPE html>
<html lang="zh-TW">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>重設密碼 - 橘測評OJ</title>
	<style>
		body { font-family: 'Arial', sans-serif; margin: 0; padding: 20px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); min-height: 100vh; display: flex; align-items: center; justify-content: center; }
		.container { background: white; padding: 40px; border-radius: 10px; box-shadow: 0 15px 35px rgba(0,0,0,0.1); max-width: 400px; width: 100%; }
		.logo { text-align: center; font-size: 24px; font-weight: bold; color: #667eea; margin-bottom: 30px; }
		.form-group { margin-bottom: 20px; }
		label { display: block; margin-bottom: 8px; color: #333; font-weight: bold; }
		input[type="password"] { width: 100%; padding: 12px; border: 2px solid #e0e0e0; border-radius: 5px; font-size: 16px; transition: border-color 0.3s; box-sizing: border-box; }
		input[type="password"]:focus { outline: none; border-color: #667eea; }
		.btn { width: 100%; padding: 12px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; border: none; border-radius: 5px; font-size: 16px; font-weight: bold; cursor: pointer; transition: transform 0.2s; }
		.btn:hover { transform: translateY(-2px); }
		.message { margin-top: 15px; padding: 10px; border-radius: 5px; text-align: center; }
		.success { background: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
		.error { background: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
		.requirements { font-size: 12px; color: #666; margin-top: 5px; }
	</style>
</head>
<body>
	<div class="container">
		<div class="logo">🍊 橘測評 OJ</div>
		<h2 style="text-align: center; color: #333; margin-bottom: 30px;">重設密碼</h2>
		
		<form id="resetForm">
			<div class="form-group">
				<label for="newPassword">新密碼</label>
				<input type="password" id="newPassword" name="new_password" required minlength="6">
				<div class="requirements">密碼長度至少6位字符</div>
			</div>
			
			<div class="form-group">
				<label for="confirmPassword">確認新密碼</label>
				<input type="password" id="confirmPassword" name="confirm_password" required minlength="6">
			</div>
			
			<button type="submit" class="btn">重設密碼</button>
		</form>
		
		<div id="message" class="message" style="display: none;"></div>
	</div>

	<script>
		document.getElementById('resetForm').addEventListener('submit', async function(e) {
			e.preventDefault();
			
			const newPassword = document.getElementById('newPassword').value;
			const confirmPassword = document.getElementById('confirmPassword').value;
			const messageDiv = document.getElementById('message');
			
			// Validate passwords match
			if (newPassword !== confirmPassword) {
				messageDiv.className = 'message error';
				messageDiv.textContent = '密碼確認不一致';
				messageDiv.style.display = 'block';
				return;
			}
			
			// Validate password length
			if (newPassword.length < 6) {
				messageDiv.className = 'message error';
				messageDiv.textContent = '密碼長度至少6位字符';
				messageDiv.style.display = 'block';
				return;
			}
			
			try {
				const response = await fetch(window.location.href, {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json',
					},
					body: JSON.stringify({
						new_password: newPassword
					})
				});
				
				const result = await response.json();
				
				if (result.success) {
					messageDiv.className = 'message success';
					messageDiv.textContent = '密碼重設成功！請使用新密碼登入。';
					messageDiv.style.display = 'block';
					
					// Disable form
					document.getElementById('resetForm').style.display = 'none';
				} else {
					messageDiv.className = 'message error';
					messageDiv.textContent = result.message || '密碼重設失敗';
					messageDiv.style.display = 'block';
				}
			} catch (error) {
				messageDiv.className = 'message error';
				messageDiv.textContent = '網路錯誤，請稍後再試';
				messageDiv.style.display = 'block';
			}
		});
	</script>
</body>
</html>`)
}

// Reset Password (POST)
// @Summary Reset password
// @Description Handle password reset with token
// @Tags User
// @Accept json
// @Produce json
// @Param token query string true "Reset token"
// @Param request body ResetPasswordRequest true "Reset password request"
// @Success 200 {object} ResponseHTTP{}
// @Failure		400
// @Failure		401
// @Failure		404
// @Failure		503
// @Router		/api/user/reset_password [post]
func ResetPassword(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: "Missing reset token",
		})
		return
	}

	// Handle POST request for password reset
	db := database.DBConn

	request := ResetPasswordRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// Validate reset token
	userID, err := utils.ValidateResetToken(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, ResponseHTTP{
			Success: false,
			Message: "Invalid or expired reset token",
		})
		return
	}

	// Find the user
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "User not found",
		})
		return
	}

	// Find Admin user for Gitea operations
	var adminUser models.User
	if err := db.First(&adminUser, models.User{
		IsAdmin: true,
	}).Error; err != nil {
		c.JSON(http.StatusNotFound, ResponseHTTP{
			Success: false,
			Message: "Admin user not found",
		})
		return
	}

	// Update password in Gitea
	adminToken, err := utils.GetToken(adminUser.ID)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	client, err := gitea.NewClient(config.GetGiteaBaseURL(),
		gitea.SetToken(adminToken),
	)
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	_, err = client.AdminEditUser(user.UserName, gitea.EditUserOption{
		LoginName: user.UserName,
		Password:  request.NewPassword,
	})
	if err != nil {
		c.JSON(503, ResponseHTTP{
			Success: false,
			Message: "Failed to update password in Gitea",
		})
		return
	}

	c.JSON(http.StatusOK, ResponseHTTP{
		Success: true,
		Message: "Password reset successfully",
	})
}
