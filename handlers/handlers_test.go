package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllHandlersExist(t *testing.T) {
	// Test admin handlers exist
	t.Run("AdminHandlers", func(t *testing.T) {
		assert.NotNil(t, ResetUserPassword, "ResetUserPassword handler should exist")
		assert.NotNil(t, GetAllUserInfo, "GetAllUserInfo handler should exist")
		assert.NotNil(t, GetUserInfo, "GetUserInfo handler should exist")
		assert.NotNil(t, UpdateUserInfo, "UpdateUserInfo handler should exist")
	})

	// Test exam handlers exist
	t.Run("ExamHandlers", func(t *testing.T) {
		assert.NotNil(t, CreateExam, "CreateExam handler should exist")
		assert.NotNil(t, GetExam, "GetExam handler should exist")
		assert.NotNil(t, UpdateExam, "UpdateExam handler should exist")
		assert.NotNil(t, DeleteExam, "DeleteExam handler should exist")
		assert.NotNil(t, ListExams, "ListExams handler should exist")
		assert.NotNil(t, GetExamQuestions, "GetExamQuestions handler should exist")
		assert.NotNil(t, AddQuestionToExam, "AddQuestionToExam handler should exist")
		assert.NotNil(t, RemoveQuestionFromExam, "RemoveQuestionFromExam handler should exist")
		assert.NotNil(t, GetTopExamScore, "GetTopExamScore handler should exist")
		assert.NotNil(t, GetExamLeaderboard, "GetExamLeaderboard handler should exist")
	})

	// Test question handlers exist
	t.Run("QuestionHandlers", func(t *testing.T) {
		assert.NotNil(t, GetQuestionList, "GetQuestionList handler should exist")
		assert.NotNil(t, GetQuestionByID, "GetQuestionByID handler should exist")
		assert.NotNil(t, GetUsersQuestions, "GetUsersQuestions handler should exist")
		assert.NotNil(t, GetUserQuestionByID, "GetUserQuestionByID handler should exist")
		assert.NotNil(t, GetQuestion, "GetQuestion handler should exist")
		assert.NotNil(t, AddQuestion, "AddQuestion handler should exist")
		assert.NotNil(t, PatchQuestion, "PatchQuestion handler should exist")
		assert.NotNil(t, DeleteQuestion, "DeleteQuestion handler should exist")
		assert.NotNil(t, GetQuestionTestScript, "GetQuestionTestScript handler should exist")
	})

	// Test score handlers exist
	t.Run("ScoreHandlers", func(t *testing.T) {
		assert.NotNil(t, GetScoreByRepo, "GetScoreByRepo handler should exist")
		assert.NotNil(t, GetAllScore, "GetAllScore handler should exist")
		assert.NotNil(t, GetLeaderboard, "GetLeaderboard handler should exist")
		assert.NotNil(t, GetScoreByQuestionID, "GetScoreByQuestionID handler should exist")
		assert.NotNil(t, ReScoreQuestion, "ReScoreQuestion handler should exist")
		assert.NotNil(t, GetTopScore, "GetTopScore handler should exist")
		assert.NotNil(t, ReScoreUserQuestion, "ReScoreUserQuestion handler should exist")
		assert.NotNil(t, GetScoreByUQRID, "GetScoreByUQRID handler should exist")
	})

	// Test user handlers exist
	t.Run("UserHandlers", func(t *testing.T) {
		assert.NotNil(t, GetUser, "GetUser handler should exist")
		assert.NotNil(t, PostUserIsPublic, "PostUserIsPublic handler should exist")
		assert.NotNil(t, ChangeUserPassword, "ChangeUserPassword handler should exist")
	})

	// Test sandbox handlers exist
	t.Run("SandboxHandlers", func(t *testing.T) {
		assert.NotNil(t, PostSandboxCmd, "PostSandboxCmd handler should exist")
		assert.NotNil(t, GetSandboxStatus, "GetSandboxStatus handler should exist")
	})

	// Test gitea handlers exist
	t.Run("GiteaHandlers", func(t *testing.T) {
		assert.NotNil(t, PostGiteaHook, "PostGiteaHook handler should exist")
		assert.NotNil(t, PostBasicAuthenticationGitea, "PostBasicAuthenticationGitea handler should exist")
		assert.NotNil(t, PostCreateQuestionRepositoryGitea, "PostCreateQuestionRepositoryGitea handler should exist")
		assert.NotNil(t, GetUserProfileGitea, "GetUserProfileGitea handler should exist")
		assert.NotNil(t, PostBulkCreateUserGitea, "PostBulkCreateUserGitea handler should exist")
		assert.NotNil(t, PostCreatePublicKeyGitea, "PostCreatePublicKeyGitea handler should exist")
	})
}

func TestResponseHTTPStruct(t *testing.T) {
	// Test ResponseHTTP struct
	response := ResponseHTTP{
		Success: true,
		Message: "Test message",
		Data:    "Test data",
	}

	assert.True(t, response.Success)
	assert.Equal(t, "Test message", response.Message)
	assert.Equal(t, "Test data", response.Data)
}
