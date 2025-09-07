package utils

import "OJ-API/config"

func MissingOrInvalidTokenPage() string {
	return `
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Password Reset - Orange Juice OJ</title>
	<style>
		* { box-sizing: border-box; margin: 0; padding: 0; }
		body { 
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
			margin: 0; 
			padding: 20px; 
			background: #fafafa;
			min-height: 100vh; 
			display: flex; 
			align-items: center; 
			justify-content: center;
			color: #15191e;
			transition: background-color 0.3s, color 0.3s;
		}
		.container { 
			background: white; 
			padding: 48px; 
			border-radius: 12px; 
			border: 1px solid #e5e5e5;
			box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
			max-width: 400px; 
			width: 100%; 
			text-align: center;
			transition: background-color 0.3s, border-color 0.3s, box-shadow 0.3s;
		}
		.error { 
			color: #dc2626; 
			font-size: 18px; 
			margin-bottom: 20px; 
			font-weight: 500;
			transition: color 0.3s;
		}
		.logo { 
			font-size: 28px; 
			font-weight: 600; 
			color: #000; 
			margin-bottom: 32px;
			transition: color 0.3s;
		}
		p {
			color: #666;
			line-height: 1.6;
			margin-bottom: 0;
			transition: color 0.3s;
		}
		
		@media (prefers-color-scheme: dark) {
			body {
				background: #191e24;
				color: #e5e5e5;
			}
			.container {
				background: #1d232a;
				border: 1px solid #15191e;
				box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
			}
			.logo {
				color: #fff;
			}
			p {
				color: #a3a3a3;
			}
			.error {
				color: #f87171;
			}
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="logo">Orange Juice OJ</div>
		<div class="error">❌ Invalid Reset Link</div>
		<p>The reset token is missing or invalid. Please request a new password reset.</p>
	</div>
</body>
</html>`
}

func ExpiredOrUsedTokenPage() string {
	return `
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Password Reset - Orange Juice OJ</title>
	<style>
		* { box-sizing: border-box; margin: 0; padding: 0; }
		body { 
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
			margin: 0; 
			padding: 20px; 
			background: #fafafa;
			min-height: 100vh; 
			display: flex; 
			align-items: center; 
			justify-content: center;
			color: #15191e;
			transition: background-color 0.3s, color 0.3s;
		}
		.container { 
			background: white; 
			padding: 48px; 
			border-radius: 12px; 
			border: 1px solid #e5e5e5;
			box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
			max-width: 400px; 
			width: 100%; 
			text-align: center;
			transition: background-color 0.3s, border-color 0.3s, box-shadow 0.3s;
		}
		.error { 
			color: #dc2626; 
			font-size: 18px; 
			margin-bottom: 20px; 
			font-weight: 500;
			transition: color 0.3s;
		}
		.logo { 
			font-size: 28px; 
			font-weight: 600; 
			color: #000; 
			margin-bottom: 32px;
			transition: color 0.3s;
		}
		p {
			color: #666;
			line-height: 1.6;
			margin-bottom: 0;
			transition: color 0.3s;
		}
		
		@media (prefers-color-scheme: dark) {
			body {
				background: #191e24;
				color: #e5e5e5;
			}
			.container {
				background: #1d232a;
				border: 1px solid #15191e;
				box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
			}
			.logo {
				color: #fff;
			}
			p {
				color: #a3a3a3;
			}
			.error {
				color: #f87171;
			}
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="logo">Orange Juice OJ</div>
		<div class="error">❌ Invalid or Expired Reset Link</div>
		<p>The reset token is invalid or has expired. Please request a new password reset.</p>
	</div>
</body>
</html>`
}

