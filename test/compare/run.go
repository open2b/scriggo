// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {

	start := time.Now()

	// Parse the command line arguments.
	var (
		color             = flag.Bool("c", false, "enable colored output. output device must support ANSI escape sequences. require verbose to take effect")
		keepTestingOnFail = flag.Bool("k", false, "keep testing on fail")
		parallel          = flag.Int("l", 4, "number of parallel tests to run")
		pattern           = flag.String("p", "", "executes test whose path is matched by the given pattern. regular expressions are supported, in the syntax of stdlib package 'regexp'")
		verbose           = flag.Bool("v", false, "verbose. if set, parallelism is set to 1 and keep testing on fail is set to false")
	)

	flag.Parse()
	if *verbose {
		*parallel = 1
		*keepTestingOnFail = false
	}

	buildCmd()

	filepaths := getAllFilepaths(*pattern)

	// Look for the longest path; used when formatting output.
	maxPathLen := 0
	for _, fp := range filepaths {
		if len(fp) > maxPathLen {
			maxPathLen = len(fp)
		}
	}

	wg := sync.WaitGroup{}
	queue := make(chan bool, *parallel)

	countTotal := int64(0)
	countSkipped := int64(0)

	for _, path := range filepaths {

		wg.Add(1)

		go func(path string) {

			queue <- true
			atomic.AddInt64(&countTotal, 1)
			src, err := ioutil.ReadFile(path)
			if err != nil {
				panic(err)
			}
			ext := filepath.Ext(path)
			mode := readMode(src, ext)
			// Print output before running the test.
			if *verbose {
				if *color {
					fmt.Print(colorInfo)
				}
				perc := strconv.Itoa(int(math.Floor(float64(countTotal) / float64(len(filepaths)) * 100)))
				for i := len(perc); i < 4; i++ {
					perc = " " + perc
				}
				perc = "[" + perc + "%  ] "
				fmt.Print(perc)
				if *color {
					fmt.Print(colorReset)
				}
				fmt.Print(path)
				for i := len(path); i < maxPathLen+2; i++ {
					fmt.Print(" ")
				}
			}
			// Skip or run the test.
			if mode == "skip" {
				atomic.AddInt64(&countSkipped, 1)
			} else {
				test(src, path, mode, ext, *keepTestingOnFail)
			}
			// Print output when the test is completed.
			if *verbose {
				if mode == "skip" {
					fmt.Println("[ skipped ]")
				} else {
					if *color {
						fmt.Printf(colorGood+"ok"+colorReset+"     (%s)", mode)
					} else {
						fmt.Printf("ok     (%s)", mode)
					}
					for i := len(mode); i < 15; i++ {
						fmt.Print(" ")
					}
					fmt.Println("")
				}
			}
			<-queue
			wg.Done()
		}(path)
	}

	wg.Wait()

	// Print output after all tests are completed.
	if *verbose {
		if *color {
			fmt.Print(colorGood)
		}
		countExecuted := countTotal - countSkipped
		end := time.Now()
		fmt.Printf("done!   %d tests executed, %d tests skipped in %s\n", countExecuted, countSkipped, end.Sub(start).Truncate(time.Duration(time.Millisecond)))
		if *color {
			fmt.Print(colorReset)
		}
	}
}

const (
	colorInfo  = "\033[1;34m"
	colorBad   = "\033[1;31m"
	colorGood  = "\033[1;32m"
	colorReset = "\033[0m"
)

// buildCmd executes 'go build' on the directory 'cmd'.
func buildCmd() {
	cmd := exec.Command("go", "build")
	cmd.Dir = "./cmd"
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		panic(stderr.String())
	}
}

// cmd calls the executable "./cmd/cmd" passing the given arguments and the
// given stdin, if not nil.
func cmd(stdin []byte, args ...string) ([]byte, []byte) {
	cmd := exec.Command("./cmd/cmd", args...)
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = bytes.NewReader(stdin)
	_ = cmd.Run()
	return stdout.Bytes(), stderr.Bytes()
}

// errorcheck run test with mode 'errorcheck' on the given source code.
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
		arg := map[string]string{
			".go":   "run program",
			".sgo":  "run script",
			".html": "render html",
		}[ext]
		stdout, stderr := cmd(cleanSrc.Bytes(), arg)
		if len(stdout) > 0 {
			panic("stdout should be empty")
		}
		if len(stderr) == 0 {
			panic("expected error, got nothing")
		}
		re := regexp.MustCompile(expectedErr)
		if !re.Match(stderr) {
			panic(fmt.Errorf("error does not match:\n\n\texpecting:  %s\n\tgot:        %s", expectedErr, stderr))
		}
	}
}

// failOnOutput fails if at least one of the given streams is not empty.
func failOnOutput(stdout, stderr []byte) {
	_ = unwrapStdout(stdout, stderr)
	_ = unwrapStderr(stdout, stderr)
}

