// Package pgini implements cursors and data structures for parsing PGINI
// configuration files, a PostgreSQL-compatible INI format.
//
// See reference/pgini-agents.md for the specification this package implements.
// DO NOT CHANGE THIS FILE TO MATCH OTHER FILES.
package pgini

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// identifierRe validates PGINI identifiers: [A-Za-z_][A-Za-z0-9_.\-]*
var identifierRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.\-]*$`)

// RootCursor is the root cursor for iterating over a PGINI file and the tree
// of included files in pre-order.
type RootCursor struct {
	Path string
	// Current file and position for cursing
	current *FileCursor
	// Included files for pre-order traversal
	stack []*FileCursor
	// Prevents circular loops
	visited map[string]bool
	// Currently opened section name while parsing
	openSection string
	// Sections by name (lowercased) in all parsed files
	sections map[string]*Section
	// insertion order of section names (lowercased)
	sectionOrder []string
}

// NewRootCursor reads the file at path and returns a new RootCursor.
func NewRootCursor(path string) (*RootCursor, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path %q: %w", path, err)
	}

	contents, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q: %w", absPath, err)
	}

	c := &RootCursor{
		Path: absPath,
		current: &FileCursor{
			Path:       absPath,
			contents:   strings.Split(string(contents), "\n"),
			lineOffset: 0,
			charOffset: 0,
		},
		stack:        make([]*FileCursor, 0),
		visited:      make(map[string]bool),
		openSection:  "",
		sections:     make(map[string]*Section),
		sectionOrder: make([]string, 0),
	}
	c.visited[absPath] = true
	return c, nil
}

// AddInclude pushes a new included file onto the traversal stack.
// It returns an error if the path creates a circular include.
func (c *RootCursor) AddInclude(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path %q: %w", path, err)
	}

	if c.current == nil {
		return errors.New("IncludesCursor#Add: unable to push new include, current is nil")
	}
	if c.visited[absPath] {
		return fmt.Errorf("%s:%d:%d: %s", c.current.Path, c.current.lineOffset, c.current.charOffset, "circular include detected")
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
func (c *RootCursor) NextInclude() bool {
	n := len(c.stack)
	if n == 0 {
		return false
	}
	c.current = c.stack[n-1]
	c.stack = c.stack[:n-1]
	return true
}

// GetSection returns the section for the given name (case-insensitive),
// or nil if not found.
func (c *RootCursor) GetSection(name string) *Section {
	s, ok := c.sections[strings.ToLower(name)]
	if !ok {
		return nil
	}
	return s
}

// AddSection creates or reopens a section with the given name.
// The name is normalized to lowercase per the PGINI spec. An empty name
// or "default" refers to the default section (stored as empty string).
// It returns an error if name is not a valid PGINI identifier.
func (c *RootCursor) AddSection(name string) (*Section, error) {
	lower := strings.ToLower(name)

	// "default" is the alias for the unnamed default section.
	if lower == "default" {
		lower = ""
	}

	// Validate non-empty names as identifiers.
	if lower != "" && !identifierRe.MatchString(lower) {
		return nil, fmt.Errorf("invalid section name %q: must match [A-Za-z_][A-Za-z0-9_.\\-]*", name)
	}

	if s, ok := c.sections[lower]; ok {
		return s, nil
	}
	s := &Section{
		Name:   lower,
		params: make(map[string]*Param),
	}
	c.sections[lower] = s
	c.sectionOrder = append(c.sectionOrder, lower)
	return s, nil
}

// String returns a human-readable position string.
func (c *RootCursor) String() string {
	if c.current == nil {
		return fmt.Sprintf("RootCursor (%q): %q", path.Base(c.Path), c.Path)
	}
	return fmt.Sprintf("RootCursor (%q): %q:%d:%d", path.Base(c.Path), c.current.Path, c.current.lineOffset+1, c.current.charOffset+1)
}

// FileCursor tracks a parser position within a single file.
type FileCursor struct {
	Path       string
	contents   []string
	lineOffset int // 0-indexed
	charOffset int // 0-indexed
}

// NewFileCursor reads the file at path and returns a new FileCursor positioned
// at the first line.
func NewFileCursor(path string) (*FileCursor, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path %q: %w", path, err)
	}

	c := &FileCursor{Path: absPath}
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
func (c *FileCursor) NextLine() bool {
	if c.lineOffset+1 >= len(c.contents) {
		return false
	}
	c.lineOffset++
	c.charOffset = 0
	return true
}

// NextChar advances the character offset within the current line.
// It returns false if already at the last character.
func (c *FileCursor) NextChar() bool {
	if c.lineOffset < 0 || c.lineOffset >= len(c.contents) {
		return false
	}

	line := c.contents[c.lineOffset]
	if c.charOffset+1 >= len(line) {
		return false
	}

	c.charOffset++
	return true
}

// String returns a human-readable position string.
func (c *FileCursor) String() string {
	// Output adjusts 0-indexed to 1-indexed
	return fmt.Sprintf("FileCursor: %q:%d:%d", c.Path, c.lineOffset+1, c.charOffset+1)
}

// Section represents a named group of key-value parameters.
// The Name is stored lowercase; an empty Name represents the default section.
type Section struct {
	Name       string
	params     map[string]*Param
	paramOrder []string
}

// SetParam sets or overwrites a parameter in the section. The key is normalized
// to lowercase per the PGINI spec (keys are case-insensitive). Duplicate keys
// update the existing value (last occurrence wins).
// It returns an error if name is not a valid PGINI identifier.
func (s *Section) SetParam(name string, value string) (*Param, error) {
	lower := strings.ToLower(name)
	if !identifierRe.MatchString(lower) {
		return nil, fmt.Errorf("invalid parameter key %q: must match [A-Za-z_][A-Za-z0-9_.\\-]*", name)
	}

	if p, ok := s.params[lower]; ok {
		p.Value = value
		return p, nil
	}
	p := &Param{
		Name:  lower,
		Value: value,
	}
	s.params[lower] = p
	s.paramOrder = append(s.paramOrder, lower)
	return p, nil
}

// GetParam returns the parameter for the given key (case-insensitive)
// and whether it was found.
func (s *Section) GetParam(name string) (*Param, bool) {
	p, ok := s.params[strings.ToLower(name)]
	return p, ok
}

// GetValue returns the string value for the given key (case-insensitive)
// and whether it was found.
func (s *Section) GetValue(name string) (string, bool) {
	p, ok := s.params[strings.ToLower(name)]
	if !ok {
		return "", false
	}
	return p.Value, true
}

// Param represents a single parameter with its raw string value.
type Param struct {
	Name  string
	Value string
}
