// Copyright (c) 2021 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"testing"

	"github.com/open2b/scriggo"

	"golang.org/x/tools/txtar"
)

//go:embed micro.txtar
var tests []byte

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

func build() ([]programToRun, error) {
	arch := txtar.Parse(tests)
	programs := make([]programToRun, len(arch.Files))
	for i, file := range arch.Files {
		program, err := scriggo.Build(bytes.NewReader(file.Data), nil, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot build %s: %s", file.Name, err)
		}
		programs[i].name = file.Name
		programs[i].code = program
	}
	return programs, nil
}
