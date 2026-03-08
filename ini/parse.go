package ini

import (
	"fmt"
	"strings"
)

// lineKind identifies what type of line was parsed.
type lineKind int

const (
	lineBlank     lineKind = iota
	lineComment   lineKind = iota
	lineSection   lineKind = iota
	lineParameter lineKind = iota
	lineInclude   lineKind = iota
)

// lineResult holds the parsed result of a single line.
type lineResult struct {
	kind        lineKind
	section     string // for lineSection
	param       *Param // for lineParameter
	includeKind string // for lineInclude: "include", "include_if_exists", "include_dir"
	includePath string // for lineInclude
}

// parseLine parses a single line of an ini file according to the BNF grammar.
func parseLine(line string, fc *FileCursor) (*lineResult, error) {
	i := 0

	// skip leading whitespace
	i = skipWhitespace(line, i)

	// blank line
	if i >= len(line) {
		return &lineResult{kind: lineBlank}, nil
	}

	ch := line[i]

	// comment line
	if ch == '#' || ch == ';' {
		return &lineResult{kind: lineComment}, nil
	}

	// section header
	if ch == '[' {
		return parseSectionLine(line, i, fc)
	}

	// check for include directive or parameter
	return parseDirectiveOrParam(line, i, fc)
}

// parseSectionLine parses a "[section_name]" line.
func parseSectionLine(line string, start int, fc *FileCursor) (*lineResult, error) {
	i := start + 1 // skip '['

	// parse identifier for section name
	nameStart := i
	if i >= len(line) || !isIdentStart(line[i]) {
		return nil, parseError(fc, start, "section", "expected identifier after '['")
	}
	i++
	for i < len(line) && isIdentContinue(line[i]) {
		i++
	}
	name := line[nameStart:i]

	// expect ']'
	if i >= len(line) || line[i] != ']' {
		return nil, parseError(fc, i, "section", "expected ']'")
	}
	i++

	// optional trailing whitespace and comment
	i = skipWhitespace(line, i)
	if i < len(line) && line[i] != '#' && line[i] != ';' {
		return nil, parseError(fc, i, "section", "unexpected characters after section header")
	}

	return &lineResult{kind: lineSection, section: name}, nil
}

// parseDirectiveOrParam parses either an include directive or a key=value parameter.
func parseDirectiveOrParam(line string, start int, fc *FileCursor) (*lineResult, error) {
	// Try to match include directives first
	// Check longest prefixes first to avoid prefix collision
	rest := line[start:]
	lower := strings.ToLower(rest)

	for _, directive := range []string{"include_if_exists", "include_dir", "include"} {
		if !strings.HasPrefix(lower, directive) {
			continue
		}
		afterDir := start + len(directive)
		if afterDir >= len(line) {
			// Bare keyword at end of line — treat as parameter
			break
		}
		nextCh := line[afterDir]
		// If followed by = or :, it's a parameter (e.g. include = 'something')
		if nextCh == '=' || nextCh == ':' {
			break
		}
		// If followed by an ident character, it's a parameter key starting with "include"
		if isIdentContinue(nextCh) {
			break
		}
		// Directive must be followed by whitespace
		if nextCh == ' ' || nextCh == '\t' {
			// Look ahead past whitespace: if next non-ws char is = or :, it's a parameter
			j := skipWhitespace(line, afterDir)
			if j < len(line) && (line[j] == '=' || line[j] == ':') {
				break
			}
			return parseIncludeLine(line, afterDir, directive, fc)
		}
	}

	return parseParamLine(line, start, fc)
}

// parseIncludeLine parses the value portion of an include directive.
func parseIncludeLine(line string, afterDir int, directive string, fc *FileCursor) (*lineResult, error) {
	i := skipWhitespace(line, afterDir)

	if i >= len(line) {
		return nil, parseError(fc, i, "include", fmt.Sprintf("missing path for %s directive", directive))
	}

	// Parse value (quoted or unquoted)
	value, newI, err := parseValueAt(line, i, fc)
	if err != nil {
		return nil, err
	}
	i = newI

	if value == "" {
		return nil, parseError(fc, i, "include", fmt.Sprintf("empty path for %s directive", directive))
	}

	// optional trailing whitespace and comment
	i = skipWhitespace(line, i)
	if i < len(line) && line[i] != '#' && line[i] != ';' {
		return nil, parseError(fc, i, "include", "unexpected characters after include path")
	}

	return &lineResult{
		kind:        lineInclude,
		includeKind: directive,
		includePath: value,
	}, nil
}

