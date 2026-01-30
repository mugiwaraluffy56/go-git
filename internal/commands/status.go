package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yourusername/gogit/internal/index"
	"github.com/yourusername/gogit/internal/object"
	"github.com/yourusername/gogit/internal/repository"
	"github.com/yourusername/gogit/internal/utils"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the working tree status",
	Long:  `Display paths that have differences between the index and the current HEAD commit, and paths that have differences between the working tree and the index.`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	repoRoot, err := FindRepoRoot()
	if err != nil {
		return err
	}

	// Get current branch
	refs := repository.NewRefs(repoRoot)
	branch, err := refs.CurrentBranch()
	if err != nil {
		branch = "HEAD (detached)"
	}

	fmt.Printf("On branch %s\n\n", branch)

	// Read index
	idx, err := index.ReadIndex(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}

	// Get HEAD tree (if exists)
	headTree := make(map[string]string) // path -> hash
	headCommitHash, err := refs.ResolveHead()
	if err == nil && headCommitHash != "" {
		obj, err := object.ReadObject(repoRoot, headCommitHash)
		if err == nil {
			if commit, ok := obj.(*object.Commit); ok {
				treeObj, err := object.ReadObject(repoRoot, commit.TreeHash)
				if err == nil {
					if tree, ok := treeObj.(*object.Tree); ok {
						for _, entry := range tree.Entries {
							headTree[entry.Name] = entry.Hash
						}
					}
				}
			}
		}
	}

	// Build index map
	indexMap := make(map[string]string) // path -> hash
	for _, entry := range idx.Entries {
		indexMap[entry.Path] = entry.HashString()
	}

	// Find staged changes (index vs HEAD)
	var stagedNew, stagedModified, stagedDeleted []string
	for path, hash := range indexMap {
		if headHash, exists := headTree[path]; !exists {
			stagedNew = append(stagedNew, path)
		} else if headHash != hash {
			stagedModified = append(stagedModified, path)
		}
	}
	for path := range headTree {
		if _, exists := indexMap[path]; !exists {
			stagedDeleted = append(stagedDeleted, path)
		}
	}

	// Find working tree changes (working dir vs index)
	var notStaged, untracked []string
	worktreeFiles := make(map[string]bool)

	err = filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip .gogit directory
		if info.IsDir() && info.Name() == ".gogit" {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return nil
		}

		worktreeFiles[relPath] = true

		// Check if file is in index
		if indexHash, exists := indexMap[relPath]; exists {
			// Compare with working tree
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			currentHash := utils.HashObject("blob", content)
			if currentHash != indexHash {
				notStaged = append(notStaged, relPath)
			}
		} else {
			untracked = append(untracked, relPath)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk working tree: %w", err)
	}

	// Find deleted files (in index but not in working tree)
	var deletedNotStaged []string
	for path := range indexMap {
		if !worktreeFiles[path] {
			deletedNotStaged = append(deletedNotStaged, path)
		}
	}

	// Print results
	hasStaged := len(stagedNew) > 0 || len(stagedModified) > 0 || len(stagedDeleted) > 0
	hasNotStaged := len(notStaged) > 0 || len(deletedNotStaged) > 0
	hasUntracked := len(untracked) > 0

	if hasStaged {
		fmt.Println("Changes to be committed:")
		fmt.Println("  (use \"gogit restore --staged <file>...\" to unstage)")
		fmt.Println()
		for _, f := range stagedNew {
			fmt.Printf("\t\033[32mnew file:   %s\033[0m\n", f)
		}
		for _, f := range stagedModified {
			fmt.Printf("\t\033[32mmodified:   %s\033[0m\n", f)
		}
		for _, f := range stagedDeleted {
			fmt.Printf("\t\033[32mdeleted:    %s\033[0m\n", f)
		}
		fmt.Println()
	}

	if hasNotStaged {
		fmt.Println("Changes not staged for commit:")
		fmt.Println("  (use \"gogit add <file>...\" to update what will be committed)")
		fmt.Println()
		for _, f := range notStaged {
			fmt.Printf("\t\033[31mmodified:   %s\033[0m\n", f)
		}
		for _, f := range deletedNotStaged {
			fmt.Printf("\t\033[31mdeleted:    %s\033[0m\n", f)
		}
		fmt.Println()
	}

	if hasUntracked {
		fmt.Println("Untracked files:")
		fmt.Println("  (use \"gogit add <file>...\" to include in what will be committed)")
		fmt.Println()
		for _, f := range untracked {
			fmt.Printf("\t\033[31m%s\033[0m\n", f)
		}
		fmt.Println()
	}

	if !hasStaged && !hasNotStaged && !hasUntracked {
		if headCommitHash == "" {
			fmt.Println("No commits yet")
		} else {
			fmt.Println("nothing to commit, working tree clean")
		}
	}

	return nil
}
