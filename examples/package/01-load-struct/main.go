// Example 01: Load a conf file directly into a Go struct.
//
// This is the simplest way to use pgini. Define a struct with `ini` tags
// and call Load[T] to parse the file and populate the struct in one step.
// You can also use LoadInto to unmarshal into an existing struct instance.

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

	fmt.Println("=== Load[T] ===")
	fmt.Printf("Name:  %s\n", cfg.Name)
	fmt.Printf("Host:  %s\n", cfg.Host)
	fmt.Printf("Port:  %d\n", cfg.Port)
	fmt.Printf("Debug: %t\n", cfg.Debug)
	fmt.Printf("Other: %q (empty — no ini tag)\n", cfg.Other)

	// LoadInto unmarshals into an existing struct, useful for pre-populating defaults.
	cfg2 := &AppConfig{Host: "0.0.0.0", Port: 3000, Other: "preserved"}
	if err := pgini.LoadInto("example.conf", "", cfg2); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n=== LoadInto (with defaults) ===")
	fmt.Printf("Name:  %s\n", cfg2.Name)
	fmt.Printf("Host:  %s\n", cfg2.Host)
	fmt.Printf("Port:  %d\n", cfg2.Port)
	fmt.Printf("Debug: %t\n", cfg2.Debug)
	fmt.Printf("Other: %q (preserved — no ini tag)\n", cfg2.Other)
}
