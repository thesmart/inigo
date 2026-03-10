// Marshaling PGINI intermediates into spec-compliant INI text.

package pgini

import (
	"fmt"
	"regexp"
	"strings"
)

// unquotedValueRe matches values that can appear unquoted in PGINI output.
// Per the grammar: unquoted-value ::= safe-char+ where safe-char is
// letter | digit | [_.\-]. This covers booleans (true, false, on, off,
// yes, no, 1, 0), integers (100, 0xFF, 077), and floats (1.5, 0.001).
var unquotedValueRe = regexp.MustCompile(`^[a-zA-Z0-9_.\-]+$`)

// MarshalText implements encoding.TextMarshaler. It produces a complete
// PGINI document with all sections in insertion order, separated by blank lines.
func (f *IniFile) MarshalText() ([]byte, error) {
	var b strings.Builder
	for i, name := range f.sectionOrder {
		if i > 0 {
			b.WriteString("\n")
		}
		text, err := f.sections[name].MarshalText()
		if err != nil {
			return nil, err
		}
		b.Write(text)
	}
	return []byte(b.String()), nil
}

// MarshalText implements encoding.TextMarshaler. It produces a PGINI section
// block: the section header (omitted for the default section) followed by
// all parameters in insertion order.
func (s *Section) MarshalText() ([]byte, error) {
	var b strings.Builder
	if s.Name != "" {
		fmt.Fprintf(&b, "[%s]\n", s.Name)
	}
	for _, key := range s.paramOrder {
		text, err := s.params[key].MarshalText()
		if err != nil {
			return nil, err
		}
		b.Write(text)
		b.WriteByte('\n')
	}
	return []byte(b.String()), nil
}

// MarshalText implements encoding.TextMarshaler. It produces a single PGINI
// parameter line: "key = value". Values matching safe-char+ (booleans,
// integers, floats, simple identifiers) are written unquoted. All other
// values are single-quoted with \' and \\ escaping.
func (p *Param) MarshalText() ([]byte, error) {
	if unquotedValueRe.MatchString(p.Value) {
		return fmt.Appendf(nil, "%s = %s", p.Name, p.Value), nil
	}
	escaped := strings.ReplaceAll(p.Value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	return fmt.Appendf(nil, "%s = '%s'", p.Name, escaped), nil
}
