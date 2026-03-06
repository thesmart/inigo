package ini

import "fmt"

// Cursor tracks a parser position within a file.
type Cursor struct {
	Line   int32
	Offset int32
}

// String returns a human-readable position string.
func (c Cursor) String() string {
	return fmt.Sprintf("line %d, offset %d", c.Line, c.Offset)
}

// FileCursor tracks a parser position within a specific file.
type FileCursor struct {
	Cursor
	Path     string
	Contents string
}

// String returns a human-readable position string including the file path.
func (fc *FileCursor) String() string {
	return fmt.Sprintf("%s:%d:%d", fc.Path, fc.Line, fc.Offset)
}
