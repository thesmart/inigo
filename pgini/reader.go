// Reader implements a PGINI file parser that reads .conf files into IniFile
// structures. It follows the EBNF grammar defined in reference/pgini-agents.md,
// using FileCursor for line iteration and RootCursor for include traversal.
//
// See reference/pgini-agents.md for the specification this package implements.

package pgini

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Load parses the PGINI file at filePath and unmarshals the named section into
// a new instance of T. T must be a struct with `ini:"KEY"` field tags.
// Use an empty string for section to read the default (unnamed) section.
func Load[T any](filePath string, section string) (*T, error) {
	f, err := Parse(filePath)
	if err != nil {
		return nil, err
	}

	var t T
	if err := f.UnmarshalSection(section, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// LoadInto parses the PGINI file at filePath and unmarshals the named section
// into the struct pointed to by structPtr. structPtr must be a pointer to a
// struct with `ini:"KEY"` field tags.
// Use an empty string for section to read the default (unnamed) section.
func LoadInto(filePath string, section string, structPtr any) error {
	f, err := Parse(filePath)
	if err != nil {
		return err
	}
	return f.UnmarshalSection(section, structPtr)
}

// Parse parses the PGINI file at filePath (and any included files) and returns
// a populated IniFile.
func Parse(filePath string) (*IniFile, error) {
	rootCursor, err := NewRootCursor(filePath)
	if err != nil {
		return nil, err
	}

	cursor := rootCursor.NextInclude()
	if cursor == nil {
		return rootCursor.File, nil
	}

	// Start parsing into the default section.
	currentSection := rootCursor.File.GetSection("")
	if err := parseCursor(rootCursor, cursor, &currentSection); err != nil {
		return nil, err
	}

	return rootCursor.File, nil
}

// parseCursor parses a single FileCursor line-by-line, populating the
// RootCursor's IniFile. On include directives it recurses via
// rootCursor.AddInclude + rootCursor.NextInclude to process includes inline,
// preserving "last wins" parameter ordering.
//
// Parameters:
//   - rootCursor: owns the IniFile and tracks visited files for circular detection
//   - cursor: the FileCursor for the current file being parsed
//   - currentSection: pointer to the active section; updated when [section] headers are encountered
func parseCursor(rootCursor *RootCursor, cursor *FileCursor, currentSection **Section) error {
	for line, ok := cursor.NextLine(); ok; line, ok = cursor.NextLine() {
		pos := skipWSP(line, 0)

		// blank line
		if pos >= len(line) {
			continue
		}

		ch := rune(line[pos])

		// comment line
		if isComment(ch) {
			continue
		}

		// section header
		if ch == '[' {
			section, err := parseSectionHeader(cursor, line, pos)
			if err != nil {
				return err
			}
			added, err := rootCursor.File.AddSection(section)
			if err != nil {
				return parseErrf(cursor, pos, "%s", err)
			}
			*currentSection = added
			continue
		}

		// identifier: parameter or include directive
		if isLetter(ch) {
			ident, newPos := scanIdentifier(line, pos)
			directive := strings.ToLower(ident)

			if directive == "include" || directive == "include_if_exists" || directive == "include_dir" {
				if err := parseInclude(rootCursor, cursor, currentSection, line, newPos, directive); err != nil {
					return err
				}
				continue
			}

			// parameter
			if err := parseParameter(cursor, *currentSection, line, ident, pos, newPos); err != nil {
				return err
			}
			continue
		}

		return parseErrf(cursor, pos, "unexpected character %q", ch)
	}
	return nil
}

// parseSectionHeader parses a section header line: [ identifier ] WSP* comment? EOL.
// pos points at the opening '['.
func parseSectionHeader(cursor *FileCursor, line string, pos int) (string, error) {
	pos++ // skip '['
	pos = skipWSP(line, pos)

	if pos >= len(line) || rune(line[pos]) == ']' {
		return "", parseErrf(cursor, pos, "empty section name")
	}

	if !isLetter(rune(line[pos])) {
		return "", parseErrf(cursor, pos, "invalid section name start %q", rune(line[pos]))
	}

	name, pos := scanIdentifier(line, pos)

	pos = skipWSP(line, pos)
	if pos >= len(line) || rune(line[pos]) != ']' {
		return "", parseErrf(cursor, pos, "expected ']' after section name %q", name)
	}
	pos++ // skip ']'

	// After ']', only whitespace and an optional comment are allowed.
	pos = skipWSP(line, pos)
	if pos < len(line) && !isComment(rune(line[pos])) {
		return "", parseErrf(cursor, pos, "unexpected character %q after section header", rune(line[pos]))
	}

	return name, nil
}

// parseParameter parses a parameter line after the key has already been scanned.
// key is the parsed identifier, keyPos is the byte position where the key started,
// and pos is the byte position after the key.
func parseParameter(cursor *FileCursor, section *Section, line string, key string, keyPos int, pos int) error {
	pos = skipWSP(line, pos)

	// Optional separator: '=' or ':'
	if pos < len(line) && (line[pos] == '=' || line[pos] == ':') {
		pos++
		pos = skipWSP(line, pos)
	}

	// Parse value (may be empty if EOL or comment follows).
	var value string
	if pos < len(line) && !isComment(rune(line[pos])) {
		if line[pos] == '\'' {
			var err error
			value, pos, err = scanQuotedValue(cursor, line, pos)
			if err != nil {
				return err
			}
		} else {
			value, pos = scanUnquotedValue(line, pos)
		}
	}

	// After value, only whitespace and optional comment allowed.
	pos = skipWSP(line, pos)
	if pos < len(line) && !isComment(rune(line[pos])) {
		return parseErrf(cursor, pos, "unexpected character %q after value", rune(line[pos]))
	}

	if _, err := section.SetParam(key, value); err != nil {
		return parseErrf(cursor, keyPos, "%s", err)
	}
	return nil
}

// parseInclude handles include, include_if_exists, and include_dir directives.
// pos is the byte position after the directive identifier.
func parseInclude(rootCursor *RootCursor, cursor *FileCursor, currentSection **Section, line string, pos int, directive string) error {
	// Require at least one whitespace after the directive name.
	if pos >= len(line) || !isWSP(rune(line[pos])) {
		return parseErrf(cursor, pos, "%s requires a quoted path argument", directive)
	}
	pos = skipWSP(line, pos)

	// Parse the quoted path.
	if pos >= len(line) || line[pos] != '\'' {
		return parseErrf(cursor, pos, "%s requires a single-quoted path", directive)
	}
	quotedPath, newPos, err := scanQuotedPath(cursor, line, pos)
	if err != nil {
		return err
	}
	pos = newPos

	// After quoted path, only whitespace and optional comment allowed.
	pos = skipWSP(line, pos)
	if pos < len(line) && !isComment(rune(line[pos])) {
		return parseErrf(cursor, pos, "unexpected character %q after %s path", rune(line[pos]), directive)
	}

	if quotedPath == "" {
		return parseErrf(cursor, pos, "%s path must not be empty", directive)
	}

	// Resolve relative paths against the current file's directory.
	resolvedPath := quotedPath
	if !filepath.IsAbs(resolvedPath) {
		resolvedPath = filepath.Join(filepath.Dir(cursor.Path), resolvedPath)
	}

	switch directive {
	case "include":
		return processIncludeFile(rootCursor, cursor, currentSection, resolvedPath, true)
	case "include_if_exists":
		return processIncludeFile(rootCursor, cursor, currentSection, resolvedPath, false)
	case "include_dir":
		return processIncludeDir(rootCursor, cursor, currentSection, resolvedPath)
	}
	return nil
}

// processIncludeFile adds a single include file to the root cursor and
// immediately parses it. If required is false, missing files are silently skipped.
func processIncludeFile(rootCursor *RootCursor, cursor *FileCursor, currentSection **Section, filePath string, required bool) error {
	err := rootCursor.AddInclude(filePath)
	if err != nil {
		if !required && os.IsNotExist(unwrapRootErr(err)) {
			return nil
		}
		return parseErrf(cursor, 0, "%s", err)
	}

	includeCursor := rootCursor.NextInclude()
	if includeCursor == nil {
		return nil
	}
	return parseCursor(rootCursor, includeCursor, currentSection)
}

// processIncludeDir reads all .conf files from a directory (skipping dotfiles),
// sorts them in ascending order, and includes each one.
func processIncludeDir(rootCursor *RootCursor, cursor *FileCursor, currentSection **Section, dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return parseErrf(cursor, 0, "include_dir %q: %s", dirPath, err)
	}

	// Collect .conf files, skip dotfiles.
	var confFiles []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".conf") {
			continue
		}
		confFiles = append(confFiles, filepath.Join(dirPath, name))
	}
	sort.Strings(confFiles)

	for _, confPath := range confFiles {
		if err := processIncludeFile(rootCursor, cursor, currentSection, confPath, true); err != nil {
			return err
		}
	}
	return nil
}

