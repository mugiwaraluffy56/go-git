package object

import (
	"fmt"

	"github.com/yourusername/gogit/internal/utils"
)

// Blob represents a Git blob object (file content)
type Blob struct {
	content []byte
}

// NewBlob creates a new Blob from content
func NewBlob(content []byte) *Blob {
	return &Blob{content: content}
}

// Type returns the object type
func (b *Blob) Type() Type {
	return TypeBlob
}

// Content returns the blob content
func (b *Blob) Content() []byte {
	return b.content
}

// Hash computes the SHA-1 hash of the blob
func (b *Blob) Hash() string {
	return utils.HashObject(string(TypeBlob), b.content)
}

// String returns the blob content as a string
func (b *Blob) String() string {
	return string(b.content)
}

// Size returns the size of the blob content
func (b *Blob) Size() int {
	return len(b.content)
}

// PrettyPrint returns a formatted representation of the blob
func (b *Blob) PrettyPrint() string {
	return fmt.Sprintf("%s", b.content)
}
