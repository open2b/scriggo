package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/native"
)

// Native packages.
var packages native.Packages

func _main() {

	// Make a file system from the program path passed as command line argument.
	fsys := makeFileSystemFromArgument()

	// Build the program.
	program, err := scriggo.Build(fsys, &scriggo.BuildOptions{
		AllowGoStmt: true,     // Allows the go statement.
		Packages:    packages, // Native packages that can be imported with the import statement.
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	// Run the program.
	err = program.Run(nil)
	if err != nil {
		if err, ok := err.(*scriggo.PanicError); ok {
			panic(err)
		}
		// Another error occurred.
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

}

// makeFileSystemFromArgument makes a file system from the program path passed
// as command line argument.
//
// If an error occurs, it prints the error and exits.
func makeFileSystemFromArgument() fs.FS {

	// Validate command line arguments.
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintf(os.Stderr, "usage: %s program.go", os.Args[0])
		os.Exit(1)
	}
	file := os.Args[1]
	name := filepath.Base(file)
	if name == ".go" {
		fmt.Printf("%s: invalid file name", name)
		os.Exit(1)
	}
	ext := filepath.Ext(name)
	if ext != ".go" {
		fmt.Printf("%s: extension must be \".go\"\n", name)
		os.Exit(1)
	}

	// Read the source of the program.
	src, err := os.ReadFile(file)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	return scriggo.Files{name: src}
}
