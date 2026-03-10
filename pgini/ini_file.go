// Defines the parsed, intermediate data representation of an INI configuration
// file that follows the PGINI standard.
//
// See reference/pgini-agents.md for the specification this package implements.
// This is the root PGINI implementation: other source files in the package
// should change to match it, but this file should not change to match others.

package pgini

import (
	"fmt"
	"iter"
	"path"
	"strings"
)

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

// String returns a human-readable summary of the IniFile.
func (f *IniFile) String() string {
	return fmt.Sprintf("IniFile(%q, %d sections)", f.Name, len(f.sectionOrder))
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

// Params returns an iterator over parameters in insertion order.
func (s *Section) Params() iter.Seq2[int, *Param] {
	return func(yield func(int, *Param) bool) {
		for i, name := range s.paramOrder {
			if !yield(i, s.params[name]) {
				return
			}
		}
	}
}

// String returns a human-readable summary of the Section.
func (s *Section) String() string {
	name := s.Name
	if name == "" {
		name = "(default)"
	}
	return fmt.Sprintf("Section(%q, %d params)", name, len(s.paramOrder))
}

// Param represents a single parameter with its raw string value.
type Param struct {
	Name  string
	Value string
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

// String returns a human-readable summary of the Param.
func (p *Param) String() string {
	return fmt.Sprintf("Param(%q, %q)", p.Name, p.Value)
}
