package sandbox

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"OJ-API/database"
	"OJ-API/models"
)

const execTimeoutDuration = time.Second * 60

// SandboxPtr is a pointer to Sandbox
var SandboxPtr *Sandbox

func (s *Sandbox) RunShellCommand(shellCommand []byte, codePath []byte, userQuestion models.UserQuestionTable) {
	db := database.DBConn

	db.Model(&userQuestion).Updates(models.UserQuestionTable{
		JudgeTime: time.Now().UTC(),
	})

	boxID := s.Reserve()
	defer s.Release(boxID)

	db.Model(&userQuestion).Updates(models.UserQuestionTable{
		Score:   -1,
		Message: "Judging...",
	})

	ctx, cancel := context.WithTimeout(context.Background(), execTimeoutDuration)
	defer cancel()

	// saving code as file
	shellCommand = append(shellCommand, []byte("\nrm build -rf")...)
	codeID, err := WriteToTempFile(shellCommand)
	if err != nil {
		db.Model(&userQuestion).Updates(models.UserQuestionTable{
			Score:   -2,
			Message: fmt.Sprintf("Failed to save code as file: %v", err),
		})
		return
	}
	defer os.Remove(shellFilename(codeID))

	// running the code
	cmdArgs := []string{
		fmt.Sprintf("--box-id=%v", boxID),
		"--fsize=5120",
		fmt.Sprintf("--dir=%v", CodeStorageFolder),
		"--wait",
		"--processes=100",
		"--open-files=0",
		"--env=PATH",
		"--stderr-to-stdout",
	}

	if len(codePath) > 0 {
		cmdArgs = append(cmdArgs,
			fmt.Sprintf("--chdir=%v", string(codePath)),
			fmt.Sprintf("--dir=%v:rw", string(codePath)),
			fmt.Sprintf("--env=CODE_PATH=%v", string(codePath)))

		// copy python code(./sandbox/python/grp_parser.py) to code path
		os.Mkdir(fmt.Sprintf("%v/%s", string(codePath), "utils"), 0755)
		copy := exec.CommandContext(ctx, "cp", "./sandbox/python/grp_parser.py", fmt.Sprintf("%v/%s", string(codePath), "utils"))
		if err := copy.Run(); err != nil {
			db.Model(&userQuestion).Updates(models.UserQuestionTable{
				Score:   -2,
				Message: fmt.Sprintf("Failed to copy python code: %v", err),
			})
			return
		}
	}

	cmdArgs = append(cmdArgs, "--run", "--", "/usr/bin/sh", shellFilename(codeID))

	cmd := exec.CommandContext(ctx, "isolate", cmdArgs...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		db.Model(&userQuestion).Updates(models.UserQuestionTable{
			Score:   -2,
			Message: fmt.Sprintf("Failed to run command: %v", err),
		})
		return
	}

	log.Printf("Command output: %s", string(out))

	// read score from file
	score, err := os.ReadFile(fmt.Sprintf("%s/score.txt", codePath))
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
	message, err := os.ReadFile(fmt.Sprintf("%s/message.txt", codePath))
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
}

func (s *Sandbox) RunShellCommandByRepo(parentsRepo string, codePath []byte, userQuestion models.UserQuestionTable) {
	db := database.DBConn

	var cmd models.QuestionTestScript
	if err := db.Joins("Question").
		Where("git_repo_url = ?", parentsRepo).Take(&cmd).Error; err != nil {
		db.Model(&userQuestion).Updates(models.UserQuestionTable{
			Score:   -2,
			Message: fmt.Sprintf("Failed to find shell command for %v: %v", parentsRepo, err),
		})
		return
	}

	s.RunShellCommand([]byte(cmd.TestScript), codePath, userQuestion)
}
