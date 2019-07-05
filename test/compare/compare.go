// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"scriggo"
)

var packages scriggo.Packages

type output struct {
	path        string
	column, row string
	msg         string
}

func makeOutput(msg string) output {
	r := regexp.MustCompile(`([\w\. ]*):(\d+):(\d+):\s(.*)`)
	if !r.MatchString(msg) {
		return output{msg: msg}
	}
	m := r.FindStringSubmatch(msg)
	path := m[1]
	column := m[2]
	row := m[3]
	msg = m[4]
	return output{
		path:   path,
		column: column,
		row:    row,
		msg:    msg,
	}
}

func (o output) match(o2 output) bool {
	if o.column != o2.column {
		return false
	}
	if o.msg != o2.msg {
		return false
	}
	return true
}

func (o output) isErr() bool {
	return o.column != "" && o.row != ""
}

func (o output) String() string {
	if !o.isErr() {
		return o.msg
	}
	path := o.path
	if path == "" {
		path = "[nopath]"
	}
	return path + ":" + o.column + ":" + o.row + " " + o.msg
}

type mainLoader []byte

func (b mainLoader) Load(path string) (interface{}, error) {
	if path == "main" {
		return bytes.NewReader(b), nil
	}
	return nil, nil
}

func runScriggoAndGetOutput(src []byte) output {
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	backupStdout := os.Stdout
	backupStderr := os.Stderr
	os.Stdout = writer
	os.Stderr = writer
	defer func() {
		os.Stdout = backupStdout
		os.Stderr = backupStderr
	}()
	out := make(chan string)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		var buf bytes.Buffer
		wg.Done()
		io.Copy(&buf, reader)
		out <- buf.String()
	}()
	wg.Wait()

	program, err := scriggo.LoadProgram(scriggo.Loaders(mainLoader(src), packages), scriggo.LoadOption{LimitMemorySize: true})

	if err != nil {
		return makeOutput(err.Error())
	}
	err = program.Run(scriggo.RunOptions{MaxMemorySize: 1000000})
	if err != nil {
		return makeOutput(err.Error())
	}
	writer.Close()
	return makeOutput(<-out)
}

func runGoAndGetOutput(src []byte) output {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "scriggotesting")
	if err != nil {
		panic(err)
	}
	defer func() {
		if !strings.HasPrefix(tmpDir, os.TempDir()) {
			panic(fmt.Errorf("invalid tmpDir: %q", tmpDir))
		}
		_ = os.RemoveAll(tmpDir)
	}()
	tmpFile, err := os.Create(filepath.Join(tmpDir, "globals.go"))
	if err != nil {
		panic(err)
	}
	_, err = tmpFile.Write(src)
	if err != nil {
		panic(err)
	}
	cmd := exec.Command("go", "run", filepath.Base(tmpFile.Name()))
	cmd.Dir = tmpDir
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()
	out := stdout.String() + stderr.String()
	return makeOutput(out)
}

const testsDir = "sources"

func fatal(a interface{}) {
	log.Fatalf("fatal error: %s", a)
}

func main() {
	verbose := false
	veryVerbose := false
	mustRecover := true
	pattern := ""
	for i, arg := range os.Args {
		if arg == "-v" || arg == "--verbose" {
			verbose = true
		}
		if arg == "-n" || arg == "--no-recover" {
			mustRecover = false
		}
		if arg == "-V" || arg == "--veryVerbose" {
			veryVerbose = true
			verbose = true
		}
		if arg == "-h" || arg == "--help" {
			fmt.Printf("Usage: %s [--no-recover|-n] [--verbose|-v] [--veryVerbose|-V] [--help|-h] [--pattern|-p PATTERN]\n", os.Args[0])
			os.Exit(0)
		}
		if arg == "-p" || arg == "--pattern" {
			pattern = os.Args[i+1]
		}
	}
	testDirs, err := ioutil.ReadDir(testsDir)
	if err != nil {
		fatal(err)
	}
	if !verbose {
		fmt.Print("comparing Scriggo with gc..")
	}
	count := 0
	for _, dir := range testDirs {
		if !dir.IsDir() {
			fatal(fmt.Errorf("%s is not a dir", dir))
		}
		files, err := ioutil.ReadDir(filepath.Join(testsDir, dir.Name()))
		if err != nil {
			fatal(err)
		}
		for _, f := range files {
			func() {
				defer func() {
					if mustRecover {
						if r := recover(); r != nil {
							fmt.Printf("!!! PANIC !!!: %v\n", r)
						}
					}
				}()
				if !strings.HasSuffix(f.Name(), ".go") {
					return
				}
				path := filepath.Join(testsDir, dir.Name(), f.Name())
				if strings.Contains(path, "_ignore_") {
					return
				}
				if pattern != "" {
					if !strings.Contains(path, pattern) {
						return
					}
				}
				src, err := ioutil.ReadFile(path)
				if err != nil {
					fatal(err)
				}
				if verbose {
					fmt.Printf("---------------------------------------------------\n")
					fmt.Print(path + "...")
				} else {
					fmt.Print(".")
				}
				count++
				scriggoOut := runScriggoAndGetOutput(src)
				goOut := runGoAndGetOutput(src)
				if (scriggoOut.isErr() || goOut.isErr()) && (dir.Name() != "errors") {
					fmt.Printf("\nTest %q returned an error, but source is not inside 'errors' directory\n", path)
					fmt.Printf("\nERROR on %q\n\tgc output:       %q\n\tScriggo output:  %q\n", path, goOut, scriggoOut)
					return
				}
				if !scriggoOut.isErr() && !goOut.isErr() && (dir.Name() == "errors") {
					fmt.Printf("\nTest %q should return error (is inside 'errors' dir), but it doesn't\n", path)
					return
				}
				if scriggoOut.match(goOut) {
					if veryVerbose {
						fmt.Printf("\noutput:\n%s\n", scriggoOut)
					}
					if verbose {
						fmt.Println("OK!")
					}
				} else {
					fmt.Printf("\nERROR on %q\n\tgc output:       %q\n\tScriggo output:  %q\n", path, goOut, scriggoOut)
				}
			}()
		}
	}
	if !verbose {
		fmt.Printf("done! (%d tests executed)\n", count)
	}
}
