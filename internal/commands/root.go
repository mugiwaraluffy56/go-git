package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gogit",
	Short: "A Git implementation in Go",
	Long: `GoGit is a Git clone built from scratch in Go.
It implements core Git functionality including objects,
trees, commits, branches, and more.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// FindRepoRoot walks up the directory tree to find .gogit
func FindRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		gogitPath := dir + "/.gogit"
		if info, err := os.Stat(gogitPath); err == nil && info.IsDir() {
			return dir, nil
		}

		parent := dir[:len(dir)-len(dir[len(dir)-1:])]
		for len(parent) > 0 && parent[len(parent)-1] != '/' {
			parent = parent[:len(parent)-1]
		}
		if parent == "" || parent == "/" {
			break
		}
		parent = parent[:len(parent)-1] // remove trailing /
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not a gogit repository (or any parent up to mount point)")
}
