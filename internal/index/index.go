package index

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/yourusername/gogit/internal/utils"
)

const (
	IndexSignature = "DIRC"
	IndexVersion   = 2
)

// Entry represents a single entry in the index
type Entry struct {
	CTimeSec  uint32
	CTimeNano uint32
	MTimeSec  uint32
	MTimeNano uint32
	Dev       uint32
	Ino       uint32
	Mode      uint32
	UID       uint32
	GID       uint32
	Size      uint32
	Hash      [20]byte
	Flags     uint16
	Path      string
}

// Index represents the Git index (staging area)
type Index struct {
	Entries []Entry
}

// NewIndex creates a new empty index
func NewIndex() *Index {
	return &Index{Entries: make([]Entry, 0)}
}

// ReadIndex reads the index file from the repository
func ReadIndex(repoPath string) (*Index, error) {
	indexPath := filepath.Join(repoPath, ".gogit", "index")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewIndex(), nil
		}
		return nil, fmt.Errorf("failed to read index: %w", err)
	}

	return parseIndex(data)
}

func parseIndex(data []byte) (*Index, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("index too small")
	}

	// Check signature
	sig := string(data[0:4])
	if sig != IndexSignature {
		return nil, fmt.Errorf("invalid index signature: %s", sig)
	}

	// Check version
	version := binary.BigEndian.Uint32(data[4:8])
	if version != IndexVersion {
		return nil, fmt.Errorf("unsupported index version: %d", version)
	}

	// Entry count
	entryCount := binary.BigEndian.Uint32(data[8:12])

	index := NewIndex()
	pos := 12

	for i := uint32(0); i < entryCount; i++ {
		if pos+62 > len(data) {
			return nil, fmt.Errorf("truncated index entry")
		}

		entry := Entry{}
		entry.CTimeSec = binary.BigEndian.Uint32(data[pos:])
		entry.CTimeNano = binary.BigEndian.Uint32(data[pos+4:])
		entry.MTimeSec = binary.BigEndian.Uint32(data[pos+8:])
		entry.MTimeNano = binary.BigEndian.Uint32(data[pos+12:])
		entry.Dev = binary.BigEndian.Uint32(data[pos+16:])
		entry.Ino = binary.BigEndian.Uint32(data[pos+20:])
		entry.Mode = binary.BigEndian.Uint32(data[pos+24:])
		entry.UID = binary.BigEndian.Uint32(data[pos+28:])
		entry.GID = binary.BigEndian.Uint32(data[pos+32:])
		entry.Size = binary.BigEndian.Uint32(data[pos+36:])
		copy(entry.Hash[:], data[pos+40:pos+60])
		entry.Flags = binary.BigEndian.Uint16(data[pos+60:])

		pos += 62

		// Read path (null-terminated)
		pathEnd := bytes.IndexByte(data[pos:], 0)
		if pathEnd == -1 {
			return nil, fmt.Errorf("invalid index entry: no null terminator")
		}
		entry.Path = string(data[pos : pos+pathEnd])
		pos += pathEnd + 1

		// Padding to 8-byte boundary
		entryLen := 62 + pathEnd + 1
		padding := (8 - (entryLen % 8)) % 8
		pos += padding

		index.Entries = append(index.Entries, entry)
	}

	return index, nil
}

// Write writes the index to the repository
func (idx *Index) Write(repoPath string) error {
	// Sort entries by path
	sort.Slice(idx.Entries, func(i, j int) bool {
		return idx.Entries[i].Path < idx.Entries[j].Path
	})

	var buf bytes.Buffer

	// Write header
	buf.WriteString(IndexSignature)
	binary.Write(&buf, binary.BigEndian, uint32(IndexVersion))
	binary.Write(&buf, binary.BigEndian, uint32(len(idx.Entries)))

	// Write entries
	for _, entry := range idx.Entries {
		binary.Write(&buf, binary.BigEndian, entry.CTimeSec)
		binary.Write(&buf, binary.BigEndian, entry.CTimeNano)
		binary.Write(&buf, binary.BigEndian, entry.MTimeSec)
		binary.Write(&buf, binary.BigEndian, entry.MTimeNano)
		binary.Write(&buf, binary.BigEndian, entry.Dev)
		binary.Write(&buf, binary.BigEndian, entry.Ino)
		binary.Write(&buf, binary.BigEndian, entry.Mode)
		binary.Write(&buf, binary.BigEndian, entry.UID)
		binary.Write(&buf, binary.BigEndian, entry.GID)
		binary.Write(&buf, binary.BigEndian, entry.Size)
		buf.Write(entry.Hash[:])
		binary.Write(&buf, binary.BigEndian, entry.Flags)
		buf.WriteString(entry.Path)
		buf.WriteByte(0)

		// Padding to 8-byte boundary
		entryLen := 62 + len(entry.Path) + 1
		padding := (8 - (entryLen % 8)) % 8
		for i := 0; i < padding; i++ {
			buf.WriteByte(0)
		}
	}

	// Calculate and append checksum
	checksum := sha1.Sum(buf.Bytes())
	buf.Write(checksum[:])

	indexPath := filepath.Join(repoPath, ".gogit", "index")
	return os.WriteFile(indexPath, buf.Bytes(), 0644)
}

// AddFile adds or updates a file in the index
func (idx *Index) AddFile(repoPath, filePath string) error {
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(repoPath, filePath)
	}

	// Get file info
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Read file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Compute hash
	hash := utils.HashObject("blob", content)
	hashBytes, _ := utils.HexToBytes(hash)

	// Get relative path
	relPath, err := filepath.Rel(repoPath, absPath)
	if err != nil {
		relPath = filePath
	}

	// Create entry
	entry := Entry{
		CTimeSec:  uint32(info.ModTime().Unix()),
		CTimeNano: uint32(info.ModTime().Nanosecond()),
		MTimeSec:  uint32(info.ModTime().Unix()),
		MTimeNano: uint32(info.ModTime().Nanosecond()),
		Mode:      0100644, // Regular file
		Size:      uint32(info.Size()),
		Flags:     uint16(len(relPath)),
		Path:      relPath,
	}
	copy(entry.Hash[:], hashBytes)

	if info.Mode()&0111 != 0 {
		entry.Mode = 0100755 // Executable
	}

	// Update or add entry
	idx.UpdateEntry(entry)

	return nil
}

// UpdateEntry updates an existing entry or adds a new one
func (idx *Index) UpdateEntry(entry Entry) {
	for i := range idx.Entries {
		if idx.Entries[i].Path == entry.Path {
			idx.Entries[i] = entry
			return
		}
	}
	idx.Entries = append(idx.Entries, entry)
}

// RemoveEntry removes an entry by path
func (idx *Index) RemoveEntry(path string) {
	for i := range idx.Entries {
		if idx.Entries[i].Path == path {
			idx.Entries = append(idx.Entries[:i], idx.Entries[i+1:]...)
			return
		}
	}
}

// GetEntry gets an entry by path
func (idx *Index) GetEntry(path string) *Entry {
	for i := range idx.Entries {
		if idx.Entries[i].Path == path {
			return &idx.Entries[i]
		}
	}
	return nil
}

// HashString returns the hash as a hex string
func (e *Entry) HashString() string {
	return utils.BytesToHex(e.Hash[:])
}

// ModTime returns the modification time
func (e *Entry) ModTime() time.Time {
	return time.Unix(int64(e.MTimeSec), int64(e.MTimeNano))
}
