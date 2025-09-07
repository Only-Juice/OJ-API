package utils

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"OJ-API/config"
)

func SendEmail(to, subject, body string) error {
	// Get SMTP configuration from environment variables
	smtpHost := config.Config("SMTP_HOST")
	smtpPort := config.Config("SMTP_PORT")
	smtpUser := config.Config("SMTP_USER")
	smtpPassword := config.Config("SMTP_PASSWORD")
	fromEmail := config.Config("FROM_EMAIL")

	// Validate required configuration
	if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPassword == "" || fromEmail == "" {
		return fmt.Errorf("missing SMTP configuration: ensure SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD, and FROM_EMAIL are set")
	}

	// Parse port
	port, err := strconv.Atoi(smtpPort)
	if err != nil {
		return fmt.Errorf("invalid SMTP port: %v", err)
	}

	// Extract clean email address for SMTP protocol
	cleanFromEmail := extractEmailAddress(fromEmail)

	// Create message
	message := formatMessage(fromEmail, to, subject, body)

	// SMTP server configuration
	smtpAddr := fmt.Sprintf("%s:%d", smtpHost, port)

	// Setup authentication
	auth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)

	// Check if TLS should be used
	useTLS := config.Config("SMTP_USE_TLS")
	if strings.ToLower(useTLS) == "true" {
		// Use TLS connection - use clean email address for SMTP protocol
		return sendEmailWithTLS(smtpAddr, auth, cleanFromEmail, []string{to}, message)
	} else {
		// Use plain SMTP - use clean email address for SMTP protocol
		return smtp.SendMail(smtpAddr, auth, cleanFromEmail, []string{to}, []byte(message))
	}
}

// sendEmailWithTLS sends email using TLS connection
func sendEmailWithTLS(addr string, auth smtp.Auth, from string, to []string, msg string) error {
	// Create TLS configuration
	tlsConfig := &tls.Config{
		ServerName: strings.Split(addr, ":")[0],
	}

	// Check if TLS verification should be skipped
	skipVerify := config.Config("SMTP_TLS_SKIP_VERIFY")
	if strings.ToLower(skipVerify) == "true" {
		tlsConfig.InsecureSkipVerify = true
	}

	// Connect to the server
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect with TLS: %v", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %v", err)
	}
	defer client.Quit()

	// Authenticate
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %v", err)
		}
	}

	// Set sender
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %v", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %v", recipient, err)
		}
	}

	// Send message
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %v", err)
	}

	_, err = writer.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %v", err)
	}

	return nil
}

// formatMessage formats the email message with proper headers
func formatMessage(from, to, subject, body string) string {
	// Extract email address from "Name <email@domain.com>" format if present
	fromAddr := extractEmailAddress(from)
	fromName := config.Config("FROM_NAME")
	if fromName == "" {
		fromName = "OJ System"
	}

	message := fmt.Sprintf("From: %s <%s>\r\n", fromName, fromAddr)
	message += fmt.Sprintf("To: %s\r\n", to)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-version: 1.0;\r\n"
	message += "Content-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	message += body
	return message
}

// extractEmailAddress extracts email address from "Name <email@domain.com>" format
func extractEmailAddress(emailString string) string {
	// If the string contains < and >, extract the email between them
	if strings.Contains(emailString, "<") && strings.Contains(emailString, ">") {
		start := strings.Index(emailString, "<")
		end := strings.Index(emailString, ">")
		if start < end {
			return emailString[start+1 : end]
		}
	}
	// Otherwise, return the string as is (assuming it's just an email address)
	return emailString
}