// getAllFilepaths returns a list of filepaths matching the given pattern.
// If pattern is "", pattern matching is always assumed true.
func getAllFilepaths(pattern string) []string {
	filepaths := []string{}
	var re *regexp.Regexp
	if pattern != "" {
		re = regexp.MustCompile(pattern)
	}
	filepath.Walk("testdata", func(path string, info os.FileInfo, err error) error {
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

// goldenCompare compares the golden file related to the given path with the
// given data.
func goldenCompare(testPath string, got []byte) {
	ext := filepath.Ext(testPath)
	if ext != ".go" && ext != ".sgo" && ext != ".html" {
		panic("unsupported ext: " + ext)
	}
	goldenPath := strings.TrimSuffix(testPath, ext) + ".golden"
	goldenData, err := ioutil.ReadFile(goldenPath)
	if err != nil {
		panic(err)
	}
	expected := bytes.TrimSpace(goldenData)
	got = bytes.TrimSpace(got)
	if bytes.Compare(expected, got) != 0 {
		panic(fmt.Errorf("\n\nexpecting:  %s\ngot:        %s", expected, got))
	}
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
func readMode(src []byte, ext string) (mode string) {
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
				panic(fmt.Errorf("not a valid directive: '%s'", l))
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
			} else {
				panic(fmt.Errorf("not a valid directive: '%s'", l))
			}
		}
	default:
		panic("unsupported extension " + ext)
	}
	panic("mode not specified")
}

// runGc runs a Go program using gc and returns its output.
func runGc(path string) ([]byte, []byte) {
	if ext := filepath.Ext(path); ext != ".go" {
		panic("unsupported ext " + ext)
	}
	cmd := exec.Command("go", "run", path)
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()
	return stdout.Bytes(), stderr.Bytes()
}

// test execute test on the given source code with the specified mode.
// When a test fails, test calls os.Exit with an non-zero error code, unless
// keepTestingOnFail is set.
func test(src []byte, path, mode, ext string, keepTestingOnFail bool) {

	defer func() {
		if r := recover(); r != nil {
			absPath, _ := filepath.Abs(path)
			fmt.Fprintf(os.Stderr, "\n%s: %s\n", absPath, r)
			if keepTestingOnFail {
				return
			}
			os.Exit(1)
		}
	}()

	switch mode + " " + ext {

	// Just compile.
	case "compile .go", "build .go":
		failOnOutput(
			cmd(src, "compile program"),
		)
	case "compile .sgo", "build .sgo":
		failOnOutput(
			cmd(src, "compile script"),
		)
	case "compile .html", "build .html":
		failOnOutput(
			cmd(src, "compile html"),
		)

	// Error check.
	case "errorcheck .go", "errorcheck .sgo", "errorcheck .html":
		errorcheck(src, ext)

	// Run or render.
	case "run .go":
		scriggoStdout, scriggoStderr := cmd(src, "run program")
		gcStdout, gcStderr := runGc(path)
		if len(scriggoStderr) > 0 && len(gcStderr) > 0 {
			panic(fmt.Errorf("expected succeed, but Scriggo returned error '%s' and gc returned error '%s'", scriggoStderr, gcStderr))
		}
		if len(scriggoStderr) > 0 && len(gcStderr) == 0 {
			panic(fmt.Errorf("expected succeed, but Scriggo returned error '%s'", scriggoStderr))
		}
		if len(scriggoStderr) == 0 && len(gcStderr) > 0 {
			panic(fmt.Errorf("expected succeed, but gc returned error '%s'", gcStderr))
		}
		if bytes.Compare(scriggoStdout, gcStdout) != 0 {
			panic("Scriggo and gc returned two different outputs: " + string(scriggoStdout) + ", " + string(gcStdout))
		}
	case "rundir .go":
		dirPath := strings.TrimSuffix(path, ".go") + ".dir"
		if _, err := os.Stat(dirPath); err != nil {
			panic(err)
		}
		goldenCompare(
			path,
			unwrapStdout(
				cmd(nil, "run program directory", dirPath),
			),
		)
	case "run .sgo":
		goldenCompare(
			path,
			unwrapStdout(
				cmd(src, "run script"),
			),
		)
	case "render .html":
		goldenCompare(
			path,
			unwrapStdout(
				cmd(src, "render html"),
			),
		)
	case "renderdir .html":
		goldenCompare(
			path,
			unwrapStdout(
				cmd(nil, "render html directory", strings.TrimSuffix(path, ".html")+".dir"),
			),
		)

	default:
		panic(fmt.Errorf("unsupported mode '%s' for test with extension '%s'", mode, ext))
	}

}

// unwrapStdout unwraps the given streams returning the stderr. Panics if stdout
// is not empty.
func unwrapStderr(stdout, stderr []byte) []byte {
	if len(stderr) > 0 {
		panic("unexpected standard output: " + string(stderr))
	}
	return stderr
}

// unwrapStdout unwraps the given streams returning the stdout. Panics if stderr
// is not empty.
func unwrapStdout(stdout, stderr []byte) []byte {
	if len(stderr) > 0 {
		panic("unexpected standard error: " + string(stderr))
	}
	return stdout
}
