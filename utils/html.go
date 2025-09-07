package utils

func MissingOrInvalidTokenPage() string {
	return `
<!DOCTYPE html>
<html lang="zh-TW">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>密碼重置 - 橘評測 OJ</title>
	<style>
		body { font-family: 'Arial', sans-serif; margin: 0; padding: 20px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); min-height: 100vh; display: flex; align-items: center; justify-content: center; }
		.container { background: white; padding: 40px; border-radius: 10px; box-shadow: 0 15px 35px rgba(0,0,0,0.1); max-width: 400px; width: 100%; text-align: center; }
		.error { color: #e74c3c; font-size: 18px; margin-bottom: 20px; }
		.logo { font-size: 24px; font-weight: bold; color: #667eea; margin-bottom: 20px; }
	</style>
</head>
<body>
	<div class="container">
		<div class="logo">橘評測 OJ</div>
		<div class="error">❌ 無效的重置連結</div>
		<p>重置代碼遺失或無效，請重新申請密碼重置。</p>
	</div>
</body>
</html>`
}

func ExpiredOrUsedTokenPage() string {
	return `
<!DOCTYPE html>
<html lang="zh-TW">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>密碼重置 - 橘評測 OJ</title>
	<style>
		body { font-family: 'Arial', sans-serif; margin: 0; padding: 20px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); min-height: 100vh; display: flex; align-items: center; justify-content: center; }
		.container { background: white; padding: 40px; border-radius: 10px; box-shadow: 0 15px 35px rgba(0,0,0,0.1); max-width: 400px; width: 100%; text-align: center; }
		.error { color: #e74c3c; font-size: 18px; margin-bottom: 20px; }
		.logo { font-size: 24px; font-weight: bold; color: #667eea; margin-bottom: 20px; }
	</style>
</head>
<body>
	<div class="container">
		<div class="logo">橘評測 OJ</div>
		<div class="error">❌ 無效或過期的重置連結</div>
		<p>重置代碼無效或已過期，請重新申請密碼重置。</p>
	</div>
</body>
</html>`
}

func PasswordResetPage() string {
	return `
<!DOCTYPE html>
<html lang="zh-TW">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>重設密碼 - 橘評測 OJ</title>
	<style>
		body { font-family: 'Arial', sans-serif; margin: 0; padding: 20px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); min-height: 100vh; display: flex; align-items: center; justify-content: center; }
		.container { background: white; padding: 40px; border-radius: 10px; box-shadow: 0 15px 35px rgba(0,0,0,0.1); max-width: 400px; width: 100%; }
		.logo { text-align: center; font-size: 24px; font-weight: bold; color: #667eea; margin-bottom: 30px; }
		.form-group { margin-bottom: 20px; }
		label { display: block; margin-bottom: 8px; color: #333; font-weight: bold; }
		input[type="password"] { width: 100%; padding: 12px; border: 2px solid #e0e0e0; border-radius: 5px; font-size: 16px; transition: border-color 0.3s; box-sizing: border-box; }
		input[type="password"]:focus { outline: none; border-color: #667eea; }
		.btn { width: 100%; padding: 12px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; border: none; border-radius: 5px; font-size: 16px; font-weight: bold; cursor: pointer; transition: transform 0.2s; display: flex; align-items: center; justify-content: center; }
		.btn:hover:not(:disabled) { transform: translateY(-2px); }
		.btn:disabled { opacity: 0.7; cursor: not-allowed; transform: none; }
		.spinner { border: 2px solid transparent; border-top: 2px solid #ffffff; border-radius: 50%; width: 16px; height: 16px; animation: spin 1s linear infinite; margin-right: 8px; }
		@keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }
		.message { margin-top: 15px; padding: 10px; border-radius: 5px; text-align: center; }
		.success { background: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
		.error { background: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
		.requirements { font-size: 12px; color: #666; margin-top: 5px; }
	</style>
</head>
<body>
	<div class="container">
		<div class="logo">🍊 橘評測 OJ</div>
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
			
			<button type="submit" class="btn" id="submitBtn">
				<span id="btnText">重設密碼</span>
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
			} finally {
				// Reset button state
				submitBtn.disabled = false;
				btnText.innerHTML = '重設密碼';
			}
		});
	</script>
</body>
</html>`
}
