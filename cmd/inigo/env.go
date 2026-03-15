package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/thesmart/inigo/pgini"
)

var envCmd = &cobra.Command{
	Use:   "env [flags] <ini-file> [section] -- <command> [args...]",
	Short: "Load INI params as env vars and exec a command",
	Long: `Load parameters from an INI file section, export them as uppercase
environment variables, and exec a command with those variables set.

If no section is given, the default (unnamed) section is used.`,
	Example: `  # Connect to PostgreSQL using a .env file
  inigo env .env -- psql

  # Use a named section from pg_service.conf
  inigo env pg_service.conf mydb -- psql

  # Use in a shell script
  #!/bin/sh
  exec inigo env /etc/myapp.conf -- ./start-server`,
	Args: cobra.MinimumNArgs(1),
	RunE: runEnv,
}

func runEnv(cmd *cobra.Command, args []string) error {
	iniFile, section, command, err := splitDashArgs(cmd, args)
	if err != nil {
		return err
	}

	if len(command) == 0 {
		return fmt.Errorf("missing command after --")
	}

	cfg, err := pgini.Parse(iniFile)
	if err != nil {
		return err
	}

	sec := cfg.GetSection(section)
	if sec == nil {
		return fmt.Errorf("section %q not found in %s", section, iniFile)
	}

	envVars := buildEnvVars(sec)
	env := mergeEnv(os.Environ(), envVars)

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

// splitDashArgs uses Cobra's ArgsLenAtDash to split args into positional
// (ini-file, optional section) and command (after --).
func splitDashArgs(cmd *cobra.Command, args []string) (iniFile, section string, command []string, err error) {
	dashAt := cmd.ArgsLenAtDash()

	var positional []string
	if dashAt < 0 {
		positional = args
	} else {
		positional = args[:dashAt]
		command = args[dashAt:]
	}

	switch len(positional) {
	case 1:
		iniFile = positional[0]
	case 2:
		iniFile = positional[0]
		section = positional[1]
	default:
		return "", "", nil, fmt.Errorf("expected <ini-file> [section], got %d argument(s)", len(positional))
	}

	return iniFile, section, command, nil
}

func buildEnvVars(sec *pgini.Section) []string {
	var env []string
	for _, param := range sec.Params() {
		envName := strings.ToUpper(param.Name)
		env = append(env, envName+"="+param.Value)
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
