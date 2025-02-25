package sandbox

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"OJ-API/database"
	"OJ-API/models"
)

const execTimeoutDuration = time.Second * 60

// SandboxPtr is a pointer to Sandbox
var SandboxPtr *Sandbox

func (s *Sandbox) RunShellCommand(shellCommand []byte, codePath []byte) string {
	ctx, cancel := context.WithTimeout(context.Background(), execTimeoutDuration)
	defer cancel()

	boxID := s.Reserve()
	defer s.Release(boxID)

	// saving code as file
	shellCommand = append(shellCommand, []byte("\nrm build -rf")...)
	codeID, err := WriteToTempFile(shellCommand)
	if err != nil {
		log.Println("error saving code as file:", err)
		return "Failed to save code as file"
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
			log.Printf("Failed to copy python code: %v", err)
			return fmt.Sprintf("Failed to copy python code: %v", err)
		}
	}
	
	

	cmdArgs = append(cmdArgs, "--run", "--", "/usr/bin/sh", shellFilename(codeID))
	

	cmd := exec.CommandContext(ctx, "isolate", cmdArgs...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to run command: %v", err)
		return fmt.Sprintf("Failed to run command: %v", err)
	}

	log.Printf("Command output: %s", string(out))
	return string(out)
}

func (s *Sandbox) RunShellCommandByRepo(parentsRepo string, codePath []byte) string {
	db := database.DBConn

	var cmd models.Sandbox
	if err := db.Where("source_git_repo = ?", parentsRepo).First(&cmd).Error; err != nil {
		return fmt.Sprintf("Failed to find shell command for %v", parentsRepo)
	}

	return s.RunShellCommand([]byte(cmd.Script), codePath)
}