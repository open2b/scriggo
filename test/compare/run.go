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
	"scriggo/template"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Some colors.
const (
	ColorInfo  = "\033[1;34m"
	ColorBad   = "\033[1;31m"
	ColorGood  = "\033[1;32m"
	ColorReset = "\033[0m"
)

// errorcheck executes the 'errorcheck' test on src.
func errorcheck(src []byte) {
	type stmt struct {
		line string
		err  *string
	}
	stmts := []stmt{}
	errorLines := []int{}
	for i, l := range strings.Split(string(src), "\n") {
		if index := strings.Index(l, "// ERROR "); index != -1 {
			err := l[index:]
			err = strings.TrimPrefix(err, "// ERROR ")
			err = strings.TrimSpace(err)
			if strings.HasPrefix(err, `"`) && strings.HasSuffix(err, `"`) {
				err = strings.TrimPrefix(err, `"`)
				err = strings.TrimSuffix(err, `"`)
			} else if strings.HasPrefix(err, "`") && strings.HasSuffix(err, "`") {
				err = strings.TrimPrefix(err, "`")
				err = strings.TrimSuffix(err, "`")
				err = regexp.QuoteMeta(err)
			} else {
				panic(fmt.Errorf("expected error must be followed by a string encapsulated in backticks (`) or double quotation marks (\"): %s", err))
			}

			stmts = append(stmts, stmt{line: l[:index], err: &err})
			errorLines = append(errorLines, i)
		} else {
			stmts = append(stmts, stmt{line: l, err: nil})
		}
	}
	if len(errorLines) == 0 {
		panic("no // ERROR comments found")
	}
	for _, errorLine := range errorLines {
		cleanSrc := &bytes.Buffer{}
		var expectedErr string
		for stmtLine, stmt := range stmts {
			if stmt.err == nil {
				cleanSrc.WriteString(stmt.line + "\n")
			} else {
				if stmtLine == errorLine {
					expectedErr = *stmt.err
					cleanSrc.WriteString(stmt.line + "\n")
				}
			}
		}
		out := runScriggo(cleanSrc.Bytes())
		if !out.isErr() {
			panic(fmt.Errorf("expected error %q, got %q", expectedErr, out.String()))
		}
		re := regexp.MustCompile(expectedErr)
		if !re.MatchString(out.String()) {
			panic(fmt.Errorf("error does not match:\n\n\texpecting:  %s\n\tgot:        %s", expectedErr, out.String()))
		}
	}
}

// mode reports the mode associated to src.
//
// Mode is specified in programs using a comment line (starting with "//"):
//
//  // mode
//
// templates must encapsulate mode in a line containing only a comment:
//
//  {# mode #}
//
// As a special case, the 'skip' comment can be followed by any sequence of
// characters (after a whitespace character) that will be ignored. This is
// useful to put an inline comment to the "skip" comment, explaining the reason
// why a given test cannot be run. For example:
//
//  // skip because feature X is not supported
//  // skip : enable when bug Y will be fixed
//
func mode(src []byte, ext string) string {
	switch ext {
	case ".go", ".sgo":
		for _, l := range strings.Split(string(src), "\n") {
			l = strings.TrimSpace(l)
			if l == "" {
				continue
			}
			if strings.HasPrefix(l, "//+build ignore") {
				panic("//+build ignore is no longer supported; remove such line from source")
			}
			if !strings.HasPrefix(l, "//") {
				return ""
			}
			l = strings.TrimPrefix(l, "//")
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "skip ") {
				return "skip"
			}
			return l
		}
	case ".html":
		for _, l := range strings.Split(string(src), "\n") {
			l = strings.TrimSpace(l)
			if l == "" {
				continue
			}
			if strings.HasPrefix(l, "{#") && strings.HasSuffix(l, "#}") {
				l = strings.TrimPrefix(l, "{#")
				l = strings.TrimSuffix(l, "#}")
				l = strings.TrimSpace(l)
				if strings.HasPrefix(l, "skip ") {
					return "skip"
				}
				return l
			}
		}
	default:
		panic("unsupported extension " + ext)
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
	// Is not an error.
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

// outputDetails returns a string which compares scriggo and gc outputs.
func outputDetails(scriggo, gc output) string {
	return fmt.Sprintf("\n\n\t[Scriggo]: %s\n\t[gc]:      %s\n", scriggo, gc)
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

// getAllFilepaths returns a list of filepaths matching the given pattern.
// If pattern is "", pattern matching is always assumed true.
func getAllFilepaths(pattern string) []string {
	const testsDir = "sources"
	filepaths := []string{}
	var re *regexp.Regexp
	if pattern != "" {
		re = regexp.MustCompile(pattern)
	}
	filepath.Walk(testsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}
		if re != nil {
			if !re.MatchString(path) {
				return nil
			}
		}
		if filepath.Ext(path) == ".go" || filepath.Ext(path) == ".html" {
			filepaths = append(filepaths, path)
		}
		return nil
	})
	if len(filepaths) == 0 {
		panic("specified pattern does not match any path")
	}
	return filepaths
}

