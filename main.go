package main

import (
	"io"
	"os"
	"path/filepath"
	"test/sandbox"

	"github.com/google/uuid"
)

func main() {
	sandbox.SandboxPtr = sandbox.NewSandbox(5)
	defer sandbox.SandboxPtr.Cleanup()
	var codePath = "/sandbox/repo/username/example/" + uuid.New().String() + "/"
	copyDir("example", codePath)
	os.Chmod(codePath, 0777)
	defer os.RemoveAll(codePath)
	sandbox.SandboxPtr.RunShellCommand([]byte("/usr/bin/cat text.txt"), []byte(codePath))
}

// copyDir copies the contents of the source directory to the destination directory.
func copyDir(src string, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		return err
	})
}
