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

// A dirLoader is a package loader used in tests which involve directories
// containing Scriggo programs.
type dirLoader string

// Load implement interface scriggo.PackageLoader.
func (dl dirLoader) Load(path string) (interface{}, error) {
	if path == "main" {
		main, err := ioutil.ReadFile(filepath.Join(string(dl), "main.go"))
		if err != nil {
			panic(err)
		}
		return bytes.NewReader(main), nil
	}
	data, err := ioutil.ReadFile(filepath.Join(string(dl), path+".go"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// isTestPath reports whether path is a valid test path.
func isTestPath(path string) bool {
	if filepath.Ext(path) != ".go" && filepath.Ext(path) != ".sgo" && filepath.Ext(path) != ".html" {
		return false
	}
	if filepath.Ext(filepath.Dir(path)) == ".dir" {
		return false
	}
	return true
}

// errorcheck executes the 'errorcheck' test on src.
func errorcheck(src []byte, ext string) {
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
		// Get output from program/script/templates and check if it matches with
		// expected error.
		var out string
		switch ext {
		case ".go":
			ret := runScriggo(cleanSrc.Bytes())
			if !ret.isErr() {
				panic(fmt.Errorf("expected error %q, got %q", expectedErr, ret.String()))
			}
			out = ret.String()
		case ".sgo":
			_, err := scriggo.LoadScript(bytes.NewReader(src), packages, nil)
			if err == nil {
				panic(fmt.Errorf("expected error %q, got nothing", expectedErr))
			}
			out = err.Error()
		case ".html":
			r := template.MapReader{"/index.html": src}
			_, err := template.Load("/index.html", r, nil, template.ContextHTML, nil)
			if err == nil {
				panic(fmt.Errorf("expected error %q, got nothing", expectedErr))
			}
			out = err.Error()
		default:
			panic("errorcheck does not support extension " + ext)
		}
		re := regexp.MustCompile(expectedErr)
		if !re.MatchString(out) {
			panic(fmt.Errorf("error does not match:\n\n\texpecting:  %s\n\tgot:        %s", expectedErr, out))
		}
	}
}

// readMode reports the readMode associated to src.
//
// Mode is specified in programs and scripts using a comment line (starting with
// "//"):
//
//  // readMode
//
// templates must encapsulate readMode in a line containing just a comment:
//
//  {# readMode #}
//
// As a special case, the 'skip' comment can be followed by any sequence of
// characters (after a whitespace character) that will be ignored. This is
// useful to put an inline comment to the "skip" comment, explaining the reason
// why a given test cannot be run. For example:
//
//  // skip because feature X is not supported
//  // skip : enable when bug Y will be fixed
//
// When onlySkipped is true, all tests with the "skip" comment are executed
// (according to the next comment mode found) and other tests (which would be
// runned normally) are ignored.
//
func readMode(src []byte, ext string, onlySkipped bool) (mode string, mustBeIgnored bool) {
	hasSkip := false
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
				continue
			}
			l = strings.TrimPrefix(l, "//")
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "skip ") {
				// Comment containing "skip" is ignored, moving to the next
				// comment, which has been "skipped."
				if onlySkipped {
					hasSkip = true
					continue
				}
				return "skip", false
			}
			// mode != "skip"
			// Comment is not "skip", but readMode is looking for skipped
			// tests, so any test that contains a valid directive is
			// ignored.
			if onlySkipped && !hasSkip {
				return "", true
			}
			return l, false
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
					if onlySkipped {
						// Comment containing "skip" is ignored, moving to the
						// next comment, which has been "skipped."
						hasSkip = true
						continue
					}
					return "skip", false
				}
				// mode != "skip"
				if onlySkipped && !hasSkip {
					// Comment is not "skip", but readMode is looking for
					// skipped tests, so any test that contains a valid
					// directive is ignored.
					return "", true
				}
				return l, false
			}
		}
	default:
		panic("unsupported extension " + ext)
	}
	panic("mode not specified")
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

// callCatchingStdout calls the given function, catching the standard output and
// returning it as a string.
func callCatchingStdout(f func()) string {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	backup := os.Stdout
	os.Stdout = w
	defer func() {
		os.Stdout = backup
	}()
	out := make(chan string)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		var buf bytes.Buffer
		wg.Done()
		io.Copy(&buf, r)
		out <- buf.String()
	}()
	wg.Wait()
	f()
	w.Close()
	return <-out
}

