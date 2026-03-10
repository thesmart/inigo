// Defines the parsed, intermediate data representation of an INI configuration
// file that follows the PGINI standard.

package pgini

import (
	"fmt"
	"iter"
	"path"
	"regexp"
	"strings"
)

// unquotedValueRe matches values that can appear unquoted in PGINI output.
// Per the grammar: unquoted-value ::= safe-char+ where safe-char is
// letter | digit | [_.\-]. This covers booleans (true, false, on, off,
// yes, no, 1, 0), integers (100, 0xFF, 077), and floats (1.5, 0.001).
var unquotedValueRe = regexp.MustCompile(`^[a-zA-Z0-9_.\-]+$`)

// IniFile represents a parsed PGINI configuration file.
type IniFile struct {
	// Path is the absolute path to the file.
	Path string
	// Name is the base filename of Path.
	Name string
	// sections by name (lowercased)
	sections map[string]*Section
	// insertion order of section names (lowercased)
	sectionOrder []string
}

// NewIniFile creates a new empty IniFile for the given path.
func NewIniFile(filePath string) *IniFile {
	return &IniFile{
		Path:         filePath,
		Name:         path.Base(filePath),
		sections:     make(map[string]*Section),
		sectionOrder: make([]string, 0),
	}
}

// GetSection returns the section for the given name (case-insensitive),
// or nil if not found.
func (f *IniFile) GetSection(name string) *Section {
	s, ok := f.sections[strings.ToLower(name)]
	if !ok {
		return nil
	}
	return s
}

// AddSection creates or reopens a section with the given name.
// The name is normalized to lowercase per the PGINI spec. An empty name
// or "default" refers to the default section (stored as empty string).
// It returns an error if name is not a valid PGINI identifier.
func (f *IniFile) AddSection(name string) (*Section, error) {
	s, err := NewSection(name)
	if err != nil {
		return nil, err
	}

	if existing, ok := f.sections[s.Name]; ok {
		return existing, nil
	}
	f.sections[s.Name] = s
	f.sectionOrder = append(f.sectionOrder, s.Name)
	return s, nil
}

// RemoveSection removes the section with the given name (case-insensitive).
// It returns true if the section existed and was removed.
func (f *IniFile) RemoveSection(name string) bool {
	lower := strings.ToLower(name)
	if lower == "default" {
		lower = ""
	}

	if _, ok := f.sections[lower]; !ok {
		return false
	}

	delete(f.sections, lower)
	for i, n := range f.sectionOrder {
		if n == lower {
			f.sectionOrder = append(f.sectionOrder[:i], f.sectionOrder[i+1:]...)
			break
		}
	}
	return true
}

// Sections returns an iterator over sections in insertion order.
func (f *IniFile) Sections() iter.Seq2[int, *Section] {
	return func(yield func(int, *Section) bool) {
		for i, name := range f.sectionOrder {
			if !yield(i, f.sections[name]) {
				return
			}
		}
	}
}

// String returns the complete PGINI file content with all sections in
// insertion order, separated by blank lines.
func (f *IniFile) String() string {
	var b strings.Builder
	for i, name := range f.sectionOrder {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(f.sections[name].String())
	}
	return b.String()
}

// Section represents a named group of key-value parameters.
// The Name is stored lowercase; an empty Name represents the default section.
type Section struct {
	Name       string
	params     map[string]*Param
	paramOrder []string
}

// NewSection creates a new empty Section with the given name.
// The name is normalized to lowercase. An empty name or "default" refers to
// the default section (stored as empty string).
// It returns an error if name is not a valid PGINI identifier.
func NewSection(name string) (*Section, error) {
	lower := strings.ToLower(name)
	if lower == "default" {
		lower = ""
	}
	if lower != "" && !identifierRe.MatchString(lower) {
		return nil, fmt.Errorf("invalid section name %q: must match [A-Za-z_][A-Za-z0-9_.\\-]*", name)
	}
	return &Section{
		Name:   lower,
		params: make(map[string]*Param),
	}, nil
}

// String returns the PGINI representation of the section, including the
// section header (unless it is the default section) followed by all parameters.
func (s *Section) String() string {
	var b strings.Builder
	if s.Name != "" {
		fmt.Fprintf(&b, "[%s]\n", s.Name)
	}
	for _, key := range s.paramOrder {
		fmt.Fprintln(&b, s.params[key].String())
	}
	return b.String()
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

// String returns the PGINI representation of the parameter as "key = value".
// Values that match safe-char+ (booleans, integers, floats, simple identifiers)
// are written unquoted. All other values are single-quoted with internal single
// quotes escaped as \' and backslashes as \\.
func (p *Param) String() string {
	if unquotedValueRe.MatchString(p.Value) {
		return fmt.Sprintf("%s = %s", p.Name, p.Value)
	}
	escaped := strings.ReplaceAll(p.Value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	return fmt.Sprintf("%s = '%s'", p.Name, escaped)
}

// NewParam creates a new Param with the given name and value.
// The name is normalized to lowercase per the PGINI spec.
// It returns an error if name is not a valid PGINI identifier.
func NewParam(name string, value string) (*Param, error) {
	lower := strings.ToLower(name)
	if !identifierRe.MatchString(lower) {
		return nil, fmt.Errorf("invalid parameter key %q: must match [A-Za-z_][A-Za-z0-9_.\\-]*", name)
	}
	return &Param{
		Name:  lower,
		Value: value,
	}, nil
}
