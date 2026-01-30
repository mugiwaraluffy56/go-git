package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yourusername/gogit/internal/diff"
	"github.com/yourusername/gogit/internal/index"
	"github.com/yourusername/gogit/internal/object"
	"github.com/yourusername/gogit/internal/utils"
)

var (
	diffCached bool
)

var diffCmd = &cobra.Command{
	Use:   "diff [file]",
	Short: "Show changes between commits, commit and working tree, etc",
	Long:  `Show changes between the working tree and the index or a tree.`,
	RunE:  runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
	diffCmd.Flags().BoolVar(&diffCached, "cached", false, "Show changes staged for commit")
	diffCmd.Flags().BoolVar(&diffCached, "staged", false, "Synonym for --cached")
}

func runDiff(cmd *cobra.Command, args []string) error {
	repoRoot, err := FindRepoRoot()
	if err != nil {
		return err
	}

	// Read index
	idx, err := index.ReadIndex(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}

	// Build index map
	indexMap := make(map[string]*index.Entry)
	for i := range idx.Entries {
		indexMap[idx.Entries[i].Path] = &idx.Entries[i]
	}

	// Get files to diff
	var filesToDiff []string
	if len(args) > 0 {
		filesToDiff = args
	} else {
		// All tracked files
		for path := range indexMap {
			filesToDiff = append(filesToDiff, path)
		}
	}

	hasDiff := false

	for _, relPath := range filesToDiff {
		entry, inIndex := indexMap[relPath]

		absPath := filepath.Join(repoRoot, relPath)
		workingContent, err := os.ReadFile(absPath)
		workingExists := err == nil

		if !inIndex && !workingExists {
			continue
		}

		var oldContent, newContent string
		var oldName, newName string

		if diffCached {
			// Compare index vs HEAD (not implemented here - simplified)
			// For now, just show index content
			if inIndex {
				blobObj, err := object.ReadObject(repoRoot, entry.HashString())
				if err == nil {
					if blob, ok := blobObj.(*object.Blob); ok {
						newContent = string(blob.Content())
						newName = relPath
						oldName = relPath
						oldContent = "" // Would be HEAD content
					}
				}
			}
		} else {
			// Compare working tree vs index
			if inIndex {
				// Get index content
				blobObj, err := object.ReadObject(repoRoot, entry.HashString())
				if err != nil {
					continue
				}
				blob, ok := blobObj.(*object.Blob)
				if !ok {
					continue
				}
				oldContent = string(blob.Content())
				oldName = relPath

				if workingExists {
					newContent = string(workingContent)
					newName = relPath

					// Check if content is the same
					if utils.HashObject("blob", workingContent) == entry.HashString() {
						continue
					}
				} else {
					// File deleted
					newContent = ""
					newName = "/dev/null"
				}
			} else if workingExists {
				// New file (not in index)
				oldContent = ""
				oldName = "/dev/null"
				newContent = string(workingContent)
				newName = relPath
			}
		}

		// Compute diff
		changes := diff.Diff(oldContent, newContent)

		// Only show if there are actual changes
		hasChanges := false
		for _, change := range changes {
			if change.Type != diff.ChangeEqual {
				hasChanges = true
				break
			}
		}

		if hasChanges {
			hasDiff = true
			fmt.Println(diff.Format(oldName, newName, changes))
		}
	}

	if !hasDiff {
		// No output means no differences
	}

	return nil
}