// parseParamLine parses a "key = value" line.
func parseParamLine(line string, start int, fc *FileCursor) (*lineResult, error) {
	i := start

	// parse key (identifier)
	if !isIdentStart(line[i]) {
		return nil, parseError(fc, i, "key", "expected identifier")
	}
	keyStart := i
	i++
	for i < len(line) && isIdentContinue(line[i]) {
		i++
	}
	key := line[keyStart:i]

	// skip whitespace
	i = skipWhitespace(line, i)

	// optional separator
	if i < len(line) && (line[i] == '=' || line[i] == ':') {
		i++
	}

	// skip whitespace
	i = skipWhitespace(line, i)

	// parse value (may be empty)
	var value string
	if i < len(line) && line[i] != '#' && line[i] != ';' {
		var err error
		value, i, err = parseValueAt(line, i, fc)
		if err != nil {
			return nil, err
		}
	}

	// trim trailing whitespace from unquoted values
	value = strings.TrimRight(value, " \t")

	// skip trailing whitespace and optional inline comment
	i = skipWhitespace(line, i)
	if i < len(line) && line[i] != '#' && line[i] != ';' {
		return nil, parseError(fc, i, "parameter", "unexpected characters after value")
	}

	return &lineResult{
		kind: lineParameter,
		param: &Param{
			Name:  key,
			Value: value,
			cursor: Cursor{
				Line:   fc.Line,
				Offset: int32(keyStart),
			},
		},
	}, nil
}

// parseValueAt parses a value starting at position i. Returns the value and the new position.
func parseValueAt(line string, i int, fc *FileCursor) (string, int, error) {
	if i >= len(line) {
		return "", i, nil
	}

	if line[i] == '\'' {
		return parseQuotedValue(line, i, fc)
	}

	val, newI := parseUnquotedValue(line, i)
	return val, newI, nil
}

// parseQuotedValue parses a single-quoted value: 'value with ”escapes” and \'too\\'
func parseQuotedValue(line string, start int, fc *FileCursor) (string, int, error) {
	var buf strings.Builder
	i := start + 1 // skip opening quote

	for i < len(line) {
		ch := line[i]
		switch {
		case ch == '\\' && i+1 < len(line) && line[i+1] == '\'':
			// backslash-escaped single quote
			buf.WriteByte('\'')
			i += 2
		case ch == '\\' && i+1 < len(line) && line[i+1] == '\\':
			// backslash-escaped backslash
			buf.WriteByte('\\')
			i += 2
		case ch == '\'':
			if i+1 < len(line) && line[i+1] == '\'' {
				// doubled single quote escape
				buf.WriteByte('\'')
				i += 2
			} else {
				// closing quote
				return buf.String(), i + 1, nil
			}
		default:
			buf.WriteByte(ch)
			i++
		}
	}

	return "", i, parseError(fc, start, "quoted-value", "unterminated single-quoted string")
}

// parseUnquotedValue parses an unquoted value up to an inline comment or end of line.
func parseUnquotedValue(line string, start int) (string, int) {
	// Unquoted value: read until # or ; (inline comment) or end of line
	// But we need to be careful: semicolons in values are ambiguous per spec
	i := start
	for i < len(line) {
		ch := line[i]
		if ch == '#' {
			break
		}
		// Semicolons as comment markers for unquoted values
		if ch == ';' {
			break
		}
		i++
	}
	return line[start:i], i
}

// isIdentStart checks if a byte can start an identifier: [A-Za-z_]
func isIdentStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

// isIdentContinue checks if a byte can continue an identifier: [A-Za-z0-9_\-.$]
func isIdentContinue(ch byte) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9') || ch == '-' || ch == '.' || ch == '$'
}

// skipWhitespace advances past spaces and tabs.
func skipWhitespace(line string, i int) int {
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	return i
}

// parseError creates a formatted parse error with position information.
func parseError(fc *FileCursor, offset int, node string, msg string) error {
	return fmt.Errorf("%s:%d:%d: <%s> %s", fc.Path, fc.Line, offset, node, msg)
}
