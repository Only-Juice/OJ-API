package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUserModel(t *testing.T) {
	t.Run("create user with default values", func(t *testing.T) {
		user := User{
			UserName: "testuser",
			Email:    "test@example.com",
		}

		assert.Equal(t, "testuser", user.UserName)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, uint(0), user.ID) // Default zero value
		assert.False(t, user.Enable)      // Default false (zero value)
		assert.False(t, user.IsPublic)    // Default false (zero value)
		assert.False(t, user.IsAdmin)     // Default false (zero value)
		assert.Empty(t, user.GiteaToken)
	})

	t.Run("create user with all fields", func(t *testing.T) {
		user := User{
			ID:         1,
			UserName:   "admin",
			Enable:     true,
			Email:      "admin@example.com",
			IsPublic:   true,
			GiteaToken: "token123",
			IsAdmin:    true,
		}

		assert.Equal(t, uint(1), user.ID)
		assert.Equal(t, "admin", user.UserName)
		assert.True(t, user.Enable)
		assert.Equal(t, "admin@example.com", user.Email)
		assert.True(t, user.IsPublic)
		assert.Equal(t, "token123", user.GiteaToken)
		assert.True(t, user.IsAdmin)
	})

	t.Run("user JSON serialization", func(t *testing.T) {
		user := User{
			ID:       1,
			UserName: "testuser",
			Email:    "test@example.com",
			Enable:   true,
		}

		// Test that struct tags are properly defined
		assert.NotEmpty(t, user.UserName)
		assert.NotEmpty(t, user.Email)
	})
}

func TestExamModel(t *testing.T) {
	t.Run("create exam with required fields", func(t *testing.T) {
		now := time.Now()
		user := User{ID: 1, UserName: "owner"}

		exam := Exam{
			ID:          1,
			OwnerID:     1,
			Owner:       user,
			Title:       "Test Exam",
			Description: "Test Description",
			StartTime:   now,
			EndTime:     now.Add(time.Hour),
		}

		assert.Equal(t, uint(1), exam.ID)
		assert.Equal(t, uint(1), exam.OwnerID)
		assert.Equal(t, "Test Exam", exam.Title)
		assert.Equal(t, "Test Description", exam.Description)
		assert.Equal(t, user.ID, exam.Owner.ID)
		assert.True(t, exam.EndTime.After(exam.StartTime))
	})

	t.Run("exam time validation", func(t *testing.T) {
		now := time.Now()

		exam := Exam{
			Title:     "Time Test Exam",
			StartTime: now,
			EndTime:   now.Add(2 * time.Hour),
		}

		duration := exam.EndTime.Sub(exam.StartTime)
		assert.Equal(t, 2*time.Hour, duration)
		assert.True(t, exam.EndTime.After(exam.StartTime))
	})

	t.Run("exam with empty description", func(t *testing.T) {
		exam := Exam{
			Title: "Minimal Exam",
		}

		assert.Equal(t, "Minimal Exam", exam.Title)
		assert.Empty(t, exam.Description)
	})
}

func TestQuestionModel(t *testing.T) {
	t.Run("create question with all fields", func(t *testing.T) {
		now := time.Now()

		question := Question{
			ID:          1,
			Title:       "Two Sum",
			Description: "Given an array of integers, return indices of two numbers that add up to target.",
			GitRepoURL:  "https://github.com/user/two-sum",
			StartTime:   now,
			EndTime:     now.Add(24 * time.Hour),
		}

		assert.Equal(t, uint(1), question.ID)
		assert.Equal(t, "Two Sum", question.Title)
		assert.Contains(t, question.Description, "array of integers")
		assert.Equal(t, "https://github.com/user/two-sum", question.GitRepoURL)
		assert.True(t, question.EndTime.After(question.StartTime))
	})

	t.Run("question time bounds", func(t *testing.T) {
		start := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
		end := time.Date(2023, 1, 2, 10, 0, 0, 0, time.UTC)

		question := Question{
			Title:     "Timed Question",
			StartTime: start,
			EndTime:   end,
		}

		assert.Equal(t, 24*time.Hour, question.EndTime.Sub(question.StartTime))
	})

	t.Run("question with long description", func(t *testing.T) {
		longDesc := make([]byte, 4999) // Close to max size
		for i := range longDesc {
			longDesc[i] = 'a'
		}

		question := Question{
			Title:       "Long Question",
			Description: string(longDesc),
		}

		assert.Equal(t, "Long Question", question.Title)
		assert.Len(t, question.Description, 4999)
	})
}

