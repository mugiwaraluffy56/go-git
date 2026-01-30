package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yourusername/gogit/internal/repository"
)

var (
	branchDelete bool
)

var branchCmd = &cobra.Command{
	Use:   "branch [name]",
	Short: "List, create, or delete branches",
	Long:  `Without arguments, list all branches. With a name, create a new branch.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runBranch,
}

func init() {
	rootCmd.AddCommand(branchCmd)
	branchCmd.Flags().BoolVarP(&branchDelete, "delete", "d", false, "Delete a branch")
}

func runBranch(cmd *cobra.Command, args []string) error {
	repoRoot, err := FindRepoRoot()
	if err != nil {
		return err
	}

	refs := repository.NewRefs(repoRoot)

	// Delete branch
	if branchDelete {
		if len(args) == 0 {
			return fmt.Errorf("branch name required for deletion")
		}
		if err := refs.DeleteBranch(args[0]); err != nil {
			return err
		}
		fmt.Printf("Deleted branch %s\n", args[0])
		return nil
	}

	// Create branch
	if len(args) > 0 {
		branchName := args[0]

		// Get current HEAD commit
		commitHash, err := refs.ResolveHead()
		if err != nil {
			return fmt.Errorf("failed to resolve HEAD: %w", err)
		}
		if commitHash == "" {
			return fmt.Errorf("cannot create branch: no commits yet")
		}

		if err := refs.CreateBranch(branchName, commitHash); err != nil {
			return err
		}

		fmt.Printf("Created branch '%s' at %s\n", branchName, commitHash[:7])
		return nil
	}

	// List branches
	branches, err := refs.ListBranches()
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	currentBranch, _ := refs.CurrentBranch()

	if len(branches) == 0 {
		fmt.Println("No branches yet (make a commit first)")
		return nil
	}

	for _, branch := range branches {
		if branch == currentBranch {
			fmt.Printf("* \033[32m%s\033[0m\n", branch)
		} else {
			fmt.Printf("  %s\n", branch)
		}
	}

	return nil
}