// runScriggo runs a Go program using Scriggo and returns its output.
func runScriggo(src []byte) output {
	program, err := scriggo.LoadProgram(scriggo.Loaders(mainLoader(src), packages), nil)
	if err != nil {
		return parseOutputMessage(err.Error())
	}
	stdout := callCatchingStdout(func() {
		err = program.Run(nil)
	})
	if err != nil {
		return parseOutputMessage(err.Error())
	}
	return parseOutputMessage(stdout)
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
	const testsDir = "src"
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
		if isTestPath(path) {
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
	onlySkipped := flag.Bool("only-skipped", false, "only run skipped tests, returning an error instead of panicking when one of them fails. Note that skipped tests may lead to unexpected behaviours and/or infinite loops")

	flag.Parse()
	if *time && !*verbose {
		panic("flag -t requires flag -v")
	}
	if *color && !*verbose {
		panic("flag -c requires flag -v")
	}

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

	filepaths := getAllFilepaths(*pattern)
	maxPathLen := 0
	for _, fp := range filepaths {
		if len(fp) > maxPathLen {
			maxPathLen = len(fp)
		}
	}
	countTotal := 0
	countSkipped := 0
	for _, path := range filepaths {
		func() {
			countTotal++
			src, err := ioutil.ReadFile(path)
			if err != nil {
				panic(err)
			}
			// A timer is used to measure time that elapses during the execution of a given
			// test. Note that timer should measure only time needed by Scriggo; any
			// other elapsed time (eg. calling the gc compiler for the comparison
			// output) is considered overhead and should no be included in counting.
			t := &timer{}
			ext := filepath.Ext(path)
			mode, mustBeIgnored := readMode(src, ext, *onlySkipped)
			if mustBeIgnored {
				return
			}
			if *onlySkipped {
				defer func() {
					if r := recover(); r != nil {
						fmt.Println("panicked: %s", r)
					}
				}()
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
			switch ext + "." + mode {
			case ".go.skip", ".sgo.skip", ".html.skip":
				countSkipped++
			case ".go.compile", ".go.build":
				t.start()
				_, err := scriggo.LoadProgram(scriggo.Loaders(mainLoader(src), packages), &scriggo.LoadOptions{LimitMemorySize: true})
				t.stop()
				if err != nil {
					panic(err.Error())
				}
			case ".go.run":
				t.start()
				scriggoOut := runScriggo(src)
				t.stop()
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
			case ".go.rundir":
				dirPath := strings.TrimSuffix(path, ".go") + ".dir"
				if _, err := os.Stat(dirPath); err != nil {
					panic(err)
				}
				dl := dirLoader(dirPath)
				t.start()
				prog, err := scriggo.LoadProgram(scriggo.CombinedLoaders{dl, packages}, nil)
				if err != nil {
					panic(err)
				}
				stdout := callCatchingStdout(func() {
					err = prog.Run(nil)
				})
				t.stop()
				if err != nil {
					panic(err)
				}
				goldenPath := strings.TrimSuffix(path, ".go") + ".golden"
				goldenData, err := ioutil.ReadFile(goldenPath)
				if err != nil {
					panic(err)
				}
				expected := strings.TrimSpace(string(goldenData))
				got := strings.TrimSpace(stdout)
				if expected != got {
					panic(fmt.Errorf("\n\nexpecting:  %s\ngot:        %s", expected, got))
				}
			case ".sgo.compile", ".sgo.build":
				t.start()
				_, err := scriggo.LoadScript(bytes.NewReader(src), packages, nil)
				t.stop()
				if err != nil {
					panic(err)
				}
			case ".go.errorcheck", ".sgo.errorcheck", ".html.errorcheck":
				t.start()
				errorcheck(src, ext)
				t.stop()
			case ".sgo.run":
				t.start()
				script, err := scriggo.LoadScript(bytes.NewReader(src), packages, nil)
				if err != nil {
					panic(err)
				}
				stdout := callCatchingStdout(func() {
					err = script.Run(nil, nil)
				})
				if err != nil {
					panic(err)
				}
				t.stop()
				goldenPath := strings.TrimSuffix(path, ".sgo") + ".golden"
				goldenData, err := ioutil.ReadFile(goldenPath)
				if err != nil {
					panic(err)
				}
				expected := strings.TrimSpace(string(goldenData))
				got := strings.TrimSpace(stdout)
				if expected != got {
					panic(fmt.Errorf("\n\nexpecting:  %s\ngot:        %s", expected, got))
				}
			case ".html.compile", ".html.build":
				r := template.MapReader{"/index.html": src}
				t.start()
				_, err := template.Load("/index.html", r, nil, template.ContextHTML, nil)
				t.stop()
				if err != nil {
					panic(err)
				}
			case ".html.render", ".html.renderdir":
				var r template.Reader
				if mode == "render" {
					r = template.MapReader{"/index.html": src}
				} else {
					r = template.DirReader(strings.TrimSuffix(path, ".html") + ".dir")
				}
				t.start()
				templ, err := template.Load("/index.html", r, nil, template.ContextHTML, nil)
				if err != nil {
					panic(err)
				}
				w := &bytes.Buffer{}
				err = templ.Render(w, nil, nil)
				t.stop()
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
			default:
				panic(fmt.Errorf("unsupported mode '%s' for test with extension '%s'", mode, ext))
			}

			// Test is completed, print some output and go to the next.
			if *verbose {
				if mode == "skip" {
					fmt.Println("[ skipped ]")
				} else {
					if *color {
						fmt.Printf(ColorGood+"ok"+ColorReset+"     (%s)", mode)
					} else {
						fmt.Printf("ok     (%s)", mode)
					}
					for i := len(mode); i < 15; i++ {
						fmt.Print(" ")
					}
					if *time {
						fmt.Println(t.delta())
					} else {
						fmt.Println("")
					}
				}
			}
		}()
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
