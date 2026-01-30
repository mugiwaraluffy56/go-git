package object

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/yourusername/gogit/internal/utils"
)

// TreeEntry represents a single entry in a tree object
type TreeEntry struct {
	Mode string // File mode (100644 for file, 100755 for executable, 040000 for directory)
	Name string // File or directory name
	Hash string // SHA-1 hash of the object
}

// Tree represents a Git tree object (directory listing)
type Tree struct {
	Entries []TreeEntry
}

// NewTree creates a new Tree
func NewTree() *Tree {
	return &Tree{Entries: make([]TreeEntry, 0)}
}

// AddEntry adds an entry to the tree
func (t *Tree) AddEntry(mode, name, hash string) {
	t.Entries = append(t.Entries, TreeEntry{Mode: mode, Name: name, Hash: hash})
}

// Type returns the object type
func (t *Tree) Type() Type {
	return TypeTree
}

// Content returns the tree content in Git format
func (t *Tree) Content() []byte {
	// Sort entries by name (Git requires this)
	sorted := make([]TreeEntry, len(t.Entries))
	copy(sorted, t.Entries)
	sort.Slice(sorted, func(i, j int) bool {
		// Directories come before files with same prefix
		return sorted[i].Name < sorted[j].Name
	})

	var buf bytes.Buffer
	for _, entry := range sorted {
		// Format: "<mode> <name>\0<20-byte-sha1>"
		buf.WriteString(entry.Mode)
		buf.WriteByte(' ')
		buf.WriteString(entry.Name)
		buf.WriteByte(0)

		// Convert hex hash to binary
		hashBytes, _ := hex.DecodeString(entry.Hash)
		buf.Write(hashBytes)
	}
	return buf.Bytes()
}

// Hash computes the SHA-1 hash of the tree
func (t *Tree) Hash() string {
	return utils.HashObject(string(TypeTree), t.Content())
}

// ParseTree parses tree content into a Tree object
func ParseTree(content []byte) (*Tree, error) {
	tree := NewTree()
	pos := 0

	for pos < len(content) {
		// Find space after mode
		spaceIdx := bytes.IndexByte(content[pos:], ' ')
		if spaceIdx == -1 {
			return nil, fmt.Errorf("invalid tree entry: no space found")
		}
		mode := string(content[pos : pos+spaceIdx])
		pos += spaceIdx + 1

		// Find null byte after name
		nullIdx := bytes.IndexByte(content[pos:], 0)
		if nullIdx == -1 {
			return nil, fmt.Errorf("invalid tree entry: no null byte found")
		}
		name := string(content[pos : pos+nullIdx])
		pos += nullIdx + 1

		// Read 20-byte hash
		if pos+20 > len(content) {
			return nil, fmt.Errorf("invalid tree entry: truncated hash")
		}
		hash := hex.EncodeToString(content[pos : pos+20])
		pos += 20

		tree.Entries = append(tree.Entries, TreeEntry{
			Mode: mode,
			Name: name,
			Hash: hash,
		})
	}

	return tree, nil
}

// PrettyPrint returns a formatted representation of the tree
func (t *Tree) PrettyPrint() string {
	var sb strings.Builder
	for _, entry := range t.Entries {
		objType := "blob"
		if entry.Mode == "40000" || entry.Mode == "040000" {
			objType = "tree"
		}
		sb.WriteString(fmt.Sprintf("%06s %s %s\t%s\n", entry.Mode, objType, entry.Hash, entry.Name))
	}
	return sb.String()
}

// GetEntryByName finds an entry by name
func (t *Tree) GetEntryByName(name string) *TreeEntry {
	for i := range t.Entries {
		if t.Entries[i].Name == name {
			return &t.Entries[i]
		}
	}
	return nil
}
