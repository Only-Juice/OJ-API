package sandbox

import (
	"OJ-API/database"
	"OJ-API/models"
	"OJ-API/utils"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const execTimeoutDuration = time.Second * 60

func (s *Sandbox) WorkerLoop(ctx context.Context) {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			utils.Info("WorkerLoop received cancel signal, stopping...")
			return
		case <-ticker.C:
			s.assignJob(ctx)
		}
	}
}

func (s *Sandbox) assignJob(ctx context.Context) {
	// 檢查系統是否正在關機，如果是則停止分配新任務
	select {
	case <-ctx.Done():
		return
	default:
	}

	for s.AvailableCount() > 0 && !s.IsJobEmpty() {
		// 在每次循環時再次檢查系統狀態
		select {
		case <-ctx.Done():
			return
		default:
		}

		job := s.ReleaseJob()
		boxID, ok := s.Reserve(1 * time.Second)
		if !ok {
			s.ReserveJob(job.Repo, job.CodePath, job.UQR)
			continue
		}
		go s.runShellCommandByRepo(ctx, boxID, job)
	}
}

func (s *Sandbox) runShellCommand(parentCtx context.Context, boxID int, cmd models.QuestionTestScript, codePath []byte, userQuestion models.UserQuestionTable) {
	db := database.DBConn

	// 檢查父 context 是否已經被取消，如果是則不開始新任務
	select {
	case <-parentCtx.Done():
		db.Model(&userQuestion).Updates(models.UserQuestionTable{
			Score:   -2,
			Message: "Job cancelled due to server shutdown",
		})
		s.Release(boxID)
		return
	default:
	}

	db.Model(&userQuestion).Updates(models.UserQuestionTable{
		JudgeTime: time.Now().UTC(),
	})

	boxRoot, _ := CopyCodeToBox(boxID, string(codePath))

	defer s.Release(boxID)

	db.Model(&userQuestion).Updates(models.UserQuestionTable{
		Score:   -1,
		Message: "Judging...",
	})

	// 使用獨立的 context，不會被父 context 取消影響，讓任務完整執行
	ctx, cancel := context.WithTimeout(context.Background(), execTimeoutDuration)
	defer cancel()

	// saving code as file
	compileScript := []byte(cmd.TestScript)
	codeID, err := WriteToTempFile(compileScript, boxID)
	if err != nil {
		db.Model(&userQuestion).Updates(models.UserQuestionTable{
			Score:   -2,
			Message: fmt.Sprintf("Failed to save code as file: %v", err),
		})
		return
	}

	defer os.Remove(shellFilename(codeID, boxID))

	if len(codePath) > 0 {
		// make utils dir at code path
		os.MkdirAll(fmt.Sprintf("%v/%s", string(boxRoot), "utils"), 0755)

		// copy grp_parser to code path using efficient Go file operations
		srcPath := "./sandbox/grp_parser/grp_parser"
		dstPath := fmt.Sprintf("%v/%s/grp_parser", string(boxRoot), "utils")

		if err := copyFile(srcPath, dstPath); err != nil {
			utils.Debug(fmt.Sprintf("Failed to copy grp_parser: %v", err))
			db.Model(&userQuestion).Updates(models.UserQuestionTable{
				Score:   -2,
				Message: fmt.Sprintf("Failed to copy score parser: %v", err),
			})
			return
		}

		s.getJsonfromdb(fmt.Sprintf("%v/%s", string(boxRoot), "utils"), cmd)
	}
	defer os.RemoveAll(string(codePath))

	/*
		Compile the code
	*/

	success, compileOut := s.runCompile(boxID, ctx, shellFilename(codeID, boxID), []byte(boxRoot))

	if !success {
		db.Model(&userQuestion).Updates(map[string]interface{}{
			"score":   0,
			"message": "Compilation Failed:\n" + compileOut,
		})
		return
	}

	/*
		Execute the code
	*/

	LogWithLocation("Start Execute")

	executeScript := append([]byte(cmd.ExecuteScript), []byte("\nrm build -rf")...)
	execodeID, err := WriteToTempFile([]byte(executeScript), boxID)
	if err != nil {
		db.Model(&userQuestion).Updates(models.UserQuestionTable{
			Score:   -2,
			Message: fmt.Sprintf("Failed to save code as file: %v", err),
		})
		return
	}
	defer os.Remove(shellFilename(execodeID, boxID))

	s.runExecute(boxID, ctx, shellFilename(execodeID, boxID), []byte(boxRoot))

	/*

		Part for result.

	*/

	utils.Debug("Compilation and execution finished successfully.")
	utils.Debug("Ready to proceed to the next step or return output.")

	// read score from file
	score, err := os.ReadFile(fmt.Sprintf("%s/score.txt", []byte(boxRoot)))
	if err != nil {
		db.Model(&userQuestion).Updates(models.UserQuestionTable{
			Score:   -2,
			Message: fmt.Sprintf("Failed to read score: %v", err),
		})
		return
	}
	// save score to database
	scoreFloat, err := strconv.ParseFloat(strings.TrimSpace(string(score)), 64)
	if err != nil {
		db.Model(&userQuestion).Updates(models.UserQuestionTable{
			Score:   -2,
			Message: fmt.Sprintf("Failed to convert score to int: %v", err),
		})
		return
	}

	// read message from file
	message, err := os.ReadFile(fmt.Sprintf("%s/message.txt", []byte(boxRoot)))
	if err != nil {
		db.Model(&userQuestion).Updates(models.UserQuestionTable{
			Score:   -2,
			Message: fmt.Sprintf("Failed to read message: %v", err),
		})
		return
	}

	if err := db.Model(&userQuestion).Updates(models.UserQuestionTable{
		Score:   scoreFloat,
		Message: strings.TrimSpace(string(message)),
	}).Error; err != nil {
		db.Model(&userQuestion).Updates(models.UserQuestionTable{
			Score:   -2,
			Message: fmt.Sprintf("Failed to update score: %v", err),
		})
		return
	}

	utils.Debug("Done for judge!")
}