// unwrapRootErr attempts to extract the deepest wrapped error for type checking.
func unwrapRootErr(err error) error {
	for {
		inner := errors.Unwrap(err)
		if inner == nil {
			return err
		}
		err = inner
	}
}

// --- Character classification helpers ---

// isLetter reports whether r is a PGINI letter: [a-zA-Z_].
func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
}

// isDigit reports whether r is a PGINI digit: [0-9].
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// isIdentChar reports whether r is valid in a PGINI identifier: letter | digit.
func isIdentChar(r rune) bool {
	return isLetter(r) || isDigit(r)
}

// isSafeChar reports whether r is a PGINI safe-char for unquoted values:
// letter | digit | [_.\-:/+].
func isSafeChar(r rune) bool {
	return isIdentChar(r) || r == '.' || r == '-' || r == ':' || r == '/' || r == '+'
}

// isWSP reports whether r is PGINI whitespace: space or tab.
func isWSP(r rune) bool {
	return r == ' ' || r == '\t'
}

// isComment reports whether r starts a PGINI comment: # or ;.
func isComment(r rune) bool {
	return r == '#' || r == ';'
}

// --- Scan helpers ---

// skipWSP advances past any spaces and tabs starting at pos, returning the
// new position.
func skipWSP(line string, pos int) int {
	for pos < len(line) && isWSP(rune(line[pos])) {
		pos++
	}
	return pos
}

