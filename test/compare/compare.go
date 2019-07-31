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
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"scriggo"
	"strconv"
	"strings"
	"sync"
	"time"
)

// mode reports the mode associated to src. If no modes are specified, an empty
// string is returned.
func mode(src []byte) string {
	for _, l := range strings.Split(string(src), "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if !strings.HasPrefix(l, "//") {
			return ""
		}
		l = strings.TrimPrefix(l, "//")
		l = strings.TrimSpace(l)
		return l
	}
	panic("no directives found")
}

// packages contains the predefined packages used in tests.
//go:generate scriggo embed -v -o packages.go
var packages scriggo.Packages

// A timer returns the string representation of the time elapsed between two
// events.
type timer struct {
	_start, _end time.Time
}

// start starts the timer.
func (t *timer) start() {
	t._start = time.Now()
}

// stop stops the timer. After calling stop is possible to call method delta.
func (t *timer) stop() {
	t._end = time.Now()
}

// delta return the string representation of the time elapsed between the call
// to start to the call of stop.
func (t *timer) delta() string {
	return fmt.Sprintf("%v", t._end.Sub(t._start))
}

// output represents the output of Scriggo or gc. output can represent either a
// compilation error (syntax or type checking) or the output in case of success.
type output struct {
	path        string
	column, row string // not error if both ""
	msg         string
}

// match reports whether this error message matches another error message.
func (o output) match(o2 output) bool {
	if o.column != o2.column {
		return false
	}
	if o.msg != o2.msg {
		return false
	}
	return true
}

// isErr reports whether o is an error or not.
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

// parseOutputMessage parses a Scriggo or gc output message.
func parseOutputMessage(msg string) output {
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

// mainLoader is used to load the Scriggo source in function runScriggo.
type mainLoader []byte

func (b mainLoader) Load(path string) (interface{}, error) {
	if path == "main" {
		return bytes.NewReader(b), nil
	}
	return nil, nil
}

// runScriggo runs a Go program using Scriggo and returns its output.
func runScriggo(src []byte) output {
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
		return parseOutputMessage(err.Error())
	}
	err = program.Run(&scriggo.RunOptions{MaxMemorySize: 1000000})
	if err != nil {
		return parseOutputMessage(err.Error())
	}
	writer.Close()
	return parseOutputMessage(<-out)
}

// runGc runs a Go program using gc and returns its output.
func runGc(src []byte) output {
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
	return parseOutputMessage(out)
}

const testsDir = "sources"

func main() {

	verbose := flag.Bool("v", false, "enable verbose output")
	pattern := flag.String("p", "", "only execute path that match pattern")
	flag.Parse()
	testDirs, err := ioutil.ReadDir(testsDir)
	if err != nil {
		panic(err)
	}
	count := 0
	filepaths := []string{}
	for _, dir := range testDirs {
		if !dir.IsDir() {
			panic(fmt.Errorf("%s is not a dir", dir))
		}
		files, err := ioutil.ReadDir(filepath.Join(testsDir, dir.Name()))
		if err != nil {
			panic(err)
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
			panic(err)
		}

		if *verbose {
			perc := strconv.Itoa(int(math.Floor(float64(count) / float64(len(filepaths)) * 100)))
			for i := len(perc); i < 3; i++ {
				perc = " " + perc
			}
			perc = "[" + perc + "% ] "
			fmt.Print(perc)
			shortPath := strings.TrimPrefix(path, "sources/")
			fmt.Print(shortPath)
			for i := len(shortPath); i < 50; i++ {
				fmt.Print(" ")
			}
		}

		// A timer is used to measure time that elapses during the execution of a given
		// test. Note that timer should measure only time needed by Scriggo; any
		// other elapsed time (eg. calling the gc compiler for the comparison
		// output) is considered overhead and should no be included in counting.
		t := &timer{}

		directive := mode(src)
		switch directive {
		case "errcheck":
			panic("errcheck is currently not implemented")
		case "skip":
		case "errcmp":
			t.start()
			scriggoOut := runScriggo(src)
			t.stop()
			gcOut := runGc(src)
			if !scriggoOut.match(gcOut) {
				panic(fmt.Errorf("error on %q\n\tgc output:       %q\n\tScriggo output:  %q\n", path, gcOut, scriggoOut))
			}
		case "ignore":
		case "compile":
			_, err := scriggo.LoadProgram(scriggo.Loaders(mainLoader(src), packages), &scriggo.LoadOptions{LimitMemorySize: true})
			if err != nil {
				panic(err.Error())
			}
		case "run":
			t.start()
			_ = runScriggo(src)
			t.stop()
		case "runcmp":
			t.start()
			scriggoOut := runScriggo(src)
			t.stop()
			gcOut := runGc(src)
			if (scriggoOut.isErr() || gcOut.isErr()) && !strings.Contains(path, "errors") {
				fmt.Printf("\nTest %q returned an error, but source is not inside 'errors' directory\n", path)
				fmt.Printf("\nERROR on %q\n\tgc output:       %q\n\tScriggo output:  %q\n", path, gcOut, scriggoOut)
				return
			}
			if !scriggoOut.isErr() && !gcOut.isErr() && strings.Contains(path, "errors") {
				fmt.Printf("\nTest %q should return error (is inside 'errors' dir), but it doesn't\n", path)
				return
			}
			if !scriggoOut.match(gcOut) {
				panic(fmt.Errorf("error on %q\n\tgc output:       %q\n\tScriggo output:  %q\n", path, gcOut, scriggoOut))
			}
		default:
			panic(fmt.Errorf("file %s has no valid directives", path))
		}

		// Test is completed, print some output and go to next.
		switch directive {
		case "ignore":
			fmt.Println("[ ignored ]")
		case "skip":
			fmt.Println("[ skipped ]")
		default:
			fmt.Printf("ok     (%s)", directive)
			for i := len(directive); i < 15; i++ {
				fmt.Print(" ")
			}
			fmt.Println(t.delta())
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
