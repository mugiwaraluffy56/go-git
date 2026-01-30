package repository

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yourusername/gogit/internal/index"
	"github.com/yourusername/gogit/internal/object"
)

// Repository represents a GoGit repository
type Repository struct {
	Path string
	Refs *Refs
}

// dirEntry represents a directory entry for tree building
type dirEntry struct {
	isDir   bool
	mode    string
	name    string
	hash    string
	entries map[string]*dirEntry
}

// Open opens an existing repository
func Open(path string) (*Repository, error) {
	gogitPath := filepath.Join(path, ".gogit")
	if _, err := os.Stat(gogitPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("not a gogit repository: %s", path)
	}

	return &Repository{
		Path: path,
		Refs: NewRefs(path),
	}, nil
}

// BuildTree creates a tree object from the current index
func (r *Repository) BuildTree(idx *index.Index) (*object.Tree, error) {
	tree := object.NewTree()

	for _, entry := range idx.Entries {
		// For simplicity, we're creating a flat tree
		// A full implementation would handle nested directories
		mode := fmt.Sprintf("%o", entry.Mode)
		tree.AddEntry(mode, entry.Path, entry.HashString())
	}

	return tree, nil
}

// BuildTreeRecursive creates tree objects for nested directory structure
func (r *Repository) BuildTreeRecursive(idx *index.Index) (string, error) {
	root := &dirEntry{
		isDir:   true,
		entries: make(map[string]*dirEntry),
	}

	// Build directory structure
	for _, entry := range idx.Entries {
		parts := splitPath(entry.Path)
		current := root

		for i, part := range parts {
			if i == len(parts)-1 {
				// File entry
				current.entries[part] = &dirEntry{
					isDir: false,
					mode:  fmt.Sprintf("%o", entry.Mode),
					name:  part,
					hash:  entry.HashString(),
				}
			} else {
				// Directory entry
				if _, exists := current.entries[part]; !exists {
					current.entries[part] = &dirEntry{
						isDir:   true,
						name:    part,
						entries: make(map[string]*dirEntry),
					}
				}
				current = current.entries[part]
			}
		}
	}

	// Build trees bottom-up
	return r.buildTreeFromDir(root)
}

func (r *Repository) buildTreeFromDir(dir *dirEntry) (string, error) {
	tree := object.NewTree()

	for name, entry := range dir.entries {
		if entry.isDir {
			// Recursively build subtree
			hash, err := r.buildTreeFromDir(entry)
			if err != nil {
				return "", err
			}
			tree.AddEntry("40000", name, hash)
		} else {
			tree.AddEntry(entry.mode, name, entry.hash)
		}
	}

	// Write tree and return hash
	hash, err := object.WriteObject(r.Path, tree)
	if err != nil {
		return "", fmt.Errorf("failed to write tree: %w", err)
	}

	return hash, nil
}

func splitPath(path string) []string {
	var parts []string
	for path != "" {
		dir, file := filepath.Split(path)
		if file != "" {
			parts = append([]string{file}, parts...)
		}
		if dir == "" {
			break
		}
		path = filepath.Clean(dir)
		if path == "." {
			break
		}
	}
	return parts
}

// GetConfig returns the repository configuration
func (r *Repository) GetConfig(key string) (string, error) {
	// Simple implementation - in reality would parse config file
	configPath := filepath.Join(r.Path, ".gogit", "config")
	_, err := os.Stat(configPath)
	if err != nil {
		return "", err
	}
	// For now, return empty - full implementation would parse INI
	return "", nil
}

// GetUserInfo returns author/committer info
func (r *Repository) GetUserInfo() (string, error) {
	// Try to get from environment or config
	name := os.Getenv("GIT_AUTHOR_NAME")
	if name == "" {
		name = os.Getenv("USER")
	}
	if name == "" {
		name = "Unknown"
	}

	email := os.Getenv("GIT_AUTHOR_EMAIL")
	if email == "" {
		hostname, _ := os.Hostname()
		email = name + "@" + hostname
	}

	return fmt.Sprintf("%s <%s>", name, email), nil
}
