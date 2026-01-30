package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Create an empty GoGit repository",
	Long:  `Initialize a new GoGit repository in the specified directory, or the current directory if not specified.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	gogitDir := filepath.Join(absPath, ".gogit")

	// Check if already initialized
	if _, err := os.Stat(gogitDir); err == nil {
		return fmt.Errorf("already a gogit repository: %s", gogitDir)
	}

	// Create directory structure
	dirs := []string{
		gogitDir,
		filepath.Join(gogitDir, "objects"),
		filepath.Join(gogitDir, "refs", "heads"),
		filepath.Join(gogitDir, "refs", "tags"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create HEAD file pointing to main branch
	headContent := "ref: refs/heads/main\n"
	if err := os.WriteFile(filepath.Join(gogitDir, "HEAD"), []byte(headContent), 0644); err != nil {
		return fmt.Errorf("failed to create HEAD: %w", err)
	}

	// Create config file
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
`
	if err := os.WriteFile(filepath.Join(gogitDir, "config"), []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	// Create description file
	descContent := "Unnamed repository; edit this file to name the repository.\n"
	if err := os.WriteFile(filepath.Join(gogitDir, "description"), []byte(descContent), 0644); err != nil {
		return fmt.Errorf("failed to create description: %w", err)
	}

	fmt.Printf("Initialized empty GoGit repository in %s\n", gogitDir)
	return nil
}