func (s *Sandbox) runShellCommandByRepo(ctx context.Context, boxID int, work *Job) {

	db := database.DBConn
	var cmd models.QuestionTestScript
	if err := db.Joins("Question").
		Where("git_repo_url = ?", work.Repo).Take(&cmd).Error; err != nil {
		db.Model(&work.UQR).Updates(models.UserQuestionTable{
			Score:   -2,
			Message: fmt.Sprintf("Failed to find shell command for %v: %v", work.Repo, err),
		})
		s.Release(boxID)
		return
	}
	s.runShellCommand(ctx, boxID, cmd, work.CodePath, work.UQR)
}

func (s *Sandbox) runCompile(box int, ctx context.Context, shellCommand string, codePath []byte) (bool, string) {
	cmdArgs := []string{
		fmt.Sprintf("--box-id=%v", box),
		"--fsize=5120",
		"--wait",
		"--processes",
		"--open-files=0",
		"--env=PATH",
		"--stderr-to-stdout",
	}

	if len(codePath) > 0 {
		cmdArgs = append(cmdArgs,
			fmt.Sprintf("--chdir=%v", string(codePath)),
			fmt.Sprintf("--dir=%v:rw", string(codePath)),
			fmt.Sprintf("--env=CODE_PATH=%v", string(codePath)))
	}

	scriptFile := shellCommand
	cmdArgs = append(cmdArgs, "--run", "--", "/usr/bin/sh", scriptFile)

	cmd := exec.CommandContext(ctx, "isolate", cmdArgs...)

	out, err := cmd.CombinedOutput()

	if err != nil {
		return false, err.Error() + "\n" + string(out)
	}

	if strings.Contains(string(out), "error:") {
		return false, string(out)
	}

	return true, string(out)
}

func (s *Sandbox) runExecute(box int, ctx context.Context, shellCommand string, codePath []byte) (string, bool) {
	cmdArgs := []string{
		fmt.Sprintf("--box-id=%v", box),
		"--fsize=5120",
		"--wait",
		"--processes=100",
		"--open-files=0",
		"--env=PATH",
	}

	if len(codePath) > 0 {
		cmdArgs = append(cmdArgs,
			fmt.Sprintf("--chdir=%v", string(codePath)),
			fmt.Sprintf("--dir=%v:rw", string(codePath)),
			fmt.Sprintf("--env=CODE_PATH=%v", string(codePath)))
	}

	cmdArgs = append(cmdArgs, "--run", "--", "/usr/bin/sh", shellCommand)

	utils.Debugf("Command: isolate %s", strings.Join(cmdArgs, " "))
	cmd := exec.CommandContext(ctx, "isolate", cmdArgs...)

	out, err := cmd.CombinedOutput()

	if err != nil {
		utils.Errorf("Failed to run command: %v", err)
		return "Execute with Error!", false
	}

	return string(out), true
}

func (s *Sandbox) getJsonfromdb(path string, row models.QuestionTestScript) {
	filename := "score.json"
	filepath := filepath.Join(path, filename)
	var prettyJSON []byte
	var tmp interface{}
	if err := json.Unmarshal(row.ScoreScript, &tmp); err != nil {
		prettyJSON = row.ScoreScript
	} else {
		prettyJSON, err = json.MarshalIndent(tmp, "", "  ")
		if err != nil {
			return
		}
	}

	if err := os.WriteFile(filepath, prettyJSON, 0644); err != nil {
		fmt.Println("WriteFile error:", err)
		return
	}

}