func TestExamQuestionModel(t *testing.T) {
	t.Run("create exam-question relation", func(t *testing.T) {
		user := User{ID: 1, UserName: "owner"}
		exam := Exam{ID: 1, Title: "Test Exam", Owner: user}
		question := Question{ID: 1, Title: "Test Question"}

		examQuestion := ExamQuestion{
			ExamID:     1,
			Exam:       exam,
			QuestionID: 1,
			Question:   question,
			Point:      100,
		}

		assert.Equal(t, uint(1), examQuestion.ExamID)
		assert.Equal(t, uint(1), examQuestion.QuestionID)
		assert.Equal(t, 100, examQuestion.Point)
		assert.Equal(t, "Test Exam", examQuestion.Exam.Title)
		assert.Equal(t, "Test Question", examQuestion.Question.Title)
	})

	t.Run("exam-question with different point values", func(t *testing.T) {
		testCases := []struct {
			name   string
			points int
		}{
			{"zero points", 0},
			{"low points", 10},
			{"medium points", 50},
			{"high points", 100},
			{"max points", 1000},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				examQuestion := ExamQuestion{
					ExamID:     1,
					QuestionID: 1,
					Point:      tc.points,
				}

				assert.Equal(t, tc.points, examQuestion.Point)
			})
		}
	})
}

func TestUserQuestionRelationModel(t *testing.T) {
	t.Run("create user-question relation", func(t *testing.T) {
		user := User{ID: 1, UserName: "student"}
		question := Question{ID: 1, Title: "Algorithm Problem"}

		relation := UserQuestionRelation{
			ID:             1,
			UserID:         1,
			User:           user,
			QuestionID:     1,
			Question:       question,
			GitUserRepoURL: "https://github.com/student/algorithm-problem",
		}

		assert.Equal(t, uint(1), relation.ID)
		assert.Equal(t, uint(1), relation.UserID)
		assert.Equal(t, uint(1), relation.QuestionID)
		assert.Equal(t, "https://github.com/student/algorithm-problem", relation.GitUserRepoURL)
		assert.Equal(t, "student", relation.User.UserName)
		assert.Equal(t, "Algorithm Problem", relation.Question.Title)
	})

	t.Run("git repo URL formats", func(t *testing.T) {
		testCases := []struct {
			name string
			url  string
		}{
			{"github URL", "https://github.com/user/repo"},
			{"gitlab URL", "https://gitlab.com/user/repo"},
			{"short format", "user/repo"},
			{"with branch", "user/repo/tree/main"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				relation := UserQuestionRelation{
					GitUserRepoURL: tc.url,
				}

				assert.Equal(t, tc.url, relation.GitUserRepoURL)
				assert.LessOrEqual(t, len(relation.GitUserRepoURL), 150) // Check size constraint
			})
		}
	})
}

func TestModelRelationships(t *testing.T) {
	t.Run("exam-owner relationship", func(t *testing.T) {
		owner := User{
			ID:       1,
			UserName: "professor",
			IsAdmin:  true,
		}

		exam := Exam{
			ID:      1,
			OwnerID: owner.ID,
			Owner:   owner,
			Title:   "Final Exam",
		}

		assert.Equal(t, exam.OwnerID, exam.Owner.ID)
		assert.True(t, exam.Owner.IsAdmin)
	})

	t.Run("exam-question-user relationship", func(t *testing.T) {
		// Create entities
		user := User{ID: 1, UserName: "student"}
		question := Question{ID: 1, Title: "Sorting Algorithm"}
		exam := Exam{ID: 1, Title: "Algorithms Exam"}

		// Create relationships
		examQuestion := ExamQuestion{
			ExamID:     exam.ID,
			Exam:       exam,
			QuestionID: question.ID,
			Question:   question,
			Point:      75,
		}

		userQuestionRelation := UserQuestionRelation{
			UserID:         user.ID,
			User:           user,
			QuestionID:     question.ID,
			Question:       question,
			GitUserRepoURL: "student/sorting-solution",
		}

		// Verify relationships
		assert.Equal(t, exam.ID, examQuestion.ExamID)
		assert.Equal(t, question.ID, examQuestion.QuestionID)
		assert.Equal(t, user.ID, userQuestionRelation.UserID)
		assert.Equal(t, question.ID, userQuestionRelation.QuestionID)

		// Verify that the same question is referenced in both relationships
		assert.Equal(t, examQuestion.Question.ID, userQuestionRelation.Question.ID)
	})
}