func SendResetEmail(email, token string) error {
	resetLink := fmt.Sprintf("%s/api/user/reset_password?token=%s", config.GetOJExternalURL(), url.QueryEscape(token))
	subject := "[橘評測 OJ] 密碼重置 - Password Reset"

	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
				<h1 style="color: white; margin: 0; font-size: 28px;">密碼重置請求</h1>
				<p style="color: #f0f0f0; margin: 10px 0 0 0; font-size: 16px;">Password Reset Request</p>
			</div>
			
			<div style="background: #ffffff; padding: 40px; border: 1px solid #e0e0e0; border-top: none;">
				<p style="font-size: 16px; margin-bottom: 20px;">您好，</p>
				
				<p style="font-size: 16px; margin-bottom: 25px;">
					我們收到了您的密碼重置請求。請點擊下方按鈕來重置您的密碼：
				</p>
				
				<div style="text-align: center; margin: 35px 0;">
					<a href="%s" style="display: inline-block; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; text-decoration: none; padding: 15px 35px; border-radius: 25px; font-size: 16px; font-weight: bold; box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);">
						重置密碼 / Reset Password
					</a>
				</div>
				
				<p style="font-size: 14px; color: #666; margin-top: 30px; padding-top: 20px; border-top: 1px solid #eee;">
					如果按鈕無法點擊，請複製以下連結到瀏覽器：<br>
					<span style="color: #667eea; word-break: break-all;">%s</span>
				</p>
				
				<p style="font-size: 14px; color: #888; margin-top: 25px;">
					⚠️ 此連結將在5分鐘後失效。如果您並未請求重置密碼，請忽略此郵件。
				</p>
			</div>
			
			<div style="background: #f8f9fa; padding: 20px; text-align: center; border-radius: 0 0 10px 10px; border: 1px solid #e0e0e0; border-top: none;">
				<p style="font-size: 12px; color: #999; margin: 0;">
					此郵件由系統自動發送，請勿回覆 | This is an automated email, please do not reply
				</p>
			</div>
		</body>
		</html>
	`, resetLink, resetLink)

	return SendEmail(email, subject, body)
}

func SendPasswordChangeNotification(email, username string, clientInfo *ClientInfo) error {
	subject := "[橘評測 OJ] 密碼變更通知 - Password Change Notification"

	// Format client information for email
	clientInfoText := "未知"
	if clientInfo != nil {
		clientInfoText = fmt.Sprintf(`
					<strong>IP 地址：</strong> %s<br>
					<strong>瀏覽器：</strong> %s<br>
					<strong>作業系統：</strong> %s<br>
					<strong>地點：</strong> %s, %s`,
			clientInfo.IPAddress,
			clientInfo.Browser,
			clientInfo.OS,
			clientInfo.Location,
			clientInfo.Country)
	}

	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
				<h1 style="color: white; margin: 0; font-size: 28px;">密碼變更通知</h1>
				<p style="color: #f0f0f0; margin: 10px 0 0 0; font-size: 16px;">Password Change Notification</p>
			</div>
			
			<div style="background: #ffffff; padding: 40px; border: 1px solid #e0e0e0; border-top: none;">
				<p style="font-size: 16px; margin-bottom: 20px;">親愛的 %s，</p>
				
				<p style="font-size: 16px; margin-bottom: 25px;">
					您的帳戶密碼已成功變更。如果這不是您本人的操作，請立即聯繫管理員。
				</p>
				
				<div style="background: #f8f9fa; padding: 20px; border-radius: 8px; margin: 25px 0;">
					<p style="font-size: 14px; color: #666; margin: 0;">
						<strong>變更時間：</strong> %s
					</p>
				</div>

				<div style="background: #e7f3ff; padding: 20px; border-radius: 8px; margin: 25px 0; border-left: 4px solid #0066cc;">
					<p style="font-size: 14px; color: #0066cc; margin: 0 0 10px 0; font-weight: bold;">🔍 操作來源資訊：</p>
					<p style="font-size: 13px; color: #444; margin: 0;">
						%s
					</p>
				</div>
				
				<p style="font-size: 14px; color: #888; margin-top: 25px;">
					⚠️ 如果您並未進行此操作，請立即聯繫系統管理員以確保帳戶安全。
				</p>
				
				<div style="text-align: center; margin: 35px 0;">
					<a href="%s" style="display: inline-block; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; text-decoration: none; padding: 15px 35px; border-radius: 25px; font-size: 16px; font-weight: bold; box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);">
						前往登入頁面 / Go to Login
					</a>
				</div>
			</div>
			
			<div style="background: #f8f9fa; padding: 20px; text-align: center; border-radius: 0 0 10px 10px; border: 1px solid #e0e0e0; border-top: none;">
				<p style="font-size: 12px; color: #999; margin: 0;">
					此郵件由系統自動發送，請勿回覆 | This is an automated email, please do not reply
				</p>
			</div>
		</body>
		</html>
	`, username, time.Now().Format("2006-01-02 15:04:05"), clientInfoText, config.GetFrontendURL())

	return SendEmail(email, subject, body)
}

