// Example 03: Build a conf file from Go structs.
//
// Create an IniFile from scratch, populate it using MarshalSection,
// and serialize it to PGINI format with MarshalIni.

package main

import (
	"fmt"
	"log"

	"github.com/thesmart/inigo/pgini"
)

type ServerConfig struct {
	Host string `ini:"host"`
	Port int    `ini:"port"`
}

type DatabaseConfig struct {
	Host string `ini:"host"`
	Port int    `ini:"port"`
	Name string `ini:"name"`
	User string `ini:"user"`
}

func main() {
	// Create an empty IniFile.
	f, err := pgini.NewIniFile("output.conf")
	if err != nil {
		log.Fatal(err)
	}

	// MarshalSection encodes a struct into the named section.
	server := ServerConfig{Host: "0.0.0.0", Port: 8080}
	if err := f.MarshalSection("server", &server); err != nil {
		log.Fatal(err)
	}

	db := DatabaseConfig{Host: "localhost", Port: 5432, Name: "myapp", User: "postgres"}
	if err := f.MarshalSection("database", &db); err != nil {
		log.Fatal(err)
	}

	// MarshalIni serializes the entire IniFile to PGINI bytes.
	data, err := f.MarshalIni()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Generated conf ===")
	fmt.Println(string(data))
}