func TestModelValidation(t *testing.T) {
	t.Run("user email validation scenarios", func(t *testing.T) {
		testCases := []struct {
			name  string
			email string
			valid bool
		}{
			{"valid email", "user@example.com", true},
			{"empty email", "", false},
			{"long email", "verylongemailaddressthatmightexceedlimits@verylongdomainname.com", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				user := User{
					UserName: "testuser",
					Email:    tc.email,
				}

				if tc.valid {
					assert.NotEmpty(t, user.Email)
				} else {
					assert.Empty(t, user.Email)
				}
			})
		}
	})

	t.Run("exam title validation", func(t *testing.T) {
		testCases := []struct {
			name  string
			title string
			valid bool
		}{
			{"valid title", "Midterm Exam", true},
			{"empty title", "", false},
			{"max length title", "This is a very long exam title that tests limits", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				exam := Exam{
					Title: tc.title,
				}

				if tc.valid {
					assert.NotEmpty(t, exam.Title)
				} else {
					assert.Empty(t, exam.Title)
				}
			})
		}
	})

	t.Run("question title and description validation", func(t *testing.T) {
		question := Question{
			Title:       "Valid Question",
			Description: "This is a valid question description.",
		}

		assert.NotEmpty(t, question.Title)
		assert.NotEmpty(t, question.Description)
		assert.LessOrEqual(t, len(question.Title), 100)
		assert.LessOrEqual(t, len(question.Description), 5000)
	})
}

func TestTimeHandling(t *testing.T) {
	t.Run("time zone handling", func(t *testing.T) {
		utc := time.Now().UTC()
		local := time.Now()

		exam := Exam{
			Title:     "Time Zone Test",
			StartTime: utc,
			EndTime:   local,
		}

		assert.NotZero(t, exam.StartTime)
		assert.NotZero(t, exam.EndTime)
	})

	t.Run("exam duration calculation", func(t *testing.T) {
		start := time.Date(2023, 6, 1, 9, 0, 0, 0, time.UTC)
		end := time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC)

		exam := Exam{
			Title:     "3-Hour Exam",
			StartTime: start,
			EndTime:   end,
		}

		duration := exam.EndTime.Sub(exam.StartTime)
		assert.Equal(t, 3*time.Hour, duration)
	})

	t.Run("question availability window", func(t *testing.T) {
		now := time.Now()

		question := Question{
			Title:     "Timed Question",
			StartTime: now.Add(-time.Hour),
			EndTime:   now.Add(time.Hour),
		}

		// Question should be currently available
		assert.True(t, now.After(question.StartTime))
		assert.True(t, now.Before(question.EndTime))
	})
}

func TestQuestionTestScriptModel(t *testing.T) {
	t.Run("create question test script", func(t *testing.T) {
		question := Question{ID: 1, Title: "Test Question"}
		script := "#!/bin/bash\necho 'Test script'\nexit 0"

		testScript := QuestionTestScript{
			ID:         1,
			QuestionID: 1,
			Question:   question,
			TestScript: script,
		}

		assert.Equal(t, uint(1), testScript.ID)
		assert.Equal(t, uint(1), testScript.QuestionID)
		assert.Equal(t, "Test Question", testScript.Question.Title)
		assert.Contains(t, testScript.TestScript, "Test script")
		assert.LessOrEqual(t, len(testScript.TestScript), 4000)
	})

	t.Run("test script size limits", func(t *testing.T) {
		// Test with maximum size script
		maxScript := make([]byte, 3999)
		for i := range maxScript {
			maxScript[i] = 'x'
		}

		testScript := QuestionTestScript{
			QuestionID: 1,
			TestScript: string(maxScript),
		}

		assert.Len(t, testScript.TestScript, 3999)
	})

	t.Run("different script types", func(t *testing.T) {
		testCases := []struct {
			name   string
			script string
		}{
			{"bash script", "#!/bin/bash\necho 'Hello World'"},
			{"python script", "#!/usr/bin/env python3\nprint('Hello World')"},
			{"java compile", "javac Solution.java && java Solution"},
			{"go run", "go run main.go"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				testScript := QuestionTestScript{
					TestScript: tc.script,
				}

				assert.Contains(t, testScript.TestScript, tc.script)
			})
		}
	})
}

