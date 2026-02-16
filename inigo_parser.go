package inigo

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Config represents a parsed INI configuration file.
type Config struct {
	sections       map[string]*Section
	defaultSection *Section
}

// Section represents a named group of key-value parameters.
type Section struct {
	name   string
	params map[string]*Param
}

// Param represents a single parameter with its raw string value.
type Param struct {
	name  string
	value string
}

// parser holds mutable state used during INI file parsing.
// It is shared across recursive include calls so that section context
// is preserved ("as if inserted at that point").
type parser struct {
	config  *Config
	section *Section
	visited map[string]bool // tracks file paths to prevent circular includes
}

// Load parses an INI file at the given path.
// Include directives are resolved relative to the file's directory.
func Load(path string) (*Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path %q: %w", path, err)
	}
	p := newParser()
	if err := p.loadFile(absPath); err != nil {
		return nil, err
	}
	return p.config, nil
}

// Parse reads INI content from a reader.
// Include directives are not supported when parsing from a reader
// since there is no base directory for path resolution.
func Parse(r io.Reader) (*Config, error) {
	p := newParser()
	if err := p.parse(r, ""); err != nil {
		return nil, err
	}
	return p.config, nil
}

func newParser() *parser {
	f := &Config{
		sections: make(map[string]*Section),
	}
	defaultSec := &Section{name: "", params: make(map[string]*Param)}
	f.sections[""] = defaultSec
	f.defaultSection = defaultSec

	return &parser{
		config:  f,
		section: defaultSec,
		visited: make(map[string]bool),
	}
}

// Section returns the section with the given name, or nil if not found.
func (f *Config) Section(name string) *Section {
	return f.sections[name]
}

// HasSection reports whether the named section exists.
func (f *Config) HasSection(name string) bool {
	_, ok := f.sections[name]
	return ok
}

// SectionNames returns all section names, excluding the default section.
func (f *Config) SectionNames() []string {
	var names []string
	for name := range f.sections {
		if name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// HasParam reports whether the section contains the named parameter.
func (s *Section) HasParam(name string) bool {
	_, ok := s.params[strings.ToLower(name)]
	return ok
}

// GetParam returns the key with the given name.
// Returns a key with empty value if not found.
func (s *Section) GetParam(name string) *Param {
	k, ok := s.params[strings.ToLower(name)]
	if !ok {
		return &Param{name: name, value: ""}
	}
	return k
}

// AllParams returns all key names in the section, sorted alphabetically.
func (s *Section) AllParams() []string {
	var names []string
	for name := range s.params {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// String returns the raw string value.
func (k *Param) String() string {
	return k.value
}

// Bool interprets the value as a boolean.
// Accepts: on/off, true/false, yes/no, 1/0 (case-insensitive),
// or any unambiguous prefix of these words.
func (k *Param) Bool() (bool, error) {
	return parseBool(k.value)
}

// Int interprets the value as an integer.
// Supports decimal, hexadecimal (0x prefix), and octal (0 prefix).
// Fractional values are rounded to the nearest integer.
func (k *Param) Int() (int64, error) {
	return parseInt(k.value)
}

// Float64 interprets the value as a floating-point number.
func (k *Param) Float64() (float64, error) {
	s := strings.TrimSpace(k.value)
	if s == "" {
		return 0, fmt.Errorf("empty numeric value")
	}
	return strconv.ParseFloat(s, 64)
}

// --- parser methods ---

// loadFile reads and parses an INI file, tracking visited paths to prevent circular includes.
func (p *parser) loadFile(absPath string) error {
	if p.visited[absPath] {
		return fmt.Errorf("circular include detected: %q", absPath)
	}
	p.visited[absPath] = true

	file, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("failed to open %q: %w", absPath, err)
	}
	defer file.Close()

	return p.parse(file, filepath.Dir(absPath))
}

// parse processes INI content line by line from a reader.
// baseDir is used to resolve relative paths in include directives;
// if empty, include directives will produce an error.
func (p *parser) parse(r io.Reader, baseDir string) error {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()

		// Remove comments outside of quoted strings
		line = stripComment(line)
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// Section header: [name]
		if strings.HasPrefix(line, "[") {
			name, err := parseSectionHeader(line)
			if err != nil {
				return err
			}
			if sec, ok := p.config.sections[name]; ok {
				p.section = sec
			} else {
				sec = &Section{name: name, params: make(map[string]*Param)}
				p.config.sections[name] = sec
				p.section = sec
			}
			continue
		}

		// Include directives (checked before key-value pairs)
		if handled, err := p.handleInclude(line, baseDir); err != nil {
			return err
		} else if handled {
			continue
		}

		// Key = value parameter
		key, value, err := parseKeyValue(line)
		if err != nil {
			return err
		}

		// Parameter names are case-insensitive; last value wins
		lowerKey := strings.ToLower(key)
		p.section.params[lowerKey] = &Param{name: key, value: value}
	}

	return scanner.Err()
}

// handleInclude checks if the line is an include directive and processes it.
// Returns true if the line was handled as an include directive.
func (p *parser) handleInclude(line, baseDir string) (bool, error) {
	lower := strings.ToLower(line)

	var directive, rest string
	// Check longest directive names first to avoid prefix collisions
	switch {
	case matchDirective(lower, line, "include_if_exists", &directive, &rest):
	case matchDirective(lower, line, "include_dir", &directive, &rest):
	case matchDirective(lower, line, "include", &directive, &rest):
	default:
		return false, nil
	}

	if baseDir == "" {
		return false, fmt.Errorf("%s: cannot resolve paths without a base directory", directive)
	}

	path, err := parseIncludePath(rest)
	if err != nil {
		return false, fmt.Errorf("%s: %w", directive, err)
	}

	resolved := resolvePath(path, baseDir)

	switch directive {
	case "include":
		return true, p.loadFile(resolved)
	case "include_if_exists":
		if _, err := os.Stat(resolved); err == nil {
			return true, p.loadFile(resolved)
		}
		return true, nil
	case "include_dir":
		return true, p.loadDir(resolved)
	}

	return false, nil
}

// loadDir includes all .conf files in the directory, sorted by ASCII order.
// Files starting with '.' are excluded.
func (p *parser) loadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory %q: %w", dir, err)
	}

	// Collect .conf files, excluding dotfiles
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".conf") {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)

	for _, name := range files {
		if err := p.loadFile(filepath.Join(dir, name)); err != nil {
			return err
		}
	}
	return nil
}

