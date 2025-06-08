package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	// Store original environment to restore later
	originalEnv := make(map[string]string)
	envVars := []string{"TEST_VAR", "DATABASE_URL", "JWT_SECRET", "ENCRYPTION_KEY"}

	// Backup original environment variables
	for _, envVar := range envVars {
		if val, exists := os.LookupEnv(envVar); exists {
			originalEnv[envVar] = val
		}
	}

	// Clean up environment variables after test
	defer func() {
		for _, envVar := range envVars {
			os.Unsetenv(envVar)
		}
		for envVar, val := range originalEnv {
			os.Setenv(envVar, val)
		}
	}()

	t.Run("get existing environment variable", func(t *testing.T) {
		// Set a test environment variable
		testKey := "TEST_VAR"
		testValue := "test_value_123"
		os.Setenv(testKey, testValue)

		result := Config(testKey)
		assert.Equal(t, testValue, result)
	})

	t.Run("get non-existing environment variable", func(t *testing.T) {
		// Ensure the variable doesn't exist
		os.Unsetenv("NON_EXISTING_VAR")

		result := Config("NON_EXISTING_VAR")
		assert.Empty(t, result)
	})

	t.Run("get empty environment variable", func(t *testing.T) {
		testKey := "EMPTY_VAR"
		os.Setenv(testKey, "")

		result := Config(testKey)
		assert.Empty(t, result)
	})

	t.Run("common configuration keys", func(t *testing.T) {
		testCases := []struct {
			key   string
			value string
		}{
			{"DATABASE_URL", "postgres://user:pass@localhost/db"},
			{"JWT_SECRET", "super_secret_jwt_key"},
			{"ENCRYPTION_KEY", "32_char_encryption_key_12345678"},
			{"PORT", "8080"},
			{"GITEA_URL", "https://gitea.example.com"},
			{"GITEA_TOKEN", "gitea_access_token"},
		}

		for _, tc := range testCases {
			t.Run(tc.key, func(t *testing.T) {
				os.Setenv(tc.key, tc.value)
				result := Config(tc.key)
				assert.Equal(t, tc.value, result)
			})
		}
	})

	t.Run("special characters in values", func(t *testing.T) {
		testCases := []struct {
			name  string
			key   string
			value string
		}{
			{"URL with special chars", "TEST_URL", "https://user:p@ss@host:5432/db?ssl=true"},
			{"JSON config", "TEST_JSON", `{"key":"value","number":123}`},
			{"Base64 data", "TEST_BASE64", "SGVsbG8gV29ybGQ="},
			{"Path with spaces", "TEST_PATH", "/path/with spaces/file.txt"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				os.Setenv(tc.key, tc.value)
				result := Config(tc.key)
				assert.Equal(t, tc.value, result)
			})
		}
	})

	t.Run("case sensitivity", func(t *testing.T) {
		os.Setenv("CASE_TEST", "uppercase")

		// Environment variables are case-sensitive
		assert.Equal(t, "uppercase", Config("CASE_TEST"))
		assert.Empty(t, Config("case_test"))
		assert.Empty(t, Config("Case_Test"))
	})

	t.Run("overriding variables", func(t *testing.T) {
		key := "OVERRIDE_TEST"

		// Set initial value
		os.Setenv(key, "initial")
		assert.Equal(t, "initial", Config(key))

		// Override with new value
		os.Setenv(key, "overridden")
		assert.Equal(t, "overridden", Config(key))
	})
}

func TestConfigWithoutEnvFile(t *testing.T) {
	t.Run("config works when .env file doesn't exist", func(t *testing.T) {
		// This test verifies that Config function doesn't crash
		// when .env.local file doesn't exist
		testKey := "NO_ENV_FILE_TEST"
		testValue := "direct_env_var"

		os.Setenv(testKey, testValue)
		defer os.Unsetenv(testKey)

		// Should not panic and should return the environment variable
		result := Config(testKey)
		assert.Equal(t, testValue, result)
	})
}

func TestConfigDatabaseScenarios(t *testing.T) {
	defer func() {
		// Clean up
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
	}()

	t.Run("full database URL", func(t *testing.T) {
		dbURL := "postgres://username:password@localhost:5432/oj_database?sslmode=disable"
		os.Setenv("DATABASE_URL", dbURL)

		result := Config("DATABASE_URL")
		assert.Equal(t, dbURL, result)
		assert.Contains(t, result, "postgres://")
		assert.Contains(t, result, "localhost:5432")
		assert.Contains(t, result, "oj_database")
	})

	t.Run("individual database components", func(t *testing.T) {
		dbConfig := map[string]string{
			"DB_HOST":     "localhost",
			"DB_PORT":     "5432",
			"DB_USER":     "ojuser",
			"DB_PASSWORD": "ojpass",
			"DB_NAME":     "oj_api",
		}

		for key, value := range dbConfig {
			os.Setenv(key, value)
		}

		for key, expectedValue := range dbConfig {
			result := Config(key)
			assert.Equal(t, expectedValue, result)
		}
	})
}