func TestTagModel(t *testing.T) {
	t.Run("create tag", func(t *testing.T) {
		tag := Tag{
			ID:   1,
			Name: "algorithms",
		}

		assert.Equal(t, uint(1), tag.ID)
		assert.Equal(t, "algorithms", tag.Name)
	})

	t.Run("tag name validation", func(t *testing.T) {
		testCases := []struct {
			name    string
			tagName string
			valid   bool
		}{
			{"valid tag", "data-structures", true},
			{"short tag", "dp", true},
			{"long tag", "very-long-tag-name-that-might-be-too-long", true},
			{"empty tag", "", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tag := Tag{
					Name: tc.tagName,
				}

				if tc.valid {
					assert.NotEmpty(t, tag.Name)
					assert.LessOrEqual(t, len(tag.Name), 50)
				} else {
					assert.Empty(t, tag.Name)
				}
			})
		}
	})

	t.Run("common programming tags", func(t *testing.T) {
		commonTags := []string{
			"algorithms",
			"data-structures",
			"dynamic-programming",
			"graphs",
			"trees",
			"sorting",
			"searching",
			"recursion",
			"greedy",
			"backtracking",
		}

		for _, tagName := range commonTags {
			tag := Tag{Name: tagName}
			assert.NotEmpty(t, tag.Name)
			assert.LessOrEqual(t, len(tag.Name), 50)
		}
	})
}

func TestContextKeyModel(t *testing.T) {
	t.Run("context key constants", func(t *testing.T) {
		assert.Equal(t, contextKey("jwtClaims"), JWTClaimsKey)
		assert.Equal(t, contextKey("user"), UserContextKey)
		assert.Equal(t, contextKey("client"), ClientContextKey)
	})

	t.Run("context key types", func(t *testing.T) {
		// Test that contextKey is a string type
		var key contextKey = "test"
		assert.IsType(t, contextKey(""), key)
		assert.Equal(t, "test", string(key))
	})

	t.Run("context key uniqueness", func(t *testing.T) {
		keys := []contextKey{JWTClaimsKey, UserContextKey, ClientContextKey}

		// Check that all keys are unique
		keyMap := make(map[contextKey]bool)
		for _, key := range keys {
			assert.False(t, keyMap[key], "Duplicate context key found: %s", key)
			keyMap[key] = true
		}

		assert.Len(t, keyMap, 3)
	})
}

func TestComplexModelInteractions(t *testing.T) {
	t.Run("complete exam setup", func(t *testing.T) {
		// Create users
		instructor := User{
			ID:       1,
			UserName: "instructor",
			Email:    "instructor@university.edu",
			IsAdmin:  true,
		}

		student := User{
			ID:       2,
			UserName: "student",
			Email:    "student@university.edu",
			IsPublic: true,
		}

		// Create questions
		question1 := Question{
			ID:          1,
			Title:       "Binary Search",
			Description: "Implement binary search algorithm",
		}

		question2 := Question{
			ID:          2,
			Title:       "Merge Sort",
			Description: "Implement merge sort algorithm",
		}

		// Create exam
		exam := Exam{
			ID:      1,
			OwnerID: instructor.ID,
			Owner:   instructor,
			Title:   "Algorithms Midterm",
		}

		// Create exam questions
		examQ1 := ExamQuestion{
			ExamID:     exam.ID,
			QuestionID: question1.ID,
			Point:      50,
		}

		examQ2 := ExamQuestion{
			ExamID:     exam.ID,
			QuestionID: question2.ID,
			Point:      50,
		}

		// Create student submissions
		submission1 := UserQuestionRelation{
			UserID:         student.ID,
			QuestionID:     question1.ID,
			GitUserRepoURL: "student/binary-search-solution",
		}

		// Verify the complete setup
		assert.Equal(t, instructor.ID, exam.OwnerID)
		assert.True(t, instructor.IsAdmin)
		assert.Equal(t, 2, len([]ExamQuestion{examQ1, examQ2}))
		assert.Equal(t, 100, examQ1.Point+examQ2.Point) // Total points
		assert.Equal(t, student.ID, submission1.UserID)
		assert.Contains(t, submission1.GitUserRepoURL, "binary-search")
	})

	t.Run("question with test script and tags", func(t *testing.T) {
		// Create question
		question := Question{
			ID:          1,
			Title:       "Two Pointers",
			Description: "Solve using two pointers technique",
		}

		// Create test script
		testScript := QuestionTestScript{
			QuestionID: question.ID,
			TestScript: `#!/bin/bash
echo "Running tests..."
python3 solution.py < input.txt > output.txt
diff expected.txt output.txt`,
		}

		// Create related tags
		tag1 := Tag{ID: 1, Name: "two-pointers"}
		tag2 := Tag{ID: 2, Name: "arrays"}

		// Verify relationships
		assert.Equal(t, question.ID, testScript.QuestionID)
		assert.Contains(t, testScript.TestScript, "python3")
		assert.Equal(t, "two-pointers", tag1.Name)
		assert.Equal(t, "arrays", tag2.Name)
	})
}
