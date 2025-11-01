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
	"regexp"
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
	var scoreMap CompileFile
	json.Unmarshal([]byte(cmd.ScoreMap), &scoreMap)

	// 檢查父 context 是否已經被取消，如果是則不開始新任務
	select {
	case <-parentCtx.Done():
		db.Model(&userQuestion).Updates(models.UserQuestionTable{
			Score:   -2,
			Message: NewErrorResult(WAITING_TO_JUDGE, "Judge Done", "Job cancelled due to server shutdown"),
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
		Message: NewErrorResult(JUDGING, "Judge", "Judging..."),
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
			Message: NewErrorResult(SYSTEM_FAILED, "System_Failed", err.Error()),
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
				Message: NewErrorResult(SYSTEM_FAILED, "Failed to copy score parser", err.Error()),
			})
			return
		}

		s.getJsonfromdb(fmt.Sprintf("%v/%s", string(boxRoot), "utils"), cmd)
	}
	defer os.RemoveAll(string(codePath))
	defer os.RemoveAll(string(mothercodePath))

	var SandboxJudgeInfo SandboxResult

	/*
		Compile the code
	*/

	SandboxJudgeInfo.CompileResult = s.runCompile(boxID, ctx, shellFilename(codeID, boxID), []byte(boxRoot), scoreMap)

	/*
		Execute the code
	*/

	execodeID, err := WriteToTempFile([]byte(cmd.ExecuteScript), boxID)
	if err != nil {
		db.Model(&userQuestion).Updates(models.UserQuestionTable{
			Score:   -2,
			Message: NewErrorResult(SYSTEM_FAILED, "Failed to save code as file", err.Error()),
		})
		return
	}

	defer os.Remove(shellFilename(execodeID, boxID))

	SandboxJudgeInfo.ExecuteResult = s.runExecute(boxID, ctx, cmd, shellFilename(execodeID, boxID), []byte(boxRoot), SandboxJudgeInfo.CompileResult)
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
			Message: NewErrorResult(SYSTEM_FAILED, "Failed to save code as file", err.Error()),
		})
		return
	}
	defer os.Remove(shellFilename(execodeID, boxID))

	compileAndExecuteResult := s.mergeCompileAndExecuteResult(SandboxJudgeInfo.CompileResult, SandboxJudgeInfo.ExecuteResult)
	SandboxJudgeInfo.JudgeScoreResult = s.runScore(boxID, ctx, shellFilename(scoreScriptID, boxID), []byte(boxRoot), compileAndExecuteResult)

	/*

		Part for result.

	*/

	utils.Debug("Compilation and execution finished successfully.")
	utils.Debug("Ready to proceed to the next step or return output.")

	totalResult, score, _ := MergeJudgeResults(boxRoot, SandboxJudgeInfo.JudgeScoreResult)

	jsonBytes, err := json.MarshalIndent(totalResult, "", "  ")
	if err != nil {
		utils.Debugf("[runHandler] Failed to marshal totalResult: %v\n", err)
	} else {
		result := string(jsonBytes)
		if err := db.Model(&userQuestion).Updates(models.UserQuestionTable{
			Score:   score,
			Message: strings.TrimSpace(string(result)),
		}).Error; err != nil {
			db.Model(&userQuestion).Updates(models.UserQuestionTable{
				Score:   -2,
				Message: NewErrorResult(SYSTEM_FAILED, "Failed to update score", err.Error()),
			})
			return
		}
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

func (s *Sandbox) runCompile(box int, ctx context.Context, shellCommand string, codePath []byte, compilefile CompileFile) []SandboxJudgeResult {
	var results []SandboxJudgeResult
	for _, task := range compilefile.Task {
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
		cmdArgs = append(cmdArgs, "--run", "--", "/usr/bin/sh", scriptFile, task.Target)

		cmd := exec.CommandContext(ctx, "isolate", cmdArgs...)

		result := SandboxJudgeResult{
			Target: task.Target,
		}

		out, err := cmd.CombinedOutput()
		if err != nil {
			result.Status = string(COMPILE_ERROR)
			result.Result = string(out)
		} else {
			result.Status = "SUCCESS"
			result.Result = string(out)
		}
		results = append(results, result)
	}

	return results
}

func (s *Sandbox) runExecute(box int, ctx context.Context, qt models.QuestionTestScript, shellCommand string, codePath []byte, compileResult []SandboxJudgeResult) []SandboxJudgeResult {
	var results []SandboxJudgeResult
	for _, target := range compileResult {
		if target.Status == "FAILED" {
			result := SandboxJudgeResult{
				Target: target.Target,
				Result: "COMPILE NOT SUCCESS",
				Status: "FAILED",
			}
			results = append(results, result)
			continue
		}
		cmdArgs := []string{
			fmt.Sprintf("--box-id=%v", box),
			fmt.Sprintf("--fsize=%v", qt.FileSize),
			"--wait",
			fmt.Sprintf("--processes=%v", qt.Processes),
			fmt.Sprintf("--open-files=%v", qt.OpenFiles),
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

		cmdArgs = append(cmdArgs, "--run", "--", "/usr/bin/bash", shellCommand, target.Target)

		cmd := exec.CommandContext(ctx, "isolate", cmdArgs...)

		result := SandboxJudgeResult{
			Target: target.Target,
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			outStr := string(out)
			if strings.Contains(outStr, "Exited with error status 1") {
				result.Status = "SUCCESS" // 整體執行成功
				result.Result = "⚠️ GTest 測試未全數通過，請檢查 JSON 結果。"
				results = append(results, result)
				continue
			}

			// ✅ 再檢查是否為 core dumped / Segfault 類型
			hasCore := strings.Contains(outStr, "core dumped") ||
				strings.Contains(outStr, "Illegal instruction") ||
				strings.Contains(outStr, "Segmentation fault") ||
				strings.Contains(outStr, "Aborted")

			hasExit := strings.Contains(outStr, "Exited with error status")

			if hasCore && hasExit {
				// 提取 exit code
				exitLine := regexp.MustCompile(`Exited with error status [0-9]+`).FindString(outStr)
				result.Result = "Illegal instruction (core dumped)\n" + exitLine +
					"\n請檢查程式中是否有使用未初始化指標、陣列越界、或動態記憶體錯誤等行為。"
				result.Status = string(RUNTIME_ERROR)
			} else {
				result.Result = outStr
				result.Status = "FAILED"
			}

		} else {
			result.Status = "SUCCESS"
			result.Result = string(out)
		}
		results = append(results, result)

	}

	return results
}

func (s *Sandbox) runScore(box int, ctx context.Context, shellCommand string, codePath []byte, mergeResult []SandboxJudgeResult) []SandboxScoreResult {
	var results []SandboxScoreResult
	for _, target := range mergeResult {
		if target.Status != "SUCCESS" {
			result := SandboxScoreResult{
				Target: target.Target,
				Result: target.Result,
				Status: target.Status,
				Score:  0.0,
			}
			results = append(results, result)
			continue
		}
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

		cmdArgs = append(cmdArgs, "--run", "--", "/usr/bin/bash", shellCommand, target.Target)

		utils.Debugf("Command: isolate %s", strings.Join(cmdArgs, " "))
		cmd := exec.CommandContext(ctx, "isolate", cmdArgs...)

		out, err := cmd.CombinedOutput()
		result := SandboxScoreResult{
			Target: target.Target,
		}
		if err != nil {
			result.Status = "FAILED"
			result.Result = string(out)

		} else {
			score, _ := s.extractScore(string(out))
			result.Status = "SUCCESS"
			result.Result = string(out)
			result.Score = score
		}
		results = append(results, result)
	}

	return results
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

func (s *Sandbox) mergeCompileAndExecuteResult(
	compileResult []SandboxJudgeResult,
	executeResult []SandboxJudgeResult,
) []SandboxJudgeResult {

	finalResults := make([]SandboxJudgeResult, 0)
	compileMap := make(map[string]SandboxJudgeResult)

	// 建立 compile map
	for _, c := range compileResult {
		compileMap[c.Target] = c
	}

	// 逐一比對 execute 結果
	for _, e := range executeResult {
		comp, ok := compileMap[e.Target]
		if ok && comp.Status == "SUCCESS" && e.Status == "SUCCESS" {
			// compile + execute 成功
			finalResults = append(finalResults, SandboxJudgeResult{
				Target: e.Target,
				Status: "SUCCESS",
				Result: "Compile and Execute success, ready for scoring.",
			})
		} else {
			// 任一失敗
			failMsg := ""
			status := ""

			if !ok {
				failMsg = "Missing compile result."
			} else if comp.Status != "SUCCESS" {
				failMsg = fmt.Sprintf("Compile failed: %s", comp.Result)
				status = comp.Status
			} else if e.Status != "SUCCESS" {
				failMsg = fmt.Sprintf("Execute failed: %s", e.Result)
				status = e.Status
			}

			finalResults = append(finalResults, SandboxJudgeResult{
				Target: e.Target,
				Status: status,
				Result: failMsg,
			})
		}
	}

	return finalResults
}

func (s *Sandbox) extractScore(out string) (float64, bool) {
	re := regexp.MustCompile(`\b\d+(\.\d+)?\b`)
	lines := strings.Split(out, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if re.MatchString(line) && !strings.Contains(line, "OK") {
			numStr := re.FindString(line)
			if score, err := strconv.ParseFloat(numStr, 64); err == nil {
				return score, true
			}
		}
	}
	return 0, false
}