func TestConfigJWTSecurityScenarios(t *testing.T) {
	defer func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("ENCRYPTION_KEY")
		os.Unsetenv("SESSION_SECRET")
	}()

	t.Run("JWT configuration", func(t *testing.T) {
		jwtSecret := "jwt_secret_key_that_should_be_complex_123456789"
		os.Setenv("JWT_SECRET", jwtSecret)

		result := Config("JWT_SECRET")
		assert.Equal(t, jwtSecret, result)
		assert.GreaterOrEqual(t, len(result), 32) // Should be reasonably long
	})

	t.Run("encryption key configuration", func(t *testing.T) {
		encryptionKey := "12345678901234567890123456789012" // 32 chars for AES-256
		os.Setenv("ENCRYPTION_KEY", encryptionKey)

		result := Config("ENCRYPTION_KEY")
		assert.Equal(t, encryptionKey, result)
		assert.Len(t, result, 32) // AES-256 requires 32-byte key
	})

	t.Run("session secret", func(t *testing.T) {
		sessionSecret := "session_secret_for_cookies"
		os.Setenv("SESSION_SECRET", sessionSecret)

		result := Config("SESSION_SECRET")
		assert.Equal(t, sessionSecret, result)
	})
}

func TestConfigGiteaIntegration(t *testing.T) {
	defer func() {
		os.Unsetenv("GITEA_URL")
		os.Unsetenv("GITEA_TOKEN")
		os.Unsetenv("GITEA_USERNAME")
		os.Unsetenv("GITEA_PASSWORD")
	}()

	t.Run("gitea server configuration", func(t *testing.T) {
		giteaConfig := map[string]string{
			"GITEA_URL":      "https://gitea.example.com",
			"GITEA_TOKEN":    "gitea_access_token_12345",
			"GITEA_USERNAME": "admin",
			"GITEA_PASSWORD": "admin_password",
		}

		for key, value := range giteaConfig {
			os.Setenv(key, value)
		}

		for key, expectedValue := range giteaConfig {
			result := Config(key)
			assert.Equal(t, expectedValue, result)
		}

		// Verify URL format
		giteaURL := Config("GITEA_URL")
		assert.True(t,
			assert.ObjectsAreEqual("https://", giteaURL[:8]) ||
				assert.ObjectsAreEqual("http://", giteaURL[:7]),
			"Gitea URL should start with http:// or https://")
	})
}

func TestConfigServerSettings(t *testing.T) {
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("HOST")
		os.Unsetenv("ENV")
		os.Unsetenv("DEBUG")
	}()

	t.Run("server configuration", func(t *testing.T) {
		serverConfig := map[string]string{
			"PORT":  "8080",
			"HOST":  "0.0.0.0",
			"ENV":   "production",
			"DEBUG": "false",
		}

		for key, value := range serverConfig {
			os.Setenv(key, value)
		}

		assert.Equal(t, "8080", Config("PORT"))
		assert.Equal(t, "0.0.0.0", Config("HOST"))
		assert.Equal(t, "production", Config("ENV"))
		assert.Equal(t, "false", Config("DEBUG"))
	})

	t.Run("development vs production settings", func(t *testing.T) {
		testCases := []struct {
			env   string
			debug string
		}{
			{"development", "true"},
			{"production", "false"},
			{"testing", "true"},
		}

		for _, tc := range testCases {
			t.Run(tc.env, func(t *testing.T) {
				os.Setenv("ENV", tc.env)
				os.Setenv("DEBUG", tc.debug)

				assert.Equal(t, tc.env, Config("ENV"))
				assert.Equal(t, tc.debug, Config("DEBUG"))
			})
		}
	})
}

func TestConfigEdgeCases(t *testing.T) {
	t.Run("very long environment variable", func(t *testing.T) {
		longValue := make([]byte, 10000)
		for i := range longValue {
			longValue[i] = 'x'
		}

		key := "LONG_VAR_TEST"
		os.Setenv(key, string(longValue))
		defer os.Unsetenv(key)

		result := Config(key)
		assert.Equal(t, string(longValue), result)
		assert.Len(t, result, 10000)
	})

	t.Run("variable with newlines", func(t *testing.T) {
		key := "MULTILINE_VAR"
		value := "line1\nline2\nline3"
		os.Setenv(key, value)
		defer os.Unsetenv(key)

		result := Config(key)
		assert.Equal(t, value, result)
		assert.Contains(t, result, "\n")
	})

	t.Run("variable with unicode characters", func(t *testing.T) {
		key := "UNICODE_VAR"
		value := "Hello 世界 🌍 émojis"
		os.Setenv(key, value)
		defer os.Unsetenv(key)

		result := Config(key)
		assert.Equal(t, value, result)
	})
}

// Benchmark test for Config function performance
func BenchmarkConfig(b *testing.B) {
	os.Setenv("BENCHMARK_VAR", "benchmark_value")
	defer os.Unsetenv("BENCHMARK_VAR")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Config("BENCHMARK_VAR")
	}
}

func BenchmarkConfigNonExistent(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Config("NON_EXISTENT_VAR")
	}
}
