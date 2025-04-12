package sandbox

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

const execTimeoutDuration = time.Second * 60

// SandboxPtr is a pointer to Sandbox
var SandboxPtr *Sandbox

// codePath must be absolute path
func (s *Sandbox) RunShellCommand(shellCommand []byte, codePath []byte) {
	boxID := s.Reserve()
	defer s.Release(boxID)

	ctx, cancel := context.WithTimeout(context.Background(), execTimeoutDuration)
	defer cancel()

	// saving code as file
	codeID, err := WriteToTempFile(shellCommand)
	if err != nil {
		log.Printf("Error writing to temp file: %v", err)
		return
	}
	// defer os.Remove(shellFilename(codeID))

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
			log.Printf("Error copying file: %v", err)
			return
		}
	}

	cmdArgs = append(cmdArgs, "--run", "--", "/usr/bin/ls", shellFilename(codeID))

	log.Printf("Command: isolate %s", strings.Join(cmdArgs, " "))
	cmd := exec.CommandContext(ctx, "isolate", cmdArgs...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error running command: %v", err)
		return
	}

	log.Printf("Command output: %s", string(out))

}