// --- pure parsing functions ---

// stripComment removes the comment portion of a line (starting with #),
// while preserving # characters inside single-quoted strings.
func stripComment(line string) string {
	inQuote := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if inQuote {
			if ch == '\\' && i+1 < len(line) && line[i+1] == '\'' {
				i++ // skip backslash-escaped quote
			} else if ch == '\'' {
				if i+1 < len(line) && line[i+1] == '\'' {
					i++ // skip doubled quote
				} else {
					inQuote = false
				}
			}
		} else {
			switch ch {
			case '\'':
				inQuote = true
			case '#':
				return line[:i]
			}
		}
	}
	return line
}

// parseSectionHeader extracts the section name from a "[name]" line.
func parseSectionHeader(line string) (string, error) {
	end := strings.IndexByte(line, ']')
	if end < 0 {
		return "", fmt.Errorf("unterminated section header: %q", line)
	}
	name := strings.TrimSpace(line[1:end])
	if name == "" {
		return "", fmt.Errorf("empty section name")
	}
	return name, nil
}

// parseKeyValue splits a line into parameter name and value.
// The = sign separates name from value; value is optional and defaults to empty string.
func parseKeyValue(line string) (string, string, error) {
	nameStr, valueStr, hasEquals := strings.Cut(line, "=")
	name := strings.TrimSpace(nameStr)
	if !isValidParamName(name) {
		return "", "", fmt.Errorf("invalid parameter name: %q", name)
	}

	if !hasEquals {
		// Bare parameter name without =
		return name, "", nil
	}

	rawValue := strings.TrimSpace(valueStr)
	value, err := parseValue(rawValue)
	if err != nil {
		return "", "", fmt.Errorf("parameter %q: %w", name, err)
	}

	return name, value, nil
}

// parseValue extracts a parameter value, handling single-quoted strings.
func parseValue(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	if raw[0] == '\'' {
		return parseQuotedValue(raw)
	}
	return raw, nil
}

