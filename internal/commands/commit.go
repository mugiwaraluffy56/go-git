package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yourusername/gogit/internal/index"
	"github.com/yourusername/gogit/internal/object"
	"github.com/yourusername/gogit/internal/repository"
)

var (
	commitMessage string
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Record changes to the repository",
	Long:  `Create a new commit containing the current contents of the index.`,
	RunE:  runCommit,
}

func init() {
	rootCmd.AddCommand(commitCmd)
	commitCmd.Flags().StringVarP(&commitMessage, "message", "m", "", "Commit message")
	commitCmd.MarkFlagRequired("message")
}

func runCommit(cmd *cobra.Command, args []string) error {
	repoRoot, err := FindRepoRoot()
	if err != nil {
		return err
	}

	// Open repository
	repo, err := repository.Open(repoRoot)
	if err != nil {
		return err
	}

	// Read index
	idx, err := index.ReadIndex(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}

	if len(idx.Entries) == 0 {
		return fmt.Errorf("nothing to commit (create/add some files and use \"gogit add\")")
	}

	// Build tree from index
	treeHash, err := repo.BuildTreeRecursive(idx)
	if err != nil {
		return fmt.Errorf("failed to build tree: %w", err)
	}

	// Get parent commit (if exists)
	parentHash, _ := repo.Refs.ResolveHead()

	// Get author info
	author, err := repo.GetUserInfo()
	if err != nil {
		author = "Unknown <unknown@unknown>"
	}

	// Create commit object
	commit := object.NewCommit(treeHash, parentHash, author, commitMessage)

	// Write commit
	commitHash, err := object.WriteObject(repoRoot, commit)
	if err != nil {
		return fmt.Errorf("failed to write commit: %w", err)
	}

	// Update HEAD
	if err := repo.Refs.UpdateHead(commitHash); err != nil {
		return fmt.Errorf("failed to update HEAD: %w", err)
	}

	// Print result
	branch, _ := repo.Refs.CurrentBranch()
	if parentHash == "" {
		fmt.Printf("[%s (root-commit) %s] %s\n", branch, commitHash[:7], commitMessage)
	} else {
		fmt.Printf("[%s %s] %s\n", branch, commitHash[:7], commitMessage)
	}

	// Show summary
	fmt.Printf(" %d file(s) changed\n", len(idx.Entries))

	return nil
}
