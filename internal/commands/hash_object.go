package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yourusername/gogit/internal/object"
)

var (
	hashObjectWrite bool
	hashObjectType  string
	hashObjectStdin bool
)

var hashObjectCmd = &cobra.Command{
	Use:   "hash-object [file]",
	Short: "Compute object ID and optionally create a blob from a file",
	Long:  `Compute the SHA-1 hash of a file and optionally write it to the object database.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runHashObject,
}

func init() {
	rootCmd.AddCommand(hashObjectCmd)
	hashObjectCmd.Flags().BoolVarP(&hashObjectWrite, "write", "w", false, "Actually write the object into the object database")
	hashObjectCmd.Flags().StringVarP(&hashObjectType, "type", "t", "blob", "Specify the type of object to be created")
	hashObjectCmd.Flags().BoolVar(&hashObjectStdin, "stdin", false, "Read the object from standard input")
}

func runHashObject(cmd *cobra.Command, args []string) error {
	var data []byte
	var err error

	if hashObjectStdin {
		data, err = os.ReadFile("/dev/stdin")
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
	} else if len(args) > 0 {
		data, err = os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", args[0], err)
		}
	} else {
		return fmt.Errorf("must specify a file or use --stdin")
	}

	blob := object.NewBlob(data)
	hash := blob.Hash()

	if hashObjectWrite {
		repoRoot, err := FindRepoRoot()
		if err != nil {
			return err
		}

		_, err = object.WriteObject(repoRoot, blob)
		if err != nil {
			return fmt.Errorf("failed to write object: %w", err)
		}
	}

	fmt.Println(hash)
	return nil
}
