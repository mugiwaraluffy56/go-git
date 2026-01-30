package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yourusername/gogit/internal/index"
	"github.com/yourusername/gogit/internal/object"
	"github.com/yourusername/gogit/internal/repository"
)

var (
	checkoutCreate bool
)

var checkoutCmd = &cobra.Command{
	Use:   "checkout <branch|commit>",
	Short: "Switch branches or restore working tree files",
	Long:  `Switch to a branch or restore working tree files.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCheckout,
}

func init() {
	rootCmd.AddCommand(checkoutCmd)
	checkoutCmd.Flags().BoolVarP(&checkoutCreate, "branch", "b", false, "Create a new branch and switch to it")
}

func runCheckout(cmd *cobra.Command, args []string) error {
	repoRoot, err := FindRepoRoot()
	if err != nil {
		return err
	}

	target := args[0]
	refs := repository.NewRefs(repoRoot)

	// Create new branch if -b flag
	if checkoutCreate {
		commitHash, err := refs.ResolveHead()
		if err != nil || commitHash == "" {
			return fmt.Errorf("cannot create branch: no commits yet")
		}

		if err := refs.CreateBranch(target, commitHash); err != nil {
			return err
		}

		if err := refs.SetHead(target, true); err != nil {
			return fmt.Errorf("failed to update HEAD: %w", err)
		}

		fmt.Printf("Switched to a new branch '%s'\n", target)
		return nil
	}

	// Check if target is a branch
	branchCommit, err := refs.GetBranchCommit(target)
	if err == nil && branchCommit != "" {
		// It's a branch
		if err := checkoutCommit(repoRoot, branchCommit); err != nil {
			return err
		}

		if err := refs.SetHead(target, true); err != nil {
			return fmt.Errorf("failed to update HEAD: %w", err)
		}

		fmt.Printf("Switched to branch '%s'\n", target)
		return nil
	}

	// Try as a commit hash
	commitHash := target
	if len(commitHash) >= 4 {
		obj, err := object.ReadObject(repoRoot, commitHash)
		if err == nil {
			if _, ok := obj.(*object.Commit); ok {
				if err := checkoutCommit(repoRoot, commitHash); err != nil {
					return err
				}

				if err := refs.SetHead(commitHash, false); err != nil {
					return fmt.Errorf("failed to update HEAD: %w", err)
				}

				fmt.Printf("Note: switching to '%s'.\n\n", commitHash[:7])
				fmt.Println("You are in 'detached HEAD' state.")
				return nil
			}
		}
	}

	return fmt.Errorf("pathspec '%s' did not match any branch or commit", target)
}

func checkoutCommit(repoRoot, commitHash string) error {
	// Read commit
	obj, err := object.ReadObject(repoRoot, commitHash)
	if err != nil {
		return fmt.Errorf("failed to read commit: %w", err)
	}

	commit, ok := obj.(*object.Commit)
	if !ok {
		return fmt.Errorf("object is not a commit")
	}

	// Read tree
	treeObj, err := object.ReadObject(repoRoot, commit.TreeHash)
	if err != nil {
		return fmt.Errorf("failed to read tree: %w", err)
	}

	tree, ok := treeObj.(*object.Tree)
	if !ok {
		return fmt.Errorf("object is not a tree")
	}

	// Update working directory and index
	idx := index.NewIndex()

	for _, entry := range tree.Entries {
		// Read blob
		blobObj, err := object.ReadObject(repoRoot, entry.Hash)
		if err != nil {
			return fmt.Errorf("failed to read blob %s: %w", entry.Name, err)
		}

		blob, ok := blobObj.(*object.Blob)
		if !ok {
			// Might be a subtree - skip for now (simplified implementation)
			continue
		}

		// Write file
		filePath := filepath.Join(repoRoot, entry.Name)

		// Ensure directory exists
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Determine file mode
		mode := os.FileMode(0644)
		if entry.Mode == "100755" {
			mode = 0755
		}

		if err := os.WriteFile(filePath, blob.Content(), mode); err != nil {
			return fmt.Errorf("failed to write file %s: %w", entry.Name, err)
		}

		// Add to index
		if err := idx.AddFile(repoRoot, filePath); err != nil {
			return fmt.Errorf("failed to update index: %w", err)
		}
	}

	// Write index
	if err := idx.Write(repoRoot); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	return nil
}
