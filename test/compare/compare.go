// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"scriggo"
)

//go:generate scriggo embed -v -o packages.go
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

	program, err := scriggo.LoadProgram(scriggo.Loaders(mainLoader(src), packages), &scriggo.LoadOptions{LimitMemorySize: true})

	if err != nil {
		return makeOutput(err.Error())
	}
	err = program.Run(&scriggo.RunOptions{MaxMemorySize: 1000000})
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

	verbose := flag.Bool("v", false, "enable verbose output")
	pattern := flag.String("p", "", "only execute path that match pattern")
	flag.Parse()
	testDirs, err := ioutil.ReadDir(testsDir)
	if err != nil {
		fatal(err)
	}
	count := 0
	filepaths := []string{}
	for _, dir := range testDirs {
		if !dir.IsDir() {
			fatal(fmt.Errorf("%s is not a dir", dir))
		}
		files, err := ioutil.ReadDir(filepath.Join(testsDir, dir.Name()))
		if err != nil {
			fatal(err)
		}
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".go") {
				continue
			}
			path := filepath.Join(testsDir, dir.Name(), f.Name())
			if *pattern != "" {
				if !strings.Contains(path, *pattern) {
					continue
				}
			}
			filepaths = append(filepaths, path)
		}
	}

	if len(filepaths) == 0 {
		fmt.Printf("warning: no matches for pattern %q\n", *pattern)
		os.Exit(0)
	}

	for _, path := range filepaths {
		count++
		src, err := ioutil.ReadFile(path)
		if err != nil {
			fatal(err)
		}
		switch mode(src) {
		case "ignore":
			continue
		}
		if *verbose {
			fmt.Printf(strings.TrimPrefix(path, "sources"))
		}
		start := time.Now()
		scriggoOut := runScriggoAndGetOutput(src)
		if *verbose {
			end := time.Now()
			for i := len(path); i < 50; i++ {
				fmt.Printf(" ")
			}
			fmt.Printf(" ok   %v", end.Sub(start))
			fmt.Printf(" [%d%%]\n", int(math.Floor(float64(count)/float64(len(filepaths))*100)))
		} else {
			fmt.Printf("\r%d%% ", int(math.Floor(float64(count)/float64(len(filepaths))*100)))
		}
		goOut := runGoAndGetOutput(src)
		if (scriggoOut.isErr() || goOut.isErr()) && !strings.Contains(path, "errors") {
			fmt.Printf("\nTest %q returned an error, but source is not inside 'errors' directory\n", path)
			fmt.Printf("\nERROR on %q\n\tgc output:       %q\n\tScriggo output:  %q\n", path, goOut, scriggoOut)
			return
		}
		if !scriggoOut.isErr() && !goOut.isErr() && strings.Contains(path, "errors") {
			fmt.Printf("\nTest %q should return error (is inside 'errors' dir), but it doesn't\n", path)
			return
		}
		if !scriggoOut.match(goOut) {
			panic(fmt.Errorf("error on %q\n\tgc output:       %q\n\tScriggo output:  %q\n", path, goOut, scriggoOut))
		}
	}

	fmt.Print("done! (")
	switch count {
	case 1:
		fmt.Print("1 test executed")
	default:
		fmt.Printf("%d tests executed", count)
	}
	fmt.Print(")\n")

}
