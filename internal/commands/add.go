package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yourusername/gogit/internal/index"
	"github.com/yourusername/gogit/internal/object"
)

var addCmd = &cobra.Command{
	Use:   "add <file>...",
	Short: "Add file contents to the index",
	Long:  `Add file contents to the index (staging area) for the next commit.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	repoRoot, err := FindRepoRoot()
	if err != nil {
		return err
	}

	// Read existing index
	idx, err := index.ReadIndex(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}

	for _, arg := range args {
		// Handle glob patterns and directories
		matches, err := filepath.Glob(arg)
		if err != nil {
			return fmt.Errorf("invalid pattern %s: %w", arg, err)
		}

		if len(matches) == 0 {
			// Try as a literal path
			matches = []string{arg}
		}

		for _, match := range matches {
			if err := addPath(repoRoot, idx, match); err != nil {
				return fmt.Errorf("failed to add %s: %w", match, err)
			}
		}
	}

	// Write updated index
	if err := idx.Write(repoRoot); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	return nil
}

func addPath(repoRoot string, idx *index.Index, path string) error {
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(repoRoot, path)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("path not found: %s", path)
	}

	if info.IsDir() {
		// Recursively add directory contents
		return filepath.Walk(absPath, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip .gogit directory
			if info.IsDir() && info.Name() == ".gogit" {
				return filepath.SkipDir
			}

			// Skip directories, only add files
			if info.IsDir() {
				return nil
			}

			return addFile(repoRoot, idx, p)
		})
	}

	return addFile(repoRoot, idx, absPath)
}

func addFile(repoRoot string, idx *index.Index, absPath string) error {
	// Read file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Create and write blob
	blob := object.NewBlob(content)
	_, err = object.WriteObject(repoRoot, blob)
	if err != nil {
		return fmt.Errorf("failed to write blob: %w", err)
	}

	// Add to index
	if err := idx.AddFile(repoRoot, absPath); err != nil {
		return fmt.Errorf("failed to add to index: %w", err)
	}

	return nil
}
