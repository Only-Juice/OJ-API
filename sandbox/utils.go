package sandbox

import (
	"OJ-API/config"
	"OJ-API/utils"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func WriteToTempFile(b []byte, boxID int) (string, error) {
	boxRoot := fmt.Sprintf("%s/%d/box", config.GetIsolatePath(), boxID)
	CodeStorageFolder := fmt.Sprintf("%s/code", boxRoot)

	// using nano second to avoid filename collision in highly concurrent requests
	id := fmt.Sprintf("%v", time.Now().UnixNano())
	err := os.WriteFile(shellFilename(id, boxID), b, 0777)

	if err != nil && errors.Is(err, os.ErrNotExist) {
		// may be the folder absent. so trying to create it
		err = os.Mkdir(CodeStorageFolder, os.ModePerm)
		if err != nil {
			utils.Error("failed creating folder:", CodeStorageFolder)
			return id, err
		}
		utils.Info("created folder:", CodeStorageFolder)
		// second attempt
		err = os.WriteFile(shellFilename(id, boxID), b, 0777)
	}

	return id, err
}

func shellFilename(timestamp string, boxID int) string {
	boxRoot := fmt.Sprintf("%s/%d/box", config.GetIsolatePath(), boxID)
	CodeStorageFolder := fmt.Sprintf("%s/code", boxRoot)

	return fmt.Sprintf("%v/%v.sh", CodeStorageFolder, timestamp)
}

func LogWithLocation(msg string) {
	// skip=1 表示上一層呼叫者
	if pc, file, line, ok := runtime.Caller(1); ok {
		fn := runtime.FuncForPC(pc)
		utils.Debugf("%s:%d [%s] → %s\n", file, line, fn.Name(), msg)
	} else {
		utils.Debugf("unknown location → %s\n", msg)
	}
}

func CopyCodeToBox(boxID int, codePath string) (string, error) {
	boxRoot := fmt.Sprintf("%s/%d/box", config.GetIsolatePath(), boxID)

	err := filepath.Walk(codePath, func(src string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(codePath, src)
		if err != nil {
			return err
		}

		dst := filepath.Join(boxRoot, relPath)

		if info.IsDir() {
			return os.MkdirAll(dst, info.Mode())
		}

		// For files
		srcFile, err := os.Open(src)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(dst)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return err
		}

		// Set permission same as source
		return os.Chmod(dst, info.Mode())
	})

	if err != nil {
		return "", fmt.Errorf("failed to copy code to box: %v", err)
	}

	return boxRoot, nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Get source file info to preserve permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy file content
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Preserve file permissions
	err = os.Chmod(dst, sourceInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}
