// Cursors are for parsing PGINI configuration files, a PostgreSQL-compatible
// INI format. They load data into intermediates: `IniFile`, `Section`, and
// `Param`.
//
// See reference/pgini-agents.md for the specification this package implements.
// This is the root PGINI implementation: other source files in the package
// should change to match it, but this file should not change to match others.

package pgini

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// RootCursor is a "start-before-first" iterator over a PGINI file and the tree
// of included files in pre-order. Call `NextInclude()` on a new FileCursor
// before attempting to read.
type RootCursor struct {
	// File holds the parsed sections, path, and ordering.
	File *IniFile
	// Current file and position for cursing
	current *FileCursor
	// Included files for pre-order traversal
	stack []*FileCursor
	// Prevents circular loops
	visited map[string]bool
}

// NewRootCursor reads the file at path and returns a new RootCursor.
func NewRootCursor(filePath string) (*RootCursor, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path %q: %w", filePath, err)
	}

	contents, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q: %w", absPath, err)
	}

	f, err := NewIniFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to construct IniFile %q: %w", absPath, err)
	}

	c := &RootCursor{
		File: f,
		current: &FileCursor{
			Path:       absPath,
			contents:   strings.Split(string(contents), "\n"),
			lineOffset: 0,
			byteOffset: 0,
		},
		stack:   make([]*FileCursor, 0),
		visited: make(map[string]bool),
	}
	c.AddInclude(absPath)
	return c, nil
}

// AddInclude pushes a new included file onto the traversal stack.
// Relative paths are resolved against the directory of the current file.
// It returns an error if the path creates a circular include.
func (c *RootCursor) AddInclude(includePath string) error {
	if c.current == nil {
		return errors.New("IncludesCursor#Add: unable to push new include, current is nil")
	}

	// Resolve relative paths against the directory of the current file.
	if !filepath.IsAbs(includePath) {
		includePath = filepath.Join(filepath.Dir(c.current.Path), includePath)
	}
	absPath, err := filepath.Abs(includePath)
	if err != nil {
		return fmt.Errorf("failed to resolve path %q: %w", includePath, err)
	}
	if c.visited[absPath] {
		return fmt.Errorf("%s:%d:%d: %s", c.current.Path, c.current.lineOffset, c.current.byteOffset, "circular include detected")
	}

	next, err := NewFileCursor(absPath)
	if err != nil {
		return err
	}
	c.stack = append(c.stack, next)
	c.visited[absPath] = true

	return nil
}

// NextInclude pops the next included file from the stack and makes it current.
// It returns false when the stack is empty.
func (c *RootCursor) NextInclude() *FileCursor {
	n := len(c.stack)
	if n == 0 {
		return nil
	}
	c.current = c.stack[n-1]
	c.stack = c.stack[:n-1]
	return c.current
}

// String returns a human-readable position string.
func (c *RootCursor) String() string {
	name := ""
	filePath := ""
	if c.File != nil {
		name = c.File.Name
		filePath = c.File.Path
	}
	if c.current == nil {
		return fmt.Sprintf("RootCursor (%q): %q", name, filePath)
	}
	return fmt.Sprintf("RootCursor (%q): %q:%d:%d", name, c.current.Path, c.current.lineOffset+1, c.current.byteOffset+1)
}

// FileCursor is a "start-before-first" iterator that iterates over
// line and character positions within a single file. Call `NextLine()`
// and `NextChar()` on a new FileCursor before attempting to read.
type FileCursor struct {
	Path       string
	contents   []string
	lineOffset int // 0-indexed
	byteOffset int // 0-indexed
}

// NewFileCursor reads the file at path and returns a new FileCursor positioned
// at the first line.
func NewFileCursor(path string) (*FileCursor, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path %q: %w", path, err)
	}

	c := &FileCursor{Path: absPath, lineOffset: -1, byteOffset: -1}
	contents, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q: %w", absPath, err)
	}
	c.contents = strings.Split(string(contents), "\n")
	return c, nil
}

// GetLine returns the current line and true, or empty string and false if
// the cursor is past the end of the file.
func (c *FileCursor) GetLine() (string, bool) {
	if c.lineOffset < 0 || c.lineOffset >= len(c.contents) {
		return "", false
	}
	return c.contents[c.lineOffset], true
}

// NextLine advances to the next line and resets the character offset.
// It returns false if already at the last line.
func (c *FileCursor) NextLine() (string, bool) {
	if c.lineOffset+1 >= len(c.contents) {
		return "", false
	}
	c.lineOffset++
	c.byteOffset = 0
	return c.GetLine()
}

// NextChar advances the character offset within the current line.
// It returns false if already at the last character.
func (c *FileCursor) NextChar() (rune, bool) {
	line, err := c.GetLine()
	if err {
		return 0, false
	}

	if c.byteOffset < 0 {
		// first call
		c.byteOffset = 0
	}

	if c.byteOffset >= len(line) {
		// no more bytes to read
		return 0, false
	}

	r, size := utf8.DecodeRuneInString(line[c.byteOffset:])
	c.byteOffset += size

	return r, true
}

// String returns a human-readable position string.
func (c *FileCursor) String() string {
	// Output adjusts 0-indexed to 1-indexed
	return fmt.Sprintf("FileCursor: %q:%d:%d", c.Path, c.lineOffset+1, c.byteOffset+1)
}
