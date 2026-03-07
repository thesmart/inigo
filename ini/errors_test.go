package ini

import (
	"strings"
	"testing"
)

func TestUnmarshalIniFileIntermediate_Errors(t *testing.T) {
	tests := []struct {
		file     string
		wantErrs []string // all substrings must appear in the error
	}{
		// Section errors
		{"section_empty_brackets.conf", []string{"<section>", "expected identifier after '['"}},
		{"section_unclosed.conf", []string{"<section>", "expected ']'"}},
		{"section_space_in_name.conf", []string{"<section>", "expected ']'"}},
		{"section_starts_with_digit.conf", []string{"<section>", "expected identifier after '['"}},
		{"section_trailing_garbage.conf", []string{"<section>", "unexpected characters after section header"}},

		// Key / identifier errors
		{"key_starts_with_digit.conf", []string{"<key>", "expected identifier"}},
		{"key_starts_with_dash.conf", []string{"<key>", "expected identifier"}},
		{"key_starts_with_equals.conf", []string{"<key>", "expected identifier"}},
		{"line_starts_with_special.conf", []string{"<key>", "expected identifier"}},
		{"line_starts_with_bang.conf", []string{"<key>", "expected identifier"}},

		// Quoting errors
		{"unterminated_quote.conf", []string{"<quoted-value>", "unterminated single-quoted string"}},
		{"unterminated_quote_with_escape.conf", []string{"<quoted-value>", "unterminated single-quoted string"}},
		{"trailing_garbage_after_value.conf", []string{"<parameter>", "unexpected characters after value"}},

		// Include errors
		{"include_empty_path.conf", []string{"<include>", "empty path for include directive"}},
		{"include_file_not_found.conf", []string{"<include>", "file not found"}},
		{"include_path_is_directory.conf", []string{"<include>", "path is a directory"}},
		{"include_trailing_garbage.conf", []string{"<include>", "unexpected characters after include path"}},
		{"include_dir_not_found.conf", []string{"<include_dir>", "failed to read directory"}},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			_, err := unmarshalIniFileIntermediate("testdata/errors/" + tt.file)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			errMsg := err.Error()
			for _, want := range tt.wantErrs {
				if !strings.Contains(errMsg, want) {
					t.Errorf("error %q does not contain %q", errMsg, want)
				}
			}
		})
	}
}
