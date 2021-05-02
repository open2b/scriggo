// Copyright (c) 2021 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/open2b/scriggo"

	"golang.org/x/tools/txtar"
)

//go:embed *.tests *_test/*
var tests embed.FS

type programToRun struct {
	name string
	code *scriggo.Program
}

func main() {
	programs, err := build()
	if err != nil {
		_, _ = fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	for _, program := range programs {
		result := testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				_, err := program.code.Run(nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
		_, err := fmt.Fprint(os.Stderr, "Benchmark", program.name, result, result.MemString(), "\n")
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
	}
}

// build builds the programs in *.tests files and in the *_test directories.
func build() ([]programToRun, error) {
	var programs []programToRun
	err := fs.WalkDir(tests, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			if path != "." && !strings.HasSuffix(path, "_test") {
				return fs.SkipDir
			}
			return nil
		}
		if strings.Contains(path, "/") {
			data, _ := tests.ReadFile(path)
			program, err := scriggo.Build(bytes.NewReader(data), nil)
			if err != nil {
				return fmt.Errorf("cannot build %s: %s", path, err)
			}
			name := capitalizeTestName(path[:strings.Index(path, "_test/")])
			programs = append(programs, programToRun{name: name, code: program})
			return nil
		}
		if !strings.HasSuffix(path, ".tests") {
			return nil
		}
		tests, err := tests.ReadFile(path)
		if err != nil {
			return err
		}
		arch := txtar.Parse(tests)
		for _, file := range arch.Files {
			program, err := scriggo.Build(bytes.NewReader(file.Data), nil)
			if err != nil {
				return fmt.Errorf("cannot build %s/%s: %s", path, file.Name, err)
			}
			programs = append(programs, programToRun{
				name: capitalizeTestName(strings.TrimSuffix(path, ".tests")) + file.Name,
				code: program,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return programs, nil
}

// capitalizeTestName capitalizes a test name. Given "call_indirect" returns
// "CallIndirect". If name is malformed, it returns an empty string.
func capitalizeTestName(name string) string {
	var i = 0
	var n = make([]byte, len(name))
	for _, c := range name {
		if 'a' <= c && c <= 'z' {
			if i == 0 || n[i] == '_' {
				n[i] = byte(c) - 32
			} else {
				n[i] = byte(c)
			}
		} else if c == '_' {
			if i == 0 || n[i] == '_' {
				return ""
			}
			n[i] = byte(c)
		} else {
			return ""
		}
		if c != '_' {
			i++
		}
	}
	return string(n[:i])
}
