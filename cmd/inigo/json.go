package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/stoewer/go-strcase"
	"github.com/thesmart/inigo/pgini"
)

var jsonCase string

var jsonCmd = &cobra.Command{
	Use:   "json [flags] <ini-file> [section] [-- <command> [args...]]",
	Short: "Output INI params as JSON",
	Long: `Load parameters from an INI file section and output them as a JSON object.

If no section is given, the default (unnamed) section is used.
If a command is given after --, exec it with the JSON string set as
the INIGO_JSON environment variable.`,
	Example: `  # Output config as JSON to stdout
  inigo json pg_service.conf mydb

  # Pretty-print with jq
  inigo json config.ini mydb | jq .

  # Save config snapshot to a file
  inigo json config.ini mydb > config.json

  # Pass JSON config to an app via env var
  inigo json pg_service.conf mydb -- ./myapp

  # Use in a shell script with command substitution
  DB_JSON=$(inigo json pg_service.conf mydb)

  # Convert keys to camelCase
  inigo json --case camelCase pg_service.conf mydb

  # Convert keys to UPPER_SNAKE_CASE
  inigo json --case UPPER_CASE pg_service.conf mydb

  # Also accepts short names
  inigo json --case snake pg_service.conf mydb`,
	Args: cobra.MinimumNArgs(1),
	RunE: runJSON,
}

func init() {
	jsonCmd.Flags().StringVarP(&jsonCase, "case", "c", "",
		"convert JSON key casing by name or example (e.g. snake_case, camelCase, UPPER_CASE, kebab-case, PascalCase)")
}

func runJSON(cmd *cobra.Command, args []string) error {
	iniFile, section, command, err := splitDashArgs(cmd, args)
	if err != nil {
		return err
	}

	cfg, err := pgini.Parse(iniFile)
	if err != nil {
		return err
	}

	sec := cfg.GetSection(section)
	if sec == nil {
		return fmt.Errorf("section %q not found in %s", section, iniFile)
	}

	convertKey, err := keyCaseFunc(jsonCase)
	if err != nil {
		return err
	}

	params := make(map[string]string)
	for _, param := range sec.Params() {
		params[convertKey(param.Name)] = param.Value
	}

	jsonBytes, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	if len(command) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), string(jsonBytes))
		return nil
	}

	env := mergeEnv(os.Environ(), []string{"INIGO_JSON=" + string(jsonBytes)})

	binary, err := exec.LookPath(command[0])
	if err != nil {
		if !silent {
			fmt.Fprintf(os.Stderr, "inigo: %s: command not found\n", command[0])
		}
		os.Exit(127)
	}

	if err := syscall.Exec(binary, command, env); err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	return nil
}

// keyCaseFunc returns a string transform for the given --case value.
// It accepts short names (snake, camel, etc.) or literal examples
// whose shape is detected (snake_case, camelCase, UPPER-CASE, etc.).
// An empty value returns an identity function (no conversion).
func keyCaseFunc(name string) (func(string) string, error) {
	if name == "" {
		return func(s string) string { return s }, nil
	}

	// Named aliases (fast path).
	switch strings.ToLower(name) {
	case "lower", "down":
		return strings.ToLower, nil
	case "upper", "up":
		return strings.ToUpper, nil
	case "snake":
		return strcase.SnakeCase, nil
	case "kebab":
		return strcase.KebabCase, nil
	case "camel":
		return strcase.LowerCamelCase, nil
	case "pascal":
		return strcase.UpperCamelCase, nil
	}

	// Example-based detection: infer from the shape of the value.
	if fn, ok := detectCaseExample(name); ok {
		return fn, nil
	}

	return nil, fmt.Errorf("unknown --case value %q (try: snake_case, camelCase, UPPER_CASE, kebab-case, PascalCase, UPPER-CASE)", name)
}

// detectCaseExample analyzes a literal example string and returns
// the matching case-conversion function.
func detectCaseExample(example string) (func(string) string, bool) {
	hasUnderscore := strings.Contains(example, "_")
	hasHyphen := strings.Contains(example, "-")

	// Mixed separators are ambiguous.
	if hasUnderscore && hasHyphen {
		return nil, false
	}
	hasCamelBoundary := false
	allUpper := true
	allLower := true

	for i, r := range example {
		if r == '_' || r == '-' {
			continue
		}
		if unicode.IsUpper(r) {
			allLower = false
		}
		if unicode.IsLower(r) {
			allUpper = false
		}
		// Detect lower→upper transition (camel boundary).
		if i > 0 && unicode.IsUpper(r) {
			prev := rune(example[i-1])
			if unicode.IsLower(prev) {
				hasCamelBoundary = true
			}
		}
	}

	startsUpper := len(example) > 0 && unicode.IsUpper(rune(example[0]))

	switch {
	// snake_case
	case hasUnderscore && allLower:
		return strcase.SnakeCase, true
	// UPPER_CASE
	case hasUnderscore && allUpper:
		return strcase.UpperSnakeCase, true
	// kebab-case
	case hasHyphen && allLower:
		return strcase.KebabCase, true
	// UPPER-CASE
	case hasHyphen && allUpper:
		return strcase.UpperKebabCase, true
	// camelCase (starts lower, has upper transition)
	case hasCamelBoundary && !startsUpper:
		return strcase.LowerCamelCase, true
	// PascalCase (starts upper, has lower letters)
	case hasCamelBoundary && startsUpper:
		return strcase.UpperCamelCase, true
	// lowercase (all lower, no separators)
	case allLower && !hasUnderscore && !hasHyphen:
		return strings.ToLower, true
	// UPPERCASE (all upper, no separators)
	case allUpper && !hasUnderscore && !hasHyphen:
		return strings.ToUpper, true
	}

	return nil, false
}