func PasswordResetPage() string {
	return `
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Reset Password - Orange Juice OJ</title>
	<style>
		* { box-sizing: border-box; margin: 0; padding: 0; }
		body { 
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
			margin: 0; 
			padding: 20px; 
			background: #fafafa;
			min-height: 100vh; 
			display: flex; 
			align-items: center; 
			justify-content: center;
			color: #15191e;
			transition: background-color 0.3s, color 0.3s;
		}
		.container { 
			background: white; 
			padding: 48px; 
			border-radius: 12px; 
			border: 1px solid #e5e5e5;
			box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
			max-width: 400px; 
			width: 100%;
			transition: background-color 0.3s, border-color 0.3s, box-shadow 0.3s;
		}
		.logo { 
			text-align: center; 
			font-size: 28px; 
			font-weight: 600; 
			color: #000; 
			margin-bottom: 32px;
			transition: color 0.3s;
		}
		h2 {
			text-align: center;
			color: #000;
			margin-bottom: 32px;
			font-size: 24px;
			font-weight: 600;
			transition: color 0.3s;
		}
		.form-group { 
			margin-bottom: 24px; 
		}
		label { 
			display: block; 
			margin-bottom: 8px; 
			color: #374151; 
			font-weight: 500;
			font-size: 14px;
			transition: color 0.3s;
		}
		input[type="password"] { 
			width: 100%; 
			padding: 12px 16px; 
			border: 1px solid #d1d5db; 
			border-radius: 8px; 
			font-size: 16px; 
			transition: all 0.2s; 
			box-sizing: border-box;
			background: #fff;
		}
		input[type="password"]:focus { 
			outline: none; 
			border-color: #000; 
			box-shadow: 0 0 0 3px rgba(0, 0, 0, 0.1);
		}
		.btn { 
			width: 100%; 
			padding: 12px 16px; 
			background: #605dff; 
			color: #edf1fe;
			border: none; 
			border-radius: 8px; 
			font-size: 16px; 
			font-weight: 500; 
			cursor: pointer; 
			transition: all 0.2s; 
			display: flex; 
			align-items: center; 
			justify-content: center;
			min-height: 48px;
		}
		.btn:hover:not(:disabled) { 
			background: #5754e8; 
		}
		.btn:disabled { 
			opacity: 0.6; 
			cursor: not-allowed; 
		}
		.spinner { 
			border: 2px solid transparent; 
			border-top: 2px solid #ffffff; 
			border-radius: 50%; 
			width: 16px; 
			height: 16px; 
			animation: spin 1s linear infinite; 
			margin-right: 8px; 
		}
		@keyframes spin { 
			0% { transform: rotate(0deg); } 
			100% { transform: rotate(360deg); } 
		}
		.message { 
			margin-top: 16px; 
			padding: 12px; 
			border-radius: 8px; 
			text-align: center;
			font-size: 14px;
		}
		.success { 
			background: #dcfce7; 
			color: #166534; 
			border: 1px solid #bbf7d0; 
		}
		.error { 
			background: #fef2f2; 
			color: #dc2626; 
			border: 1px solid #fecaca; 
		}
		.requirements { 
			font-size: 12px; 
			color: #6b7280; 
			margin-top: 4px;
			transition: color 0.3s;
		}
		
		@media (prefers-color-scheme: dark) {
			body {
				background: #191e24;
				color: #e5e5e5;
			}
			.container {
				background: #1d232a;
				border: 1px solid #15191e;
				box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
			}
			.logo, h2 {
				color: #ededed;
			}
			label {
				color: #d1d5db;
			}
			input[type="password"] {
				background: #1f2937;
				border: 1px solid #374151;
				color: #e5e5e5;
			}
			input[type="password"]:focus {
				border-color: #fff;
				box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.1);
			}
			.btn {
				background: #605dff;
				color: #edf1fe;
			}
			.btn:hover:not(:disabled) {
				background: #5754e8;
			}
			.success {
				background: #064e3b;
				color: #a7f3d0;
				border: 1px solid #047857;
			}
			.error {
				background: #7f1d1d;
				color: #fca5a5;
				border: 1px solid #dc2626;
			}
			.requirements {
				color: #9ca3af;
			}
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="logo">🍊 Orange Juice OJ</div>
		<h2>Reset Password</h2>
		
		<form id="resetForm">
			<div class="form-group">
				<label for="newPassword">New Password</label>
				<input type="password" id="newPassword" name="new_password" required minlength="6">
				<div class="requirements">Password must be at least 6 characters long</div>
			</div>
			
			<div class="form-group">
				<label for="confirmPassword">Confirm New Password</label>
				<input type="password" id="confirmPassword" name="confirm_password" required minlength="6">
			</div>
			
			<button type="submit" class="btn" id="submitBtn">
				<span id="btnText">Reset Password</span>
			</button>
		</form>
		
		<div id="message" class="message" style="display: none;"></div>
	</div>

	<script>
		document.getElementById('resetForm').addEventListener('submit', async function(e) {
			e.preventDefault();
			
			const newPassword = document.getElementById('newPassword').value;
			const confirmPassword = document.getElementById('confirmPassword').value;
			const messageDiv = document.getElementById('message');
			const submitBtn = document.getElementById('submitBtn');
			const btnText = document.getElementById('btnText');
			
			// Hide previous messages
			messageDiv.style.display = 'none';
			
			// Validate passwords match
			if (newPassword !== confirmPassword) {
				messageDiv.className = 'message error';
				messageDiv.textContent = 'Password confirmation does not match';
				messageDiv.style.display = 'block';
				return;
			}
			
			// Validate password length
			if (newPassword.length < 6) {
				messageDiv.className = 'message error';
				messageDiv.textContent = 'Password must be at least 6 characters long';
				messageDiv.style.display = 'block';
				return;
			}
			
			// Start loading state
			submitBtn.disabled = true;
			btnText.innerHTML = '<div class="spinner"></div>';
			
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
					messageDiv.style.display = 'block';
					
					// Disable form
					document.getElementById('resetForm').style.display = 'none';
					
					// Start countdown
					let countdown = 3;
					messageDiv.innerHTML = 'Password reset successful!<br>Redirecting to login page in ' + countdown + ' seconds...';
					
					const countdownTimer = setInterval(function() {
						countdown--;
						if (countdown > 0) {
							messageDiv.innerHTML = 'Password reset successful!<br>Redirecting to login page in ' + countdown + ' seconds...';
						} else {
							clearInterval(countdownTimer);
							window.location.href = '` + config.GetFrontendURL() + `';
						}
					}, 1000);
				} else {
					messageDiv.className = 'message error';
					messageDiv.textContent = result.message || 'Password reset failed';
					messageDiv.style.display = 'block';
				}
			} catch (error) {
				messageDiv.className = 'message error';
				messageDiv.textContent = 'Network error, please try again later';
				messageDiv.style.display = 'block';
			} finally {
				// Reset button state
				submitBtn.disabled = false;
				btnText.innerHTML = 'Reset Password';
			}
		});
	</script>
</body>
</html>`
}
