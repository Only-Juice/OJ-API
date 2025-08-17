package utils

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type ExportQuestionScoreResponse struct {
	UserName               string    `json:"user_name"`
	GitUserRepoURL         string    `json:"git_user_repo_url"`
	Score                  float64   `json:"score"`
	EarliestBestSubmitTime time.Time `json:"earliest_best_submit_time"`
}

// ExportQuestionScoreToXLSX generates and sends an XLSX file with question scores
func ExportQuestionScoreToXLSX(c *gin.Context, questionID uint, scores []ExportQuestionScoreResponse) error {
	// Create a new Excel file
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Question Scores"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}

	// Set headers
	headers := []string{"User Name", "Git Repository URL", "Score", "Earliest Best Submit Time"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	// Set data
	for i, score := range scores {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), score.UserName)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), score.GitUserRepoURL)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), score.Score)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), score.EarliestBestSubmitTime.Format("2006-01-02 15:04:05"))
	}

	// Set active sheet
	f.SetActiveSheet(index)

	// Set response headers
	filename := fmt.Sprintf("question_%d_scores_%s.xlsx", questionID, time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Write to response
	return f.Write(c.Writer)
}
