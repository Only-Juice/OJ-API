package sandbox

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
	"path/filepath"
)

const execTimeoutDuration = time.Second * 60

// SandboxPtr is a pointer to Sandbox
var SandboxPtr *Sandbox

// codePath must be absolute path
func (s *Sandbox) RunShellCommand(compileCommand []byte,executeCommand []byte, codePath []byte) string {
	boxID := s.Reserve()
	defer s.Release(boxID)

	ctx, cancel := context.WithTimeout(context.Background(), execTimeoutDuration)
	defer cancel()

	// 取得目前工作目錄
	wd, err := os.Getwd()
	resultDir := filepath.Join(wd, "result")
	if err != nil {
		log.Printf("Failed to get working directory: %v", err)
		return "Internal Error"
	}

	//Compile Code
	compileCommand = append(compileCommand, []byte("\nrm build -rf")...)
	comcodeID, err := WriteToTempFile(compileCommand)
	if err != nil {
		log.Println("error saving code as file:", err)
		return "Failed to save code as file"
	}
	defer os.Remove(shellFilename(comcodeID))

	CompileSuccess,compile_res := s.RunCompile(boxID,ctx,comcodeID,codePath)

	// 存 compile 結果
	compileLogPath := filepath.Join(resultDir, "compile_result.txt")
	err = os.WriteFile(compileLogPath, []byte(compile_res), 0644)
	if err != nil {
		log.Printf("Failed to write compile result: %v", err)
	}

	if !CompileSuccess {
		return "Compile Failed!"
	}

	//Execute Code

	executeCommand = append(executeCommand, []byte("\nrm build -rf")...)
	execodeID, err := WriteToTempFile(executeCommand)
	if err != nil {
		log.Println("error saving code as file:", err)
		return "Failed to save code as file"
	}
	defer os.Remove(shellFilename(execodeID))

	execute_res,execute_output := s.RunExecute(boxID,ctx,execodeID,codePath)
	
	// 存 execute 結果
	executeLogPath := filepath.Join(resultDir,"execute_result.txt")
	err = os.WriteFile(executeLogPath, []byte(execute_res), 0644)
	if err != nil {
		log.Printf("Failed to write execute result: %v", err)
	}
	executeLogPath = filepath.Join(resultDir,"execute_output.txt")
	err = os.WriteFile(executeLogPath, []byte(execute_output), 0644)
	if err != nil {
		log.Printf("Failed to write execute result: %v", err)
	}

	return execute_res
}

func (s *Sandbox) RunCompile(box int,ctx context.Context,shellCommand string, codePath []byte) (bool, string) {
	cmdArgs := []string{
		fmt.Sprintf("--box-id=%v", box),
		"--fsize=5120",
		fmt.Sprintf("--dir=%v", CodeStorageFolder),
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

	cmdArgs = append(cmdArgs, "--run", "--", "/usr/bin/sh", shellFilename(shellCommand))
	cmd := exec.CommandContext(ctx, "isolate", cmdArgs...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return false,"Compile with Error!"
	}

	if strings.Contains(string(out), "error:") {
		return false,string(out)
	}

	return true,string(out)

}
func (s *Sandbox) RunExecute(box int,ctx context.Context,shellCommand string, codePath []byte) (string,string) {
	cmdArgs := []string{
		fmt.Sprintf("--box-id=%v", box),
		"--fsize=5120",
		fmt.Sprintf("--dir=%v", CodeStorageFolder),
		"--wait",
		"--processes=100",
		"--open-files=0",
		"--env=PATH",
		"--stdout=out.txt",
		"--time=1",
		"--wall-time=1.5",
		"--mem=131072",
	}

	if len(codePath) > 0 {
		cmdArgs = append(cmdArgs,
			fmt.Sprintf("--chdir=%v", string(codePath)),
			fmt.Sprintf("--dir=%v:rw", string(codePath)),
			fmt.Sprintf("--env=CODE_PATH=%v", string(codePath)))
	}


	cmdArgs = append(cmdArgs, "--run", "--", "/usr/bin/sh", shellFilename(shellCommand))

	log.Printf("Command: isolate %s", strings.Join(cmdArgs, " "))
	cmd := exec.CommandContext(ctx, "isolate", cmdArgs...)

	out, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("Failed to run command: %v", err)
	}

	boxOutputPath := fmt.Sprintf("/var/local/lib/isolate/%v/box/out.txt", box)

	output, readErr := os.ReadFile(boxOutputPath)
	if readErr != nil {
		log.Printf("Failed to read output: %v", readErr)
	}

	log.Printf("Program Output: %s", string(output))

	return string(out),string(output)
}