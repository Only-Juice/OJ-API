package sandbox

import (
	"OJ-API/config"
	"OJ-API/database"
	"OJ-API/gitclone"
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

type JudgeInfo struct {
	QuestionInfo   models.QuestionTestScript
	MotherCodePath string
	BoxID          int
	CodePath       []byte
	UQR            models.UserQuestionTable
}

func (s *Sandbox) runShellCommand(parentCtx context.Context, judgeinfo JudgeInfo) {
	db := database.DBConn
	userQuestion := judgeinfo.UQR
	boxID := judgeinfo.BoxID
	codePath := judgeinfo.CodePath
	mothercodePath := judgeinfo.MotherCodePath
	cmd := judgeinfo.QuestionInfo

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

	CopyDir(mothercodePath+"/test", string(codePath)+"/test")
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
	compileScript := []byte(cmd.CompileScript)
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
	defer os.RemoveAll(string(mothercodePath))

	/*
		Compile the code
	*/

	compileResult, compileSuccess := s.runCompile(boxID, ctx, shellFilename(codeID, boxID), []byte(boxRoot))

	/*
		Execute the code
	*/

	execodeID, err := WriteToTempFile([]byte(cmd.ExecuteScript), boxID)
	if err != nil {
		db.Model(&userQuestion).Updates(models.UserQuestionTable{
			Score:   -2,
			Message: fmt.Sprintf("Failed to save code as file: %v", err),
		})
		return
	}
	defer os.Remove(shellFilename(execodeID, boxID))

	exeResult, exeSuccess := s.runExecute(boxID, ctx, cmd, shellFilename(execodeID, boxID), []byte(boxRoot))
	/*
	*
	*	Part for calculate score.
	*
	 */

	ScoreScript := cmd.ScoreScript

	scoreScriptID, err := WriteToTempFile([]byte(ScoreScript), boxID)
	if err != nil {
		db.Model(&userQuestion).Updates(models.UserQuestionTable{
			Score:   -2,
			Message: fmt.Sprintf("Failed to save code as file: %v", err),
		})
		return
	}
	defer os.Remove(shellFilename(execodeID, boxID))
	s.runScore(boxID, ctx, shellFilename(scoreScriptID, boxID), []byte(boxRoot))

	/*

		Part for result.

	*/

	utils.Debug("Compilation and execution finished successfully.")
	utils.Debug("Ready to proceed to the next step or return output.")

	// read score from file
	score, err := os.ReadFile(fmt.Sprintf("%s/score.txt", []byte(boxRoot)))

	if err != nil {

		if !compileSuccess {
			db.Model(&userQuestion).Updates(map[string]interface{}{
				"score":   0,
				"message": "Compilation Failed:\n" + compileResult,
			})
			return
		}

		if !exeSuccess {
			db.Model(&userQuestion).Updates(map[string]interface{}{
				"score":   0,
				"message": "Execute failed:\n" + exeResult,
			})
			return
		}

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
	gitURL := config.GetGiteaBaseURL() + "/" + cmd.Question.GitRepoURL
	mothercodepath, err := gitclone.CloneRepository(cmd.Question.GitRepoURL, gitURL, "", "", "")

	if err != nil {
		db.Model(&work.UQR).Updates(models.UserQuestionTable{
			Score:   -2,
			Message: fmt.Sprintf("Can't get test info: %v", err),
		})
		s.Release(boxID)
		return
	}

	judgeinfo := JudgeInfo{
		QuestionInfo:   cmd,
		MotherCodePath: mothercodepath,
		BoxID:          boxID,
		CodePath:       work.CodePath,
		UQR:            work.UQR,
	}
	s.runShellCommand(ctx, judgeinfo)
}

func (s *Sandbox) runCompile(box int, ctx context.Context, shellCommand string, codePath []byte) (string, bool) {

	cmdArgs := []string{
		fmt.Sprintf("--box-id=%v", box),
		"--fsize=10240",
		"--wait",
		"--processes",
		"--open-files=0",
		"--env=PATH",
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
		return err.Error() + "\n" + string(out), false
	}

	return string(out), true
}

func (s *Sandbox) runExecute(box int, ctx context.Context, qt models.QuestionTestScript, shellCommand string, codePath []byte) (string, bool) {
	cmdArgs := []string{
		fmt.Sprintf("--box-id=%v", box),
		fmt.Sprintf("--fsize=%v", qt.FileSize),
		"--wait",
		fmt.Sprintf("--processes=%v", qt.Processes),
		fmt.Sprintf("--open-files=%v", qt.Openfiles),
		"--env=PATH",
		fmt.Sprintf("--time=%.3f", float64(qt.Time)/1000.0),
		fmt.Sprintf("--wall-time=%.3f", float64(qt.WallTime)/1000.0),
		fmt.Sprintf("--mem=%v", qt.Memory),
		fmt.Sprintf("--stack=%v", qt.StackMemory),
	}

	if len(codePath) > 0 {
		cmdArgs = append(cmdArgs,
			fmt.Sprintf("--chdir=%v", string(codePath)),
			fmt.Sprintf("--dir=%v:rw", string(codePath)),
			fmt.Sprintf("--env=CODE_PATH=%v", string(codePath)))
	}

	cmdArgs = append(cmdArgs, "--run", "--", "/usr/bin/bash", shellCommand)

	utils.Debugf("Command: isolate %s", strings.Join(cmdArgs, " "))
	cmd := exec.CommandContext(ctx, "isolate", cmdArgs...)

	out, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Sprintf("%v\n%s", err, string(out)), false
	}

	return string(out), true
}

func (s *Sandbox) runScore(box int, ctx context.Context, shellCommand string, codePath []byte) (string, bool) {
	cmdArgs := []string{
		fmt.Sprintf("--box-id=%v", box),
		"--fsize=10240",
		"--wait",
		"--processes=100",
		"--open-files=65536",
		"--env=PATH",
	}

	if len(codePath) > 0 {
		cmdArgs = append(cmdArgs,
			fmt.Sprintf("--chdir=%v", string(codePath)),
			fmt.Sprintf("--dir=%v:rw", string(codePath)),
			fmt.Sprintf("--env=CODE_PATH=%v", string(codePath)))
	}

	cmdArgs = append(cmdArgs, "--run", "--", "/usr/bin/bash", shellCommand)

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
	if err := json.Unmarshal([]byte(row.ScoreMap), &tmp); err != nil {
		prettyJSON = []byte(row.ScoreMap)
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