// scanIdentifier extracts a PGINI identifier starting at pos.
// identifier ::= letter ( letter | digit )*
// Returns the identifier string and the position after the last identifier char.
func scanIdentifier(line string, pos int) (string, int) {
	start := pos
	for pos < len(line) && isIdentChar(rune(line[pos])) {
		pos++
	}
	return line[start:pos], pos
}

// scanQuotedValue extracts and de-escapes a single-quoted PGINI value.
// This is the inverse of pginiEscape (ini_file.go:310-342).
// pos must point at the opening single quote.
// Returns the de-escaped value, the position after the closing quote, and any error.
func scanQuotedValue(cursor *FileCursor, line string, pos int) (string, int, error) {
	if pos >= len(line) || line[pos] != '\'' {
		return "", pos, parseErrf(cursor, pos, "expected opening single quote")
	}
	pos++ // skip opening quote

	var b strings.Builder
	for pos < len(line) {
		ch := line[pos]

		// Closing quote
		if ch == '\'' {
			// Check for doubled quote: '' → literal single quote
			if pos+1 < len(line) && line[pos+1] == '\'' {
				b.WriteByte('\'')
				pos += 2
				continue
			}
			// End of quoted value
			pos++
			return b.String(), pos, nil
		}

		// Backslash escape sequence
		if ch == '\\' {
			if pos+1 >= len(line) {
				return "", pos, parseErrf(cursor, pos, "unterminated escape sequence at end of line")
			}
			pos++
			escaped := line[pos]
			switch escaped {
			case '\\':
				b.WriteByte('\\')
			case '\'':
				b.WriteByte('\'')
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '0', '1', '2', '3', '4', '5', '6', '7':
				// Octal escape: 1–3 octal digits
				var octVal byte
				digits := 0
				for digits < 3 && pos < len(line) && line[pos] >= '0' && line[pos] <= '7' {
					octVal = (octVal << 3) + (line[pos] - '0')
					pos++
					digits++
				}
				b.WriteByte(octVal)
				continue // pos already advanced past the octal digits
			default:
				return "", pos - 1, parseErrf(cursor, pos-1, "invalid escape sequence '\\%c'", escaped)
			}
			pos++
			continue
		}

		// Regular character (including UTF-8 multi-byte)
		b.WriteByte(ch)
		pos++
	}

	// Reached end of line without closing quote.
	return "", pos, parseErrf(cursor, pos, "unterminated quoted value")
}

// scanUnquotedValue extracts an unquoted PGINI value: safe-char+.
// safe-char ::= letter | digit | [_.\-:/+]
// Returns the value and the position after the last safe char.
func scanUnquotedValue(line string, pos int) (string, int) {
	start := pos
	for pos < len(line) && isSafeChar(rune(line[pos])) {
		pos++
	}
	return line[start:pos], pos
}

// scanQuotedPath extracts a single-quoted path without escape processing.
// quoted-path ::= "'" (abs-path | rel-path) "'"
// segment-char ::= [^#x00-#x1F #x27 #x7F /] (everything except control chars, single quote, and /)
// pos must point at the opening single quote.
func scanQuotedPath(cursor *FileCursor, line string, pos int) (string, int, error) {
	if pos >= len(line) || line[pos] != '\'' {
		return "", pos, parseErrf(cursor, pos, "expected opening single quote for path")
	}
	pos++ // skip opening quote

	start := pos
	for pos < len(line) {
		ch := line[pos]
		if ch == '\'' {
			path := line[start:pos]
			pos++ // skip closing quote
			return path, pos, nil
		}
		// Reject control characters per the grammar.
		if ch <= 0x1F || ch == 0x7F {
			return "", pos, parseErrf(cursor, pos, "invalid control character in path at position %d", pos)
		}
		pos++
	}

	return "", pos, parseErrf(cursor, pos, "unterminated quoted path")
}

// parseErrf formats a parse error with file path, line number, and column.
// Line and column are 1-indexed in the output.
func parseErrf(cursor *FileCursor, col int, format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s:%d:%d: %s", cursor.Path, cursor.lineOffset+1, col+1, msg)
}
