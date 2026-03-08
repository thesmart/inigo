package ini

import (
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ********************************
// Intermediates and Methods
// ********************************

// IniFile is an intermediate structural representation of an ini configuration file.
type IniFile struct {
	Path           string
	cursor         *FileCursor
	stack          []*FileCursor
	visited        map[string]bool
	DefaultSection *Section
	sections       map[string]*Section
	sectionOrder   []string // insertion order of section keys (lowercased)
}

// Section represents a named group of key-value parameters.
type Section struct {
	Name       string
	params     map[string]*Param
	paramOrder []string // insertion order of param keys (lowercased)
}

// Param represents a single parameter with its raw string value.
type Param struct {
	Name   string
	Value  string
	cursor Cursor
}

// Get returns the section for the given name (case-insensitive), or DefaultSection for "".
func (f *IniFile) Get(name string) *Section {
	if name == "" {
		return f.DefaultSection
	}
	return f.sections[strings.ToLower(name)]
}

// Sections iterates over sections in insertion order, default section first.
func (f *IniFile) Sections() iter.Seq[*Section] {
	return func(yield func(*Section) bool) {
		if !yield(f.DefaultSection) {
			return
		}
		for _, key := range f.sectionOrder {
			if !yield(f.sections[key]) {
				return
			}
		}
	}
}

// Get returns the param value for the given name (case-insensitive) and whether it was found.
func (s *Section) Get(name string) (string, bool) {
	p, ok := s.params[strings.ToLower(name)]
	if !ok {
		return "", false
	}
	return p.Value, true
}

// Params iterates over params in insertion order, yielding (name, value) pairs.
func (s *Section) Params() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for _, key := range s.paramOrder {
			p := s.params[key]
			if !yield(p.Name, p.Value) {
				return
			}
		}
	}
}

// ********************************
// (Un)Marshaling and Parsing
// ********************************

// Load parses an ini file from disk and returns the parsed IniFile.
func Load(path string) (*IniFile, error) {
	return unmarshalIniFileIntermediate(path)
}

// Parse parses ini content from a string and returns the parsed IniFile.
// The path is used for include directive resolution and error messages.
func Parse(path string, contents string) (*IniFile, error) {
	return unmarshalIniStringIntermediate(path, contents)
}

// unmarshalIniFileIntermediate parses an ini file from disk into an IniFile intermediate.
func unmarshalIniFileIntermediate(path string) (*IniFile, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path %q: %w", path, err)
	}

	contents, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q: %w", absPath, err)
	}

	return unmarshalIniStringIntermediate(absPath, string(contents))
}

// unmarshalIniStringIntermediate parses ini contents from a string into an IniFile intermediate.
// The path is used for include directive resolution and error messages.
func unmarshalIniStringIntermediate(path string, contents string) (*IniFile, error) {
	iniFile := &IniFile{
		Path:     path,
		visited:  make(map[string]bool),
		sections: make(map[string]*Section),
		DefaultSection: &Section{
			Name:   "",
			params: make(map[string]*Param),
		},
	}

	fc := &FileCursor{
		Cursor:   Cursor{Line: 1, Offset: 0},
		Path:     path,
		Contents: contents,
	}

	iniFile.visited[path] = true
	iniFile.stack = append(iniFile.stack, fc)

	for len(iniFile.stack) > 0 {
		// pop from stack
		n := len(iniFile.stack)
		iniFile.cursor = iniFile.stack[n-1]
		iniFile.stack = iniFile.stack[:n-1]

		if err := parseFileCursor(iniFile); err != nil {
			return nil, err
		}
	}

	iniFile.cursor = nil
	return iniFile, nil
}