// parseQuotedValue extracts a single-quoted string value.
// Embedded single quotes can be escaped as ” (doubled) or \' (backslash).
func parseQuotedValue(raw string) (string, error) {
	var buf strings.Builder
	i := 1 // skip opening quote
	for i < len(raw) {
		ch := raw[i]
		if ch == '\\' && i+1 < len(raw) && raw[i+1] == '\'' {
			// Backslash-escaped quote
			buf.WriteByte('\'')
			i += 2
		} else if ch == '\'' {
			if i+1 < len(raw) && raw[i+1] == '\'' {
				// Doubled quote escape
				buf.WriteByte('\'')
				i += 2
			} else {
				// Closing quote
				return buf.String(), nil
			}
		} else {
			buf.WriteByte(ch)
			i++
		}
	}
	return "", fmt.Errorf("unterminated single-quoted string")
}

// isValidParamName checks that a parameter name follows the naming rules:
// starts with a-z, A-Z, or _, followed by a-z, A-Z, 0-9, _, or $.
func isValidParamName(name string) bool {
	if name == "" {
		return false
	}
	first := name[0]
	if !isAlpha(first) && first != '_' {
		return false
	}
	for i := 1; i < len(name); i++ {
		ch := name[i]
		if !isAlpha(ch) && !isDigit(ch) && ch != '_' && ch != '$' {
			return false
		}
	}
	return true
}

// matchDirective checks if lower starts with the named directive followed by
// whitespace or a quote. If so, sets directive and rest and returns true.
func matchDirective(lower, original, name string, directive, rest *string) bool {
	if !strings.HasPrefix(lower, name) || len(original) <= len(name) {
		return false
	}
	// The character after the directive must be a separator
	next := original[len(name)]
	if next != ' ' && next != '\t' && next != '\'' {
		return false
	}
	*directive = name
	*rest = strings.TrimSpace(original[len(name):])
	return true
}

// parseIncludePath extracts a file path from an include directive argument.
// The path may be single-quoted or bare.
func parseIncludePath(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("missing path")
	}

	if s[0] == '\'' {
		// Quoted path
		end := strings.IndexByte(s[1:], '\'')
		if end < 0 {
			return "", fmt.Errorf("unterminated quoted path")
		}
		return s[1 : end+1], nil
	}

	// Unquoted path: take until whitespace
	if idx := strings.IndexFunc(s, unicode.IsSpace); idx >= 0 {
		return s[:idx], nil
	}
	return s, nil
}

// resolvePath makes a relative path absolute using baseDir.
func resolvePath(path, baseDir string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}

var trueWords = []string{"on", "true", "yes"}
var falseWords = []string{"off", "false", "no"}

// parseBool interprets a string as a boolean value.
// Accepts on/off, true/false, yes/no, 1/0 (case-insensitive),
// or any unambiguous prefix of these words.
func parseBool(s string) (bool, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return false, fmt.Errorf("empty boolean value")
	}
	if s == "1" {
		return true, nil
	}
	if s == "0" {
		return false, nil
	}

	// Check if s is an unambiguous prefix of any boolean word
	matchesTrue := false
	matchesFalse := false
	for _, w := range trueWords {
		if strings.HasPrefix(w, s) {
			matchesTrue = true
		}
	}
	for _, w := range falseWords {
		if strings.HasPrefix(w, s) {
			matchesFalse = true
		}
	}

	if matchesTrue && !matchesFalse {
		return true, nil
	}
	if matchesFalse && !matchesTrue {
		return false, nil
	}
	if matchesTrue && matchesFalse {
		return false, fmt.Errorf("ambiguous boolean value: %q", s)
	}
	return false, fmt.Errorf("invalid boolean value: %q", s)
}

// parseInt interprets a string as an integer value.
// Supports decimal, hexadecimal (0x prefix), and octal (0 prefix).
// Fractional values are rounded to the nearest integer.
func parseInt(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty integer value")
	}

	// Try integer parsing with automatic base detection (decimal, 0x hex, 0 octal)
	if n, err := strconv.ParseInt(s, 0, 64); err == nil {
		return n, nil
	}

	// Fall back to float parsing and round to nearest integer
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int64(math.Round(f)), nil
	}

	return 0, fmt.Errorf("invalid integer value: %q", s)
}

func isAlpha(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
