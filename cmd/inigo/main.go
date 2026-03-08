package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var silent bool

var rootCmd = &cobra.Command{
	Use:   "inigo",
	Short: "Load INI config params and exec commands",
	Long: `Feeds configuration from a file into other tools like psql, curl, or your
own apps — without needing parsing in those apps.`,
	Example: `  # Connect to PostgreSQL using a .env file
  inigo env .env -- psql

  # Dump config as JSON for use in a shell script
  inigo json config.ini mydb | jq .

  # Pass filtered env vars to a Docker container
  inigo env --filter PG .env -- docker run --env-file /dev/stdin myimage

  # Use in a shell script
  #!/bin/sh
  exec inigo env /etc/myapp.conf -- ./myapp`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&silent, "silent", "s", false, "suppress error messages on stderr")
	rootCmd.AddCommand(envCmd)
	rootCmd.AddCommand(jsonCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		if !silent {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
