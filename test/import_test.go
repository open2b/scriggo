// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"log"
	"testing"

	"scrigo"
)

func TestScrigoImport(t *testing.T) {
	cases := map[string]scrigo.MapReader{

		`Just package "main", no imports`: scrigo.MapReader(map[string][]byte{
			"/main": []byte(
				`package main
				func main() {
				}`)}),

		`"main" importing "pkg"`: scrigo.MapReader(map[string][]byte{
			"/main": []byte(
				`package main
				import "pkg"
				func main() {
					pkg.F()
				}`),
			"/pkg.go": []byte(
				`package pkg
				func F() {
					print("called pkg.F()")
				}`)}),

		`"main" importing "pkg1" and "pkg2`: scrigo.MapReader(map[string][]byte{
			"/main": []byte(
				`package main
				import "pkg1"
				import "pkg2"
				func main() {
					pkg1.F1()
					pkg2.F2()
				}`),
			"/pkg1.go": []byte(
				`package pkg1
				func F1() {
					print("called pkg1.F1()")
				}`),
			"/pkg2.go": []byte(
				`package pkg2
				func F2() {
					print("called pkg2.F2()")
				}`)}),

		`"main" importing "pkg1" importing "pkg2" (1)`: scrigo.MapReader(map[string][]byte{
			"/main": []byte(
				`package main
				import "pkg1"
				func main() {
					pkg1.F()
				}`),
			"/pkg1.go": []byte(
				`package pkg1
				import "pkg2"
				func F() {
					pkg2.G()
				}`),
			"/pkg2.go": []byte(
				`package pkg2
				func G() {
					print("called pkg2.G()")
				}`)}),

		`"main" importing "pkg1" importing "pkg2" (dot import)`: scrigo.MapReader(map[string][]byte{
			"/main": []byte(
				`package main
				import . "pkg1"
				func main() {
					F()
				}`),
			"/pkg1.go": []byte(
				`package pkg1
				import p2 "pkg2"
				func F() {
					p2.G()
				}`),
			"/pkg2.go": []byte(
				`package pkg2
				func G() {
					print("called pkg1.G()")
				}`)}),
	}
	for name, reader := range cases {
		t.Run(name, func(t *testing.T) {
			log.Printf("█ [DEBUG] █ name: %v\n", name) // TODO(Gianluca): remove.
			program, err := scrigo.LoadProgram([]scrigo.PackageImporter{reader}, scrigo.LimitMemorySize)
			if err != nil {
				t.Errorf("compiling error: %s", err)
				return
			}
			err = program.Run(scrigo.RunOptions{MaxMemorySize: 1000000})
			if err != nil {
				t.Errorf("execution error: %s", err)
				return
			}
		})
	}
}
