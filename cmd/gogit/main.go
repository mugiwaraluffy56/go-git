package main

import (
	"os"

	"github.com/yourusername/gogit/internal/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
