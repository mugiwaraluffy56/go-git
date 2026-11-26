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

	files, err := collectTreeFiles(repoRoot, commit.TreeHash, "")
	if err != nil {
		return err
	}
	if err := checkCheckoutChanges(repoRoot, files); err != nil {
		return err
	}

	// Update working directory and index
	idx := index.NewIndex()

	for _, entry := range files {
		// Read blob
		blobObj, err := object.ReadObject(repoRoot, entry.Hash)
		if err != nil {
			return fmt.Errorf("failed to read blob %s: %w", entry.Name, err)
		}

		blob, ok := blobObj.(*object.Blob)
		if !ok {
			return fmt.Errorf("object for %q is not a blob", entry.Name)
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

		// WriteFile applies the requested mode only when creating a file.
		info, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("failed to stat %s: %w", entry.Name, err)
		}
		if (info.Mode()&0111 != 0) != (mode&0111 != 0) {
			if err := os.Chmod(filePath, info.Mode().Perm()&^0111|mode&0111); err != nil {
				return fmt.Errorf("failed to update mode for %s: %w", entry.Name, err)
			}
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

// checkCheckoutChanges checks the entire index because checkout replaces it.
func checkCheckoutChanges(repoRoot string, target []object.TreeEntry) error {
	idx, err := index.ReadIndex(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}

	headHash, err := repository.NewRefs(repoRoot).ResolveHead()
	if err != nil {
		return err
	}
	headFiles := make(map[string]object.TreeEntry)
	if headHash != "" {
		obj, err := object.ReadObject(repoRoot, headHash)
		if err != nil {
			return fmt.Errorf("failed to read HEAD commit: %w", err)
		}
		commit, ok := obj.(*object.Commit)
		if !ok {
			return fmt.Errorf("HEAD does not point to a commit")
		}
		files, err := collectTreeFiles(repoRoot, commit.TreeHash, "")
		if err != nil {
			return err
		}
		for _, file := range files {
			headFiles[file.Name] = file
		}
	}

	tracked := make(map[string]bool, len(idx.Entries))
	for _, entry := range idx.Entries {
		head, exists := headFiles[entry.Path]
		if !exists || head.Hash != entry.HashString() || head.Mode != fmt.Sprintf("%o", entry.Mode) {
			return fmt.Errorf("cannot switch with staged changes to %q", entry.Path)
		}
		tracked[entry.Path] = true
	}
	for path := range headFiles {
		// Older checkouts omitted nested paths from the index. Check their
		// working-tree contents separately if this checkout would write them.
		if filepath.Dir(path) == "." && !tracked[path] {
			return fmt.Errorf("cannot switch with staged deletion of %q", path)
		}
	}

	for _, entry := range idx.Entries {
		path := filepath.Join(repoRoot, entry.Path)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			return fmt.Errorf("cannot switch with local changes to %q", entry.Path)
		}
		if err != nil {
			return fmt.Errorf("failed to inspect %q: %w", entry.Path, err)
		}
		if !info.Mode().IsRegular() || (info.Mode()&0111 != 0) != (entry.Mode&0111 != 0) {
			return fmt.Errorf("cannot switch with local changes to %q", entry.Path)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %q: %w", entry.Path, err)
		}
		if utils.HashObject("blob", content) != entry.HashString() {
			return fmt.Errorf("cannot switch with local changes to %q", entry.Path)
		}
	}

	for _, entry := range target {
		if err := checkCheckoutParents(repoRoot, entry.Name); err != nil {
			return err
		}
		if tracked[entry.Name] {
			continue
		}
		path := filepath.Join(repoRoot, entry.Name)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("failed to inspect %q: %w", entry.Name, err)
		}

		// Recover clean nested files left unindexed by older checkouts.
		if head, exists := headFiles[entry.Name]; exists && info.Mode().IsRegular() &&
			(info.Mode()&0111 != 0) == (head.Mode == "100755") {
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read %q: %w", entry.Name, err)
			}
			if utils.HashObject("blob", content) == head.Hash {
				continue
			}
		}
		return fmt.Errorf("cannot switch: untracked path %q would be overwritten", entry.Name)
	}
	return nil
}

func checkCheckoutParents(repoRoot, path string) error {
	dir := filepath.Dir(path)
	if dir == "." {
		return nil
	}
	if err := checkCheckoutParents(repoRoot, dir); err != nil {
		return err
	}
	info, err := os.Lstat(filepath.Join(repoRoot, dir))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to inspect %q: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("cannot switch: path %q blocks directory for %q", dir, path)
	}
	return nil
}

// collectTreeFiles returns leaf entries with paths relative to the repository root.
func collectTreeFiles(repoRoot, hash, dir string) ([]object.TreeEntry, error) {
	obj, err := object.ReadObject(repoRoot, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to read tree %s: %w", hash, err)
	}
	tree, ok := obj.(*object.Tree)
	if !ok {
		return nil, fmt.Errorf("object %s is not a tree", hash)
	}

	var files []object.TreeEntry
	for _, entry := range tree.Entries {
		if !filepath.IsLocal(entry.Name) || filepath.Base(entry.Name) != entry.Name || entry.Name == "." || entry.Name == ".gogit" {
			return nil, fmt.Errorf("invalid tree entry name %q", entry.Name)
		}
		entry.Name = filepath.Join(dir, entry.Name)
		if entry.Mode == "40000" || entry.Mode == "040000" {
			children, err := collectTreeFiles(repoRoot, entry.Hash, entry.Name)
			if err != nil {
				return nil, err
			}
			files = append(files, children...)
		} else {
			files = append(files, entry)
		}
	}
	return files, nil
}
