package object

import (
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/gogit/internal/utils"
)

// Commit represents a Git commit object
type Commit struct {
	TreeHash    string
	ParentHash  string // Empty for initial commit, single parent for now
	Author      string
	AuthorTime  time.Time
	Committer   string
	CommitTime  time.Time
	Message     string
}

// NewCommit creates a new Commit
func NewCommit(treeHash, parentHash, author, message string) *Commit {
	now := time.Now()
	return &Commit{
		TreeHash:   treeHash,
		ParentHash: parentHash,
		Author:     author,
		AuthorTime: now,
		Committer:  author,
		CommitTime: now,
		Message:    message,
	}
}

// Type returns the object type
func (c *Commit) Type() Type {
	return TypeCommit
}

// Content returns the commit content in Git format
func (c *Commit) Content() []byte {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("tree %s\n", c.TreeHash))

	if c.ParentHash != "" {
		sb.WriteString(fmt.Sprintf("parent %s\n", c.ParentHash))
	}

	// Format: "author Name <email> timestamp timezone"
	authorTime := c.AuthorTime.Unix()
	_, offset := c.AuthorTime.Zone()
	tzOffset := fmt.Sprintf("%+03d%02d", offset/3600, (offset%3600)/60)
	sb.WriteString(fmt.Sprintf("author %s %d %s\n", c.Author, authorTime, tzOffset))
	sb.WriteString(fmt.Sprintf("committer %s %d %s\n", c.Committer, c.CommitTime.Unix(), tzOffset))

	sb.WriteString("\n")
	sb.WriteString(c.Message)
	if !strings.HasSuffix(c.Message, "\n") {
		sb.WriteString("\n")
	}

	return []byte(sb.String())
}

// Hash computes the SHA-1 hash of the commit
func (c *Commit) Hash() string {
	return utils.HashObject(string(TypeCommit), c.Content())
}

// ParseCommit parses commit content into a Commit object
func ParseCommit(content []byte) (*Commit, error) {
	commit := &Commit{}
	lines := strings.Split(string(content), "\n")

	inMessage := false
	var messageLines []string

	for _, line := range lines {
		if inMessage {
			messageLines = append(messageLines, line)
			continue
		}

		if line == "" {
			inMessage = true
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "tree":
			commit.TreeHash = value
		case "parent":
			commit.ParentHash = value
		case "author":
			commit.Author, commit.AuthorTime = parseAuthorLine(value)
		case "committer":
			commit.Committer, commit.CommitTime = parseAuthorLine(value)
		}
	}

	commit.Message = strings.TrimRight(strings.Join(messageLines, "\n"), "\n")

	return commit, nil
}

// parseAuthorLine parses "Name <email> timestamp timezone"
func parseAuthorLine(line string) (string, time.Time) {
	// Find the last two space-separated values (timestamp and timezone)
	parts := strings.Split(line, " ")
	if len(parts) < 3 {
		return line, time.Time{}
	}

	// Timezone is last, timestamp is second to last
	tzStr := parts[len(parts)-1]
	tsStr := parts[len(parts)-2]
	name := strings.Join(parts[:len(parts)-2], " ")

	var ts int64
	fmt.Sscanf(tsStr, "%d", &ts)

	// Parse timezone offset
	var tzHour, tzMin int
	fmt.Sscanf(tzStr, "%03d%02d", &tzHour, &tzMin)
	offset := tzHour*3600 + tzMin*60

	loc := time.FixedZone("", offset)
	t := time.Unix(ts, 0).In(loc)

	return name, t
}

// PrettyPrint returns a formatted representation of the commit
func (c *Commit) PrettyPrint() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("tree %s\n", c.TreeHash))
	if c.ParentHash != "" {
		sb.WriteString(fmt.Sprintf("parent %s\n", c.ParentHash))
	}

	authorTime := c.AuthorTime.Unix()
	_, offset := c.AuthorTime.Zone()
	tzOffset := fmt.Sprintf("%+03d%02d", offset/3600, (offset%3600)/60)
	sb.WriteString(fmt.Sprintf("author %s %d %s\n", c.Author, authorTime, tzOffset))
	sb.WriteString(fmt.Sprintf("committer %s %d %s\n", c.CommitTime.Unix(), c.Committer, tzOffset))
	sb.WriteString("\n")
	sb.WriteString(c.Message)
	sb.WriteString("\n")

	return sb.String()
}

// ShortHash returns the first 7 characters of the hash
func (c *Commit) ShortHash() string {
	hash := c.Hash()
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}
