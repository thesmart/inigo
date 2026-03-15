// Example 04: Custom marshal and unmarshal methods.
//
// For types that aren't primitives (like time.Duration or []string),
// define Marshal<FieldName> and Unmarshal<FieldName> methods on the struct.
// pgini calls these automatically during MarshalSection and UnmarshalSection.

package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/thesmart/inigo/pgini"
)

type Config struct {
	Timeout time.Duration `ini:"timeout"`
	Tags    []string      `ini:"tags"`
}

// UnmarshalTimeout parses a Go duration string (e.g. "30s") into time.Duration.
func (c *Config) UnmarshalTimeout(value string) (*time.Duration, error) {
	d, err := time.ParseDuration(value)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// MarshalTimeout formats time.Duration back to a string.
func (c *Config) MarshalTimeout(value *time.Duration) (string, error) {
	return value.String(), nil
}

// UnmarshalTags splits a comma-separated string into a slice.
func (c *Config) UnmarshalTags(value string) (*[]string, error) {
	parts := strings.Split(value, ",")
	return &parts, nil
}

// MarshalTags joins a slice back into a comma-separated string.
func (c *Config) MarshalTags(value *[]string) (string, error) {
	return strings.Join(*value, ","), nil
}

func main() {
	// Load with custom unmarshalers.
	cfg, err := pgini.Load[Config]("example.conf", "")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Loaded ===")
	fmt.Printf("Timeout: %s\n", cfg.Timeout)
	fmt.Printf("Tags:    %v\n", cfg.Tags)

	// Round-trip: marshal back to a new IniFile.
	f, err := pgini.NewIniFile("roundtrip.conf")
	if err != nil {
		log.Fatal(err)
	}
	if err := f.MarshalSection("", cfg); err != nil {
		log.Fatal(err)
	}

	data, err := f.MarshalIni()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n=== Round-tripped conf ===")
	fmt.Println(string(data))
}
