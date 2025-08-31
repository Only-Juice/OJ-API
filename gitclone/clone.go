package gitclone

import (
	"OJ-API/config"
	"OJ-API/utils"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/uuid"
)

// CloneRepository 執行 git clone 操作
func CloneRepository(GitFullName, GitRepoURL, GitAfterHash, GitUsername, GitToken string) (string, error) {

	utils.Debugf("%s", GitFullName)
	utils.Debugf("%s", GitRepoURL)
	utils.Debugf("%s", GitAfterHash)
	utils.Debugf("%s", GitUsername)
	utils.Debugf("%s", GitToken)

	// 生成唯一的代碼路徑
	codePath := fmt.Sprintf("%s/%s", config.Config("REPO_FOLDER"), GitFullName+"/"+uuid.New().String())

	// 配置 clone 選項
	cloneOptions := &git.CloneOptions{
		URL:      GitRepoURL,
		Progress: nil, // 在生產環境中不輸出進度
	}

	// 如果有帳號密碼才設定 Auth（處理私有 repo）
	if GitUsername != "" && GitToken != "" {
		cloneOptions.Auth = &http.BasicAuth{
			Username: GitUsername,
			Password: GitToken,
		}
	}

	// 執行 clone
	repo, err := git.PlainClone(codePath, false, cloneOptions)
	if err != nil {
		return "", fmt.Errorf("failed to clone repository: %v", err)
	}

	// 如果有指定的 commit hash，則 checkout 到該 commit
	if GitAfterHash != "" && GitAfterHash != "0000000000000000000000000000000000000000" {
		// 獲取 worktree 並 checkout 到指定 commit
		worktree, err := repo.Worktree()
		if err != nil {
			return "", fmt.Errorf("failed to get worktree: %v", err)
		}

		// Checkout 到指定的 commit hash
		err = worktree.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(GitAfterHash),
		})
		if err != nil {
			return "", fmt.Errorf("failed to checkout to %s: %v", GitAfterHash, err)
		}
		utils.Debugf("Successfully cloned and checked out %s to %s at commit %s", GitFullName, codePath, GitAfterHash)
	} else {
		utils.Debugf("Successfully cloned %s to %s (using HEAD)", GitFullName, codePath)
	}

	// 設置目錄權限為 777 (讀寫執行權限)
	err = os.Chmod(codePath, 0777)
	if err != nil {
		utils.Warnf("Warning: failed to set permissions for %s: %v", codePath, err)
	}

	// 遞歸設置所有子目錄和文件的權限
	err = filepath.Walk(codePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return os.Chmod(path, 0777)
		}
		return os.Chmod(path, 0644)
	})
	if err != nil {
		utils.Warnf("Warning: failed to set recursive permissions for %s: %v", codePath, err)
	}

	return codePath, nil
}
