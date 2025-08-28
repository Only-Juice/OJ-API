package main

import (
	"OJ-API/config"
	"OJ-API/utils"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/uuid"
)

// CloneRepository 執行 git clone 操作
func CloneRepository(GitFullName, GitRepoURL, GitAfterHash, GitUsername, GitToken string, cutoffTime time.Time) (string, error) {
	// 生成唯一的代碼路徑
	codePath := fmt.Sprintf("%s/%s", config.Config("REPO_FOLDER"), GitFullName+"/"+uuid.New().String())

	// 配置 clone 選項
	cloneOptions := &git.CloneOptions{
		URL: GitRepoURL,
		Auth: &http.BasicAuth{
			Username: GitUsername,
			Password: GitToken,
		},
		Progress: nil, // 在生產環境中不輸出進度
	}

	// 執行 clone
	repo, err := git.PlainClone(codePath, false, cloneOptions)
	if err != nil {
		return "", fmt.Errorf("failed to clone repository: %v", err)
	}

	// 如果有指定的 commit hash，則檢查該 commit 是否在時間內
	if GitAfterHash != "" && GitAfterHash != "0000000000000000000000000000000000000000" {
		// 檢查指定的 commit 是否在時間之前
		commit, err := repo.CommitObject(plumbing.NewHash(GitAfterHash))
		if err != nil {
			return "", fmt.Errorf("failed to get commit %s: %v", GitAfterHash, err)
		}

		// 如果指定的 commit 時間晚於 cutoff 時間，返回錯誤
		if commit.Author.When.After(cutoffTime) {
			return "", fmt.Errorf("commit %s (%v) is after cutoff time %v",
				GitAfterHash, commit.Author.When, cutoffTime)
		}

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
		// 沒有指定 hash，找出時間前的最新 commit
		utils.Debugf("No specific hash provided, finding latest commit before cutoff time %v", cutoffTime)

		// 獲取 commit 歷史
		commitIter, err := repo.Log(&git.LogOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to get commit history: %v", err)
		}

		var latestValidCommit *object.Commit
		commitIter.ForEach(func(c *object.Commit) error {
			// 找到第一個（最新的）在 cutoff 時間之前的 commit
			if c.Author.When.Before(cutoffTime) || c.Author.When.Equal(cutoffTime) {
				latestValidCommit = c
				return fmt.Errorf("found") // 用 error 來跳出循環
			}
			return nil
		})

		if latestValidCommit == nil {
			return "", fmt.Errorf("no commits found before cutoff time %v", cutoffTime)
		}

		// 獲取 worktree 並 checkout 到找到的 commit
		worktree, err := repo.Worktree()
		if err != nil {
			return "", fmt.Errorf("failed to get worktree: %v", err)
		}

		// Checkout 到找到的 commit hash
		err = worktree.Checkout(&git.CheckoutOptions{
			Hash: latestValidCommit.Hash,
		})
		if err != nil {
			return "", fmt.Errorf("failed to checkout to %s: %v", latestValidCommit.Hash, err)
		}

		utils.Debugf("Successfully cloned and checked out %s to %s at latest valid commit %s (%v)",
			GitFullName, codePath, latestValidCommit.Hash.String(), latestValidCommit.Author.When)
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