func SendPasswordResetNotification(email, username, newPassword string) error {
	subject := "[橘評測 OJ] 密碼重置通知 - Password Reset Notification"

	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
				<h1 style="color: white; margin: 0; font-size: 28px;">密碼重置通知</h1>
				<p style="color: #f0f0f0; margin: 10px 0 0 0; font-size: 16px;">Password Reset Notification</p>
			</div>
			
			<div style="background: #ffffff; padding: 40px; border: 1px solid #e0e0e0; border-top: none;">
				<p style="font-size: 16px; margin-bottom: 20px;">親愛的 %s，</p>
				
				<p style="font-size: 16px; margin-bottom: 25px;">
					管理員已為您重置密碼。您的新密碼如下：
				</p>
				
				<div style="background: #f8f9fa; padding: 20px; border-radius: 8px; margin: 25px 0; text-align: center;">
					<p style="font-size: 18px; color: #333; margin: 0; font-weight: bold; font-family: monospace;">
						新密碼：<span style="background: #e9ecef; padding: 5px 10px; border-radius: 4px;">%s</span>
					</p>
				</div>
				
				<div style="background: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; border-radius: 8px; margin: 25px 0;">
					<p style="font-size: 14px; color: #856404; margin: 0;">
						<strong>⚠️ 安全提醒：</strong><br>
						• 請立即登入並變更為您個人的密碼<br>
						• 請勿與他人分享此密碼<br>
						• 建議使用包含英文、數字和特殊符號的強密碼
					</p>
				</div>
				
				<p style="font-size: 14px; color: #666; margin-top: 30px; padding-top: 20px; border-top: 1px solid #eee;">
					重置時間：%s
				</p>
				
				<div style="text-align: center; margin: 35px 0;">
					<a href="%s" style="display: inline-block; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; text-decoration: none; padding: 15px 35px; border-radius: 25px; font-size: 16px; font-weight: bold; box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);">
						立即登入 / Login Now
					</a>
				</div>
			</div>
			
			<div style="background: #f8f9fa; padding: 20px; text-align: center; border-radius: 0 0 10px 10px; border: 1px solid #e0e0e0; border-top: none;">
				<p style="font-size: 12px; color: #999; margin: 0;">
					此郵件由系統自動發送，請勿回覆 | This is an automated email, please do not reply
				</p>
			</div>
		</body>
		</html>
	`, username, newPassword, time.Now().Format("2006-01-02 15:04:05"), config.GetFrontendURL())

	return SendEmail(email, subject, body)
}

func SendDefaultPasswordNotification(email, username, newPassword string) error {
	subject := "[橘評測 OJ] 預設密碼通知 - Default Password Notification"

	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
			<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
				<h1 style="color: white; margin: 0; font-size: 28px;">預設密碼通知</h1>
				<p style="color: #f0f0f0; margin: 10px 0 0 0; font-size: 16px;">Default Password Notification</p>
			</div>
			
			<div style="background: #ffffff; padding: 40px; border: 1px solid #e0e0e0; border-top: none;">
				<p style="font-size: 16px; margin-bottom: 20px;">親愛的 %s，</p>
				
				<p style="font-size: 16px; margin-bottom: 25px;">
					您的帳戶已由管理員建立。以下是您的預設登入資訊：
				</p>
				
				<div style="background: #f8f9fa; padding: 20px; border-radius: 8px; margin: 25px 0; text-align: center;">
					<p style="font-size: 18px; color: #333; margin: 0; font-weight: bold; font-family: monospace;">
						使用者名稱：<span style="background: #e9ecef; padding: 5px 10px; border-radius: 4px;">%s</span><br><br>
						預設密碼：<span style="background: #e9ecef; padding: 5px 10px; border-radius: 4px;">%s</span>
					</p>
				</div>
				
				<div style="background: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; border-radius: 8px; margin: 25px 0;">
					<p style="font-size: 14px; color: #856404; margin: 0;">
						<strong>⚠️ 安全提醒：</strong><br>
						• 請於首次登入後立即變更此預設密碼<br>
						• 請勿與他人分享此密碼<br>
						• 建議使用包含英文、數字和特殊符號的強密碼
					</p>
				</div>
				
				<p style="font-size: 14px; color: #666; margin-top: 30px; padding-top: 20px; border-top: 1px solid #eee;">
					帳戶建立時間：%s
				</p>
				
				<div style="text-align: center; margin: 35px 0;">
					<a href="%s" style="display: inline-block; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; text-decoration: none; padding: 15px 35px; border-radius: 25px; font-size: 16px; font-weight: bold; box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);">
						立即登入 / Login Now
					</a>
				</div>
			</div>
			
			<div style="background: #f8f9fa; padding: 20px; text-align: center; border-radius: 0 0 10px 10px; border: 1px solid #e0e0e0; border-top: none;">
				<p style="font-size: 12px; color: #999; margin: 0;">
					此郵件由系統自動發送，請勿回覆 | This is an automated email, please do not reply
				</p>
			</div>
		</body>
		</html>
	`, username, username, newPassword, time.Now().Format("2006-01-02 15:04:05"), config.GetFrontendURL())

	return SendEmail(email, subject, body)
}
