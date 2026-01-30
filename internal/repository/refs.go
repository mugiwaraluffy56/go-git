package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Refs manages Git references (branches, tags, HEAD)
type Refs struct {
	repoPath string
}

// NewRefs creates a new Refs manager
func NewRefs(repoPath string) *Refs {
	return &Refs{repoPath: repoPath}
}

// ResolveHead resolves HEAD to a commit hash
func (r *Refs) ResolveHead() (string, error) {
	headPath := filepath.Join(r.repoPath, ".gogit", "HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		return "", fmt.Errorf("failed to read HEAD: %w", err)
	}

	headContent := strings.TrimSpace(string(content))

	// Check if HEAD is a symbolic reference
	if strings.HasPrefix(headContent, "ref: ") {
		refPath := strings.TrimPrefix(headContent, "ref: ")
		return r.ResolveRef(refPath)
	}

	// HEAD is a direct commit hash
	return headContent, nil
}

// ResolveRef resolves a reference to a commit hash
func (r *Refs) ResolveRef(refPath string) (string, error) {
	fullPath := filepath.Join(r.repoPath, ".gogit", refPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // Ref doesn't exist (e.g., new repo)
		}
		return "", fmt.Errorf("failed to read ref %s: %w", refPath, err)
	}

	return strings.TrimSpace(string(content)), nil
}

// UpdateHead updates HEAD to point to a new commit or ref
func (r *Refs) UpdateHead(target string) error {
	headPath := filepath.Join(r.repoPath, ".gogit", "HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		return fmt.Errorf("failed to read HEAD: %w", err)
	}

	headContent := strings.TrimSpace(string(content))

	// If HEAD is a symbolic reference, update the branch
	if strings.HasPrefix(headContent, "ref: ") {
		refPath := strings.TrimPrefix(headContent, "ref: ")
		return r.UpdateRef(refPath, target)
	}

	// Otherwise update HEAD directly (detached HEAD state)
	return os.WriteFile(headPath, []byte(target+"\n"), 0644)
}

// UpdateRef updates a reference to point to a commit
func (r *Refs) UpdateRef(refPath, commitHash string) error {
	fullPath := filepath.Join(r.repoPath, ".gogit", refPath)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create ref directory: %w", err)
	}

	return os.WriteFile(fullPath, []byte(commitHash+"\n"), 0644)
}

// CurrentBranch returns the name of the current branch
func (r *Refs) CurrentBranch() (string, error) {
	headPath := filepath.Join(r.repoPath, ".gogit", "HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		return "", fmt.Errorf("failed to read HEAD: %w", err)
	}

	headContent := strings.TrimSpace(string(content))

	if strings.HasPrefix(headContent, "ref: refs/heads/") {
		return strings.TrimPrefix(headContent, "ref: refs/heads/"), nil
	}

	return "", fmt.Errorf("HEAD is not on a branch")
}

// ListBranches returns all local branches
func (r *Refs) ListBranches() ([]string, error) {
	headsPath := filepath.Join(r.repoPath, ".gogit", "refs", "heads")
	entries, err := os.ReadDir(headsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read branches: %w", err)
	}

	var branches []string
	for _, entry := range entries {
		if !entry.IsDir() {
			branches = append(branches, entry.Name())
		}
	}

	return branches, nil
}

// CreateBranch creates a new branch pointing to a commit
func (r *Refs) CreateBranch(name, commitHash string) error {
	refPath := filepath.Join("refs", "heads", name)
	fullPath := filepath.Join(r.repoPath, ".gogit", refPath)

	// Check if branch already exists
	if _, err := os.Stat(fullPath); err == nil {
		return fmt.Errorf("branch '%s' already exists", name)
	}

	return r.UpdateRef(refPath, commitHash)
}

// DeleteBranch deletes a branch
func (r *Refs) DeleteBranch(name string) error {
	fullPath := filepath.Join(r.repoPath, ".gogit", "refs", "heads", name)

	// Check if it's the current branch
	currentBranch, _ := r.CurrentBranch()
	if currentBranch == name {
		return fmt.Errorf("cannot delete the current branch '%s'", name)
	}

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete branch '%s': %w", name, err)
	}

	return nil
}

// SetHead sets HEAD to point to a branch or commit
func (r *Refs) SetHead(target string, symbolic bool) error {
	headPath := filepath.Join(r.repoPath, ".gogit", "HEAD")

	var content string
	if symbolic {
		content = fmt.Sprintf("ref: refs/heads/%s\n", target)
	} else {
		content = target + "\n"
	}

	return os.WriteFile(headPath, []byte(content), 0644)
}

// GetBranchCommit returns the commit hash for a branch
func (r *Refs) GetBranchCommit(branch string) (string, error) {
	refPath := filepath.Join("refs", "heads", branch)
	return r.ResolveRef(refPath)
}
