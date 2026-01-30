package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yourusername/gogit/internal/object"
)

var (
	catFilePretty bool
	catFileType   bool
	catFileSize   bool
)

var catFileCmd = &cobra.Command{
	Use:   "cat-file <object>",
	Short: "Provide content, type, or size information for repository objects",
	Long:  `Display information about objects stored in the repository.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCatFile,
}

func init() {
	rootCmd.AddCommand(catFileCmd)
	catFileCmd.Flags().BoolVarP(&catFilePretty, "pretty", "p", false, "Pretty-print the contents of <object>")
	catFileCmd.Flags().BoolVarP(&catFileType, "type", "t", false, "Show the object type")
	catFileCmd.Flags().BoolVarP(&catFileSize, "size", "s", false, "Show the object size")
}

func runCatFile(cmd *cobra.Command, args []string) error {
	hash := args[0]

	repoRoot, err := FindRepoRoot()
	if err != nil {
		return err
	}

	// If only type or size is requested, use GetObjectInfo for efficiency
	if catFileType || catFileSize {
		objType, size, err := object.GetObjectInfo(repoRoot, hash)
		if err != nil {
			return fmt.Errorf("failed to get object info: %w", err)
		}

		if catFileType {
			fmt.Println(objType)
		}
		if catFileSize {
			fmt.Println(size)
		}
		return nil
	}

	// Read and parse the full object
	obj, err := object.ReadObject(repoRoot, hash)
	if err != nil {
		return fmt.Errorf("failed to read object: %w", err)
	}

	if catFilePretty {
		switch o := obj.(type) {
		case *object.Blob:
			fmt.Print(string(o.Content()))
		case *object.Tree:
			fmt.Print(o.PrettyPrint())
		case *object.Commit:
			fmt.Print(o.PrettyPrint())
		default:
			fmt.Print(string(obj.Content()))
		}
	} else {
		// Default: print raw content
		fmt.Print(string(obj.Content()))
	}

	return nil
}
