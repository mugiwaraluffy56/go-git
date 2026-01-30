package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourusername/gogit/internal/object"
	"github.com/yourusername/gogit/internal/repository"
)

var (
	logOneline bool
	logCount   int
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show commit logs",
	Long:  `Show the commit history starting from HEAD.`,
	RunE:  runLog,
}

func init() {
	rootCmd.AddCommand(logCmd)
	logCmd.Flags().BoolVar(&logOneline, "oneline", false, "Show each commit on a single line")
	logCmd.Flags().IntVarP(&logCount, "number", "n", 0, "Limit the number of commits to show")
}

func runLog(cmd *cobra.Command, args []string) error {
	repoRoot, err := FindRepoRoot()
	if err != nil {
		return err
	}

	refs := repository.NewRefs(repoRoot)

	// Get HEAD commit
	commitHash, err := refs.ResolveHead()
	if err != nil {
		return fmt.Errorf("failed to resolve HEAD: %w", err)
	}

	if commitHash == "" {
		fmt.Println("No commits yet")
		return nil
	}

	count := 0
	for commitHash != "" {
		if logCount > 0 && count >= logCount {
			break
		}

		obj, err := object.ReadObject(repoRoot, commitHash)
		if err != nil {
			return fmt.Errorf("failed to read commit %s: %w", commitHash, err)
		}

		commit, ok := obj.(*object.Commit)
		if !ok {
			return fmt.Errorf("object %s is not a commit", commitHash)
		}

		if logOneline {
			// Short format
			firstLine := strings.Split(commit.Message, "\n")[0]
			fmt.Printf("\033[33m%s\033[0m %s\n", commitHash[:7], firstLine)
		} else {
			// Full format
			fmt.Printf("\033[33mcommit %s\033[0m\n", commitHash)
			fmt.Printf("Author: %s\n", commit.Author)
			fmt.Printf("Date:   %s\n", commit.AuthorTime.Format("Mon Jan 2 15:04:05 2006 -0700"))
			fmt.Printf("\n    %s\n\n", strings.ReplaceAll(commit.Message, "\n", "\n    "))
		}

		// Move to parent
		commitHash = commit.ParentHash
		count++
	}

	return nil
}
