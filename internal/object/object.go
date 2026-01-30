package object

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/yourusername/gogit/internal/utils"
)

// Type represents the type of a Git object
type Type string

const (
	TypeBlob   Type = "blob"
	TypeTree   Type = "tree"
	TypeCommit Type = "commit"
	TypeTag    Type = "tag"
)

// Object represents a Git object
type Object interface {
	Type() Type
	Content() []byte
	Hash() string
}

// ParseObject parses a raw object (after decompression)
func ParseObject(data []byte) (Object, error) {
	// Find the null byte separating header from content
	nullIdx := bytes.IndexByte(data, 0)
	if nullIdx == -1 {
		return nil, fmt.Errorf("invalid object: no null byte found")
	}

	header := string(data[:nullIdx])
	content := data[nullIdx+1:]

	// Parse header: "<type> <size>"
	spaceIdx := bytes.IndexByte([]byte(header), ' ')
	if spaceIdx == -1 {
		return nil, fmt.Errorf("invalid object header: %s", header)
	}

	objType := Type(header[:spaceIdx])
	sizeStr := header[spaceIdx+1:]
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid object size: %s", sizeStr)
	}

	if len(content) != size {
		return nil, fmt.Errorf("object size mismatch: expected %d, got %d", size, len(content))
	}

	switch objType {
	case TypeBlob:
		return &Blob{content: content}, nil
	case TypeTree:
		return ParseTree(content)
	case TypeCommit:
		return ParseCommit(content)
	default:
		return nil, fmt.Errorf("unknown object type: %s", objType)
	}
}

// ReadObject reads an object from the repository
func ReadObject(repoPath, hash string) (Object, error) {
	if len(hash) < 4 {
		return nil, fmt.Errorf("hash too short: %s", hash)
	}

	objPath := filepath.Join(repoPath, ".gogit", "objects", hash[:2], hash[2:])

	compressed, err := os.ReadFile(objPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read object %s: %w", hash, err)
	}

	data, err := utils.Decompress(compressed)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress object %s: %w", hash, err)
	}

	return ParseObject(data)
}

// WriteObject writes an object to the repository
func WriteObject(repoPath string, obj Object) (string, error) {
	content := obj.Content()
	header := fmt.Sprintf("%s %d\x00", obj.Type(), len(content))
	store := append([]byte(header), content...)

	hash := utils.HashBytes(store)

	dir := filepath.Join(repoPath, ".gogit", "objects", hash[:2])
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create object directory: %w", err)
	}

	objPath := filepath.Join(dir, hash[2:])

	// Check if object already exists
	if _, err := os.Stat(objPath); err == nil {
		return hash, nil
	}

	compressed, err := utils.Compress(store)
	if err != nil {
		return "", fmt.Errorf("failed to compress object: %w", err)
	}

	// Write to temp file first, then rename (atomic)
	tmpPath := objPath + ".tmp"
	if err := os.WriteFile(tmpPath, compressed, 0444); err != nil {
		return "", fmt.Errorf("failed to write object: %w", err)
	}

	if err := os.Rename(tmpPath, objPath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to rename object: %w", err)
	}

	return hash, nil
}

// GetObjectInfo returns type and size without fully parsing
func GetObjectInfo(repoPath, hash string) (Type, int, error) {
	objPath := filepath.Join(repoPath, ".gogit", "objects", hash[:2], hash[2:])

	compressed, err := os.ReadFile(objPath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read object %s: %w", hash, err)
	}

	data, err := utils.Decompress(compressed)
	if err != nil {
		return "", 0, fmt.Errorf("failed to decompress object %s: %w", hash, err)
	}

	nullIdx := bytes.IndexByte(data, 0)
	if nullIdx == -1 {
		return "", 0, fmt.Errorf("invalid object: no null byte found")
	}

	header := string(data[:nullIdx])
	spaceIdx := bytes.IndexByte([]byte(header), ' ')
	if spaceIdx == -1 {
		return "", 0, fmt.Errorf("invalid object header: %s", header)
	}

	objType := Type(header[:spaceIdx])
	sizeStr := header[spaceIdx+1:]
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid object size: %s", sizeStr)
	}

	return objType, size, nil
}
