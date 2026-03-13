// Example 02: Parse a conf file and query it programmatically.
//
// Use Parse to get an *IniFile, then navigate sections and parameters
// using the query API. This is useful when you don't know the schema
// ahead of time or need to inspect the file's structure.

package main

import (
	"fmt"
	"log"

	"github.com/thesmart/inigo/pgini"
)

func main() {
	// Parse reads the file and returns an *IniFile.
	f, err := pgini.Parse("example.conf")
	if err != nil {
		log.Fatal(err)
	}

	// Iterate all sections in insertion order.
	fmt.Println("=== All sections ===")
	for _, section := range f.Sections() {
		name := section.Name
		if name == "" {
			// Iterate parameters within the section.
			for _, param := range section.Params() {
				fmt.Printf("%s = %s\n", param.Name, param.Value)
			}
		} else {
			fmt.Printf("\n[%s]\n", name)
			// Iterate parameters within the section.
			for _, param := range section.Params() {
				fmt.Printf("  %s = %s\n", param.Name, param.Value)
			}
		}
	}

	// Look up a specific value by section and key.
	fmt.Println("\n=== Direct lookup ===")
	db := f.GetSection("database")
	if host, ok := db.GetValue("host"); ok {
		fmt.Printf("database.host = %s\n", host)
	}
	if port, ok := db.GetValue("port"); ok {
		fmt.Printf("database.port = %s\n", port)
	}
}
