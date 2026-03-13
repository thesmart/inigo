// Example 01: Load a conf file directly into a Go struct.
//
// This is the simplest way to use pgini. Define a struct with `ini` tags
// and call Load[T] to parse the file and populate the struct in one step.

package main

import (
	"fmt"
	"log"

	"github.com/thesmart/inigo/pgini"
)

// AppConfig maps conf keys to struct fields via `ini` tags.
// Fields without an `ini` tag are ignored.
type AppConfig struct {
	Name  string `ini:"name"`
	Host  string `ini:"host"`
	Port  int    `ini:"port"`
	Debug bool   `ini:"debug"`
	Other string // no tag — ignored by pgini
}

func main() {
	// Load parses example.conf and unmarshals the default section into AppConfig.
	cfg, err := pgini.Load[AppConfig]("example.conf", "")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Name:  %s\n", cfg.Name)
	fmt.Printf("Host:  %s\n", cfg.Host)
	fmt.Printf("Port:  %d\n", cfg.Port)
	fmt.Printf("Debug: %t\n", cfg.Debug)
	fmt.Printf("Other: %q (empty — no ini tag)\n", cfg.Other)
}
