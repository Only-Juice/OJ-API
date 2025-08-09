package utils

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strconv"
	"strings"

	"OJ-API/config"
)

func SendResetEmail(email, token string) error {
	resetLink := fmt.Sprintf("%s/api/user/reset_password?token=%s", config.GetOJBaseURL(), token)
	subject := "[橘測評OJ] 密碼重置 - Password Reset"

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
					⚠️ 此連結將在24小時後失效。如果您並未請求重置密碼，請忽略此郵件。
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
