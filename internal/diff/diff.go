package diff

import (
	"fmt"
	"strings"
)

// Line represents a line in a diff
type Line struct {
	Text   string
	Number int
}

// Change represents a single change in the diff
type Change struct {
	Type    ChangeType
	OldLine int
	NewLine int
	Text    string
}

// ChangeType represents the type of change
type ChangeType int

const (
	ChangeEqual ChangeType = iota
	ChangeInsert
	ChangeDelete
)

// Diff computes the difference between two strings
func Diff(oldText, newText string) []Change {
	oldLines := strings.Split(oldText, "\n")
	newLines := strings.Split(newText, "\n")

	return diffLines(oldLines, newLines)
}

// diffLines implements a simple line-based diff algorithm
// This is a simplified version - a full implementation would use Myers diff
func diffLines(oldLines, newLines []string) []Change {
	// Simple LCS-based diff
	m, n := len(oldLines), len(newLines)

	// Create LCS table
	lcs := make([][]int, m+1)
	for i := range lcs {
		lcs[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if oldLines[i-1] == newLines[j-1] {
				lcs[i][j] = lcs[i-1][j-1] + 1
			} else {
				lcs[i][j] = max(lcs[i-1][j], lcs[i][j-1])
			}
		}
	}

	// Backtrack to find changes
	i, j := m, n
	var result []Change

	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			result = append(result, Change{
				Type:    ChangeEqual,
				OldLine: i,
				NewLine: j,
				Text:    oldLines[i-1],
			})
			i--
			j--
		} else if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
			result = append(result, Change{
				Type:    ChangeInsert,
				NewLine: j,
				Text:    newLines[j-1],
			})
			j--
		} else if i > 0 {
			result = append(result, Change{
				Type:    ChangeDelete,
				OldLine: i,
				Text:    oldLines[i-1],
			})
			i--
		}
	}

	// Reverse to get correct order
	for left, right := 0, len(result)-1; left < right; left, right = left+1, right-1 {
		result[left], result[right] = result[right], result[left]
	}

	return result
}

// Format formats the diff as a unified diff string
func Format(oldName, newName string, changes []Change) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("--- a/%s\n", oldName))
	sb.WriteString(fmt.Sprintf("+++ b/%s\n", newName))

	// Group changes into hunks
	hunks := groupIntoHunks(changes, 3)

	for _, hunk := range hunks {
		oldStart, oldCount, newStart, newCount := hunkHeader(hunk)
		sb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount))

		for _, change := range hunk {
			switch change.Type {
			case ChangeEqual:
				sb.WriteString(fmt.Sprintf(" %s\n", change.Text))
			case ChangeInsert:
				sb.WriteString(fmt.Sprintf("\033[32m+%s\033[0m\n", change.Text))
			case ChangeDelete:
				sb.WriteString(fmt.Sprintf("\033[31m-%s\033[0m\n", change.Text))
			}
		}
	}

	return sb.String()
}

// groupIntoHunks groups changes into hunks with context
func groupIntoHunks(changes []Change, context int) [][]Change {
	if len(changes) == 0 {
		return nil
	}

	var hunks [][]Change
	var currentHunk []Change
	lastChangeIdx := -1

	for i, change := range changes {
		if change.Type != ChangeEqual {
			// Start new hunk if needed
			if lastChangeIdx == -1 || i-lastChangeIdx > context*2 {
				if len(currentHunk) > 0 {
					hunks = append(hunks, currentHunk)
				}
				currentHunk = nil

				// Add leading context
				start := i - context
				if start < 0 {
					start = 0
				}
				for j := start; j < i; j++ {
					currentHunk = append(currentHunk, changes[j])
				}
			} else {
				// Add lines since last change
				for j := lastChangeIdx + 1; j < i; j++ {
					currentHunk = append(currentHunk, changes[j])
				}
			}

			currentHunk = append(currentHunk, change)
			lastChangeIdx = i
		}
	}

	// Add trailing context
	if lastChangeIdx != -1 {
		end := lastChangeIdx + context + 1
		if end > len(changes) {
			end = len(changes)
		}
		for j := lastChangeIdx + 1; j < end; j++ {
			currentHunk = append(currentHunk, changes[j])
		}
		hunks = append(hunks, currentHunk)
	}

	return hunks
}

// hunkHeader calculates the hunk header values
func hunkHeader(hunk []Change) (oldStart, oldCount, newStart, newCount int) {
	if len(hunk) == 0 {
		return 1, 0, 1, 0
	}

	// Find first line numbers
	for _, change := range hunk {
		if change.OldLine > 0 {
			oldStart = change.OldLine
			break
		}
	}
	for _, change := range hunk {
		if change.NewLine > 0 {
			newStart = change.NewLine
			break
		}
	}

	if oldStart == 0 {
		oldStart = 1
	}
	if newStart == 0 {
		newStart = 1
	}

	// Count lines
	for _, change := range hunk {
		switch change.Type {
		case ChangeEqual:
			oldCount++
			newCount++
		case ChangeDelete:
			oldCount++
		case ChangeInsert:
			newCount++
		}
	}

	return
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