func main() {

	// Parse the command line arguments.
	verbose := flag.Bool("v", false, "enable verbose output")
	pattern := flag.String("p", "", "executes test whose path is matched by the given pattern. Regexp is supported, in the syntax of stdlib package 'regexp'")
	time := flag.Bool("t", false, "show time elapsed between the beginning and the ending of every test. Require flag -v")
	color := flag.Bool("c", false, "enable colored output. Output device must support ANSI escape sequences. Require flag -v")

	defer func() {
		if r := recover(); r != nil {
			if *verbose {
				if *color {
					fmt.Print(ColorBad)
				}
				fmt.Print("\n\nPANIC!\n\n")
				if *color {
					fmt.Print(ColorReset)
				}
			}
			panic(r)
		}
	}()

	flag.Parse()

	if *time && !*verbose {
		panic("flag -t requires flag -v")
	}

	if *color && !*verbose {
		panic("flag -c requires flag -v")
	}

	// Get the list of all tests to run.
	filepaths := getAllFilepaths(*pattern)

	// Find the longest path in list: this is used later to format the output.
	maxPathLen := 0
	for _, fp := range filepaths {
		if len(fp) > maxPathLen {
			maxPathLen = len(fp)
		}
	}

	countTotal := 0
	countSkipped := 0

	for _, path := range filepaths {
		countTotal++
		src, err := ioutil.ReadFile(path)
		if err != nil {
			panic(err)
		}

		if *verbose {
			if *color {
				fmt.Print(ColorInfo)
			}
			perc := strconv.Itoa(int(math.Floor(float64(countTotal) / float64(len(filepaths)) * 100)))
			for i := len(perc); i < 4; i++ {
				perc = " " + perc
			}
			perc = "[" + perc + "%  ] "
			fmt.Print(perc)
			if *color {
				fmt.Print(ColorReset)
			}
			fmt.Print(path)
			for i := len(path); i < maxPathLen+2; i++ {
				fmt.Print(" ")
			}
		}

		// A timer is used to measure time that elapses during the execution of a given
		// test. Note that timer should measure only time needed by Scriggo; any
		// other elapsed time (eg. calling the gc compiler for the comparison
		// output) is considered overhead and should no be included in counting.
		tim := &timer{}

		ext := filepath.Ext(path)
		directive := mode(src, ext)
		switch ext {
		case ".go":
			switch directive {
			case "errorcheck":
				tim.start()
				errorcheck(src)
				tim.stop()
			case "skip":
				countSkipped++
				// Do nothing.
			case "compile", "build":
				tim.start()
				_, err := scriggo.LoadProgram(scriggo.Loaders(mainLoader(src), packages), &scriggo.LoadOptions{LimitMemorySize: true})
				tim.stop()
				if err != nil {
					panic(err.Error())
				}
			case "run":
				tim.start()
				scriggoOut := runScriggo(src)
				tim.stop()
				gcOut := runGc(src)
				if scriggoOut.isErr() && gcOut.isErr() {
					panic("expected succeed, but Scriggo and gc returned an error" + outputDetails(scriggoOut, gcOut))
				}
				if scriggoOut.isErr() && !gcOut.isErr() {
					panic("expected succeed, but Scriggo returned an error" + outputDetails(scriggoOut, gcOut))
				}
				if !scriggoOut.isErr() && gcOut.isErr() {
					panic("expected succeed, but gc returned an error" + outputDetails(scriggoOut, gcOut))
				}
				if scriggoOut.msg != gcOut.msg {
					panic("Scriggo and gc returned two different outputs" + outputDetails(scriggoOut, gcOut))
				}
			default:
				panic(fmt.Errorf("file %s has no valid directives", path))
			}
		case ".sgo":
			switch directive {
			case "skip":
				countSkipped++
				// Do nothing.
			default:
				panic(fmt.Errorf("file %s has no valid directives", path))
			}
		case ".html":
			switch directive {
			case "compile", "build":
				r := template.MapReader{"/index.html": src}
				_, err := template.Load("/index.html", r, nil, template.ContextHTML, nil)
				if err != nil {
					panic(err)
				}
			case "render":
				r := template.MapReader{"/index.html": src}
				tim.start()
				t, err := template.Load("/index.html", r, nil, template.ContextHTML, nil)
				if err != nil {
					panic(err)
				}
				tim.stop()
				w := &bytes.Buffer{}
				err = t.Render(w, nil, nil)
				if err != nil {
					panic(err)
				}
				goldenPath := strings.TrimSuffix(path, ".html") + ".golden"
				goldenData, err := ioutil.ReadFile(goldenPath)
				if err != nil {
					panic(err)
				}
				expected := strings.TrimSpace(string(goldenData))
				got := strings.TrimSpace(w.String())
				if expected != got {
					panic(fmt.Errorf("\n\nexpecting:  %s\ngot:        %s", expected, got))
				}
			case "errorcheck":
				panic("TODO: not implemented") // TODO(Gianluca): to implement.
			case "skip":
				countSkipped++
				// Do nothing.
			default:
				panic(fmt.Errorf("file %s has no valid directives", path))
			}
		case ".ext":
			panic("unsupported file extension " + ext)
		}

		// Test is completed, print some output and go to the next.
		if *verbose {
			if directive == "skip" {
				fmt.Println("[ skipped ]")
			} else {
				if *color {
					fmt.Printf(ColorGood+"ok"+ColorReset+"     (%s)", directive)
				} else {
					fmt.Printf("ok     (%s)", directive)
				}
				for i := len(directive); i < 15; i++ {
					fmt.Print(" ")
				}
				if *time {
					fmt.Println(tim.delta())
				} else {
					fmt.Println("")
				}
			}
		}

	}

	if *verbose {
		if *color {
			fmt.Print(ColorGood)
		}
		countExecuted := countTotal - countSkipped
		fmt.Printf("done!   %d tests executed, %d tests skipped\n", countExecuted, countSkipped)
		if *color {
			fmt.Print(ColorReset)
		}
	}

}