// parseFileCursor parses all lines from the current file cursor.
func parseFileCursor(iniFile *IniFile) error {
	fc := iniFile.cursor
	lines := strings.Split(fc.Contents, "\n")
	currentSection := iniFile.DefaultSection

	// Track which section was last active (for includes that resume)
	for i, line := range lines {
		fc.Line = int32(i + 1)
		fc.Offset = 0

		result, err := parseLine(line, fc)
		if err != nil {
			return err
		}

		switch result.kind {
		case lineBlank, lineComment:
			continue

		case lineSection:
			sectionKey := strings.ToLower(result.section)
			if _, ok := iniFile.sections[sectionKey]; ok {
				// Duplicate sections are ignored (first one wins).
				// Set currentSection to nil so subsequent parameters are discarded.
				currentSection = nil
			} else {
				sec := &Section{
					Name:   result.section,
					params: make(map[string]*Param),
				}
				iniFile.sections[sectionKey] = sec
				iniFile.sectionOrder = append(iniFile.sectionOrder, sectionKey)
				currentSection = sec
			}

		case lineParameter:
			if currentSection == nil {
				// Inside a duplicate section; ignore parameters.
				continue
			}
			paramKey := strings.ToLower(result.param.Name)
			if _, exists := currentSection.params[paramKey]; !exists {
				currentSection.paramOrder = append(currentSection.paramOrder, paramKey)
			}
			currentSection.params[paramKey] = result.param

		case lineInclude:
			if err := handleIncludeDirective(iniFile, result, currentSection); err != nil {
				return err
			}
			// After an include, the current file cursor contents are already fully split,
			// so we continue processing remaining lines. The include files were pushed
			// and processed inline via recursive calls. Actually, per the spec, we need
			// to push remaining lines and the include files onto the stack.
			// Let's restructure: we need to push the remaining lines as a new FileCursor
			// and then push the include files.
			remainingLines := strings.Join(lines[i+1:], "\n")
			if remainingLines != "" {
				remainFC := &FileCursor{
					Cursor:   Cursor{Line: int32(i + 2), Offset: 0},
					Path:     fc.Path,
					Contents: remainingLines,
				}
				// Push remaining lines first (they'll be processed after includes)
				iniFile.stack = append(iniFile.stack, remainFC)
			}
			// The include files are already on the stack from handleIncludeDirective
			return nil
		}
	}

	return nil
}

// handleIncludeDirective processes an include directive and pushes file cursors onto the stack.
func handleIncludeDirective(iniFile *IniFile, result *lineResult, currentSection *Section) error {
	fc := iniFile.cursor
	baseDir := filepath.Dir(fc.Path)
	resolved := resolvePath(result.includePath, baseDir)

	switch result.includeKind {
	case "include":
		return pushIncludeFile(iniFile, resolved, fc)
	case "include_if_exists":
		if _, err := os.Stat(resolved); err != nil {
			return nil // silently skip
		}
		return pushIncludeFile(iniFile, resolved, fc)
	case "include_dir":
		return pushIncludeDir(iniFile, resolved, fc)
	}
	return nil
}

// pushIncludeFile reads and pushes a single include file onto the stack.
func pushIncludeFile(iniFile *IniFile, absPath string, fc *FileCursor) error {
	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return parseError(fc, 0, "include", fmt.Sprintf("failed to resolve path %q: %v", absPath, err))
	}

	if iniFile.visited[absPath] {
		return parseError(fc, 0, "include", fmt.Sprintf("circular include detected: %q", absPath))
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return parseError(fc, 0, "include", fmt.Sprintf("file not found: %q", absPath))
	}
	if info.IsDir() {
		return parseError(fc, 0, "include", fmt.Sprintf("path is a directory: %q", absPath))
	}

	contents, err := os.ReadFile(absPath)
	if err != nil {
		return parseError(fc, 0, "include", fmt.Sprintf("failed to read file %q: %v", absPath, err))
	}

	iniFile.visited[absPath] = true
	newFC := &FileCursor{
		Cursor:   Cursor{Line: 1, Offset: 0},
		Path:     absPath,
		Contents: string(contents),
	}
	iniFile.stack = append(iniFile.stack, newFC)
	return nil
}

// pushIncludeDir reads a directory and pushes all .conf files onto the stack in reverse
// lexicographic order (so they're processed in lexicographic order when popped).
func pushIncludeDir(iniFile *IniFile, dirPath string, fc *FileCursor) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return parseError(fc, 0, "include_dir", fmt.Sprintf("failed to read directory %q: %v", dirPath, err))
	}

	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".conf") {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)

	// Push in reverse order so first file is on top of stack
	for i := len(files) - 1; i >= 0; i-- {
		absPath := filepath.Join(dirPath, files[i])
		if err := pushIncludeFile(iniFile, absPath, fc); err != nil {
			return err
		}
	}
	return nil
}

// resolvePath resolves a relative path against a base directory.
func resolvePath(path, baseDir string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}
