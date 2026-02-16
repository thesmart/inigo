package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/thesmart/inigo"
)

const usageText = `Usage: inigo [flags] <ini-file> <section> -- <command> [args...]

Load INI config params as environment variables and exec a command.

Flags:
  -p, --prefix PREFIX   Prepend PREFIX to env var names (e.g. --prefix PG)
  -h, --help            Show this help message

Params from the INI section are converted to uppercase environment
variables. With --prefix PG, param "host" becomes PGHOST.

Example:
  inigo --prefix PG pg_service.conf mydb -- psql
`

type args struct {
	prefix  string
	iniFile string
	section string
	command []string
}

func main() {
	a, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "inigo: %v\n", err)
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(2)
	}

	cfg, err := inigo.Load(a.iniFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "inigo: %v\n", err)
		os.Exit(1)
	}

	sec := cfg.Section(a.section)
	if sec == nil {
		fmt.Fprintf(os.Stderr, "inigo: section %q not found in %s\n", a.section, a.iniFile)
		os.Exit(1)
	}

	env := mergeEnv(os.Environ(), buildEnv(sec, a.prefix))

	binary, err := exec.LookPath(a.command[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "inigo: %s: command not found\n", a.command[0])
		os.Exit(127)
	}

	err = syscall.Exec(binary, a.command, env)
	fmt.Fprintf(os.Stderr, "inigo: exec: %v\n", err)
	os.Exit(126)
}

func parseArgs(argv []string) (args, error) {
	var a args

	// Check for --help before requiring "--" separator
	for _, arg := range argv {
		if arg == "-h" || arg == "--help" {
			fmt.Print(usageText)
			os.Exit(0)
		}
	}

	// Find "--" separator
	dashIdx := -1
	for i, arg := range argv {
		if arg == "--" {
			dashIdx = i
			break
		}
	}

	if dashIdx < 0 {
		return a, fmt.Errorf("missing -- separator before command")
	}

	a.command = argv[dashIdx+1:]
	if len(a.command) == 0 {
		return a, fmt.Errorf("missing command after --")
	}

	// Parse flags and positional args before "--"
	pre := argv[:dashIdx]
	var positional []string

	for i := 0; i < len(pre); i++ {
		switch pre[i] {
		case "-p", "--prefix":
			if i+1 >= len(pre) {
				return a, fmt.Errorf("--prefix requires a value")
			}
			i++
			a.prefix = strings.ToUpper(pre[i])
		default:
			if strings.HasPrefix(pre[i], "-") {
				return a, fmt.Errorf("unknown flag: %s", pre[i])
			}
			positional = append(positional, pre[i])
		}
	}

	if len(positional) != 2 {
		return a, fmt.Errorf("expected <ini-file> <section>, got %d argument(s)", len(positional))
	}

	a.iniFile = positional[0]
	a.section = positional[1]
	return a, nil
}

func buildEnv(sec *inigo.Section, prefix string) []string {
	params := sec.AllParams()
	env := make([]string, 0, len(params))
	for _, name := range params {
		envName := prefix + strings.ToUpper(name)
		envVal := sec.GetParam(name).String()
		env = append(env, envName+"="+envVal)
	}
	return env
}

func mergeEnv(current, overlay []string) []string {
	seen := make(map[string]int, len(current))
	result := make([]string, 0, len(current)+len(overlay))

	for _, entry := range current {
		key, _, _ := strings.Cut(entry, "=")
		seen[key] = len(result)
		result = append(result, entry)
	}

	for _, entry := range overlay {
		key, _, _ := strings.Cut(entry, "=")
		if idx, ok := seen[key]; ok {
			result[idx] = entry
		} else {
			seen[key] = len(result)
			result = append(result, entry)
		}
	}

	return result
}
