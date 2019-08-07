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
	"scriggo"
	"strconv"
	"strings"
	"time"
)

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

// TODO(Gianluca): use []byte and compare them. Convert to string only if
// necessary. Use bytes.TrimSpace.
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

// Some colors.
const (
	ColorInfo  = "\033[1;34m"
	ColorBad   = "\033[1;31m"
	ColorGood  = "\033[1;32m"
	ColorReset = "\033[0m"
)

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

	start := time.Now()

	// Parse the command line arguments.
	verbose := flag.Bool("v", false, "enable verbose output")
	pattern := flag.String("p", "", "executes test whose path is matched by the given pattern. Regexp is supported, in the syntax of stdlib package 'regexp'")
	color := flag.Bool("c", false, "enable colored output. Output device must support ANSI escape sequences. Require flag -v")
	onlySkipped := flag.Bool("only-skipped", false, "only run skipped tests, returning an error instead of panicking when one of them fails. Note that skipped tests may lead to unexpected behaviours and/or infinite loops")

	flag.Parse()
	if *color && !*verbose {
		panic("flag -c requires flag -v")
	}

	// Builds cmd.
	{
		cmd := exec.Command("go", "build")
		cmd.Dir = "./cmd"
		stderr := &bytes.Buffer{}
		cmd.Stderr = stderr
		err := cmd.Run()
		if err != nil {
			panic(stderr.String())
		}
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
			switch mode + " " + ext {
			case "skip .go", "skip .sgo", "skip .html":
				countSkipped++
			case "compile .go", "build .go":
				cmd(src, "compile program")
			case "run .go":
				scriggoStdout, scriggoStderr := cmd(src, "run program")
				gcStdout, gcStderr := runGc(path)
				if len(scriggoStderr) > 0 && len(gcStderr) > 0 {
					panic("expected succeed, but Scriggo and gc returned an error")
				}
				if len(scriggoStderr) > 0 && len(gcStderr) == 0 {
					panic("expected succeed, but Scriggo returned an error")
				}
				if len(scriggoStderr) == 0 && len(gcStderr) > 0 {
					panic("expected succeed, but gc returned an error")
				}
				if bytes.Compare(scriggoStdout, gcStdout) != 0 {
					panic("Scriggo and gc returned two different outputs")
				}
			case "rundir .go":
				dirPath := strings.TrimSuffix(path, ".go") + ".dir"
				if _, err := os.Stat(dirPath); err != nil {
					panic(err)
				}
				stdout, stderr := cmd(nil, "run program directory", dirPath)
				if len(stderr) > 0 {
					panic("unexpected stderr " + string(stderr))
				}
				goldenCompare(path, stdout)
			case "compile .sgo", "build .sgo":
				cmd(src, "compile script")
			case "errorcheck .go", "errorcheck .sgo", "errorcheck .html":
				errorcheck(src, ext)
			case "run .sgo":
				stdout, stderr := cmd(src, "run script")
				if len(stderr) > 0 {
					panic("unexpected stderr " + string(stderr))
				}
				goldenCompare(path, stdout)
			case "compile .html", "build .html":
				cmd(src, "compile html")
			case "render .html":
				stdout, stderr := cmd(src, "render html")
				if len(stderr) > 0 {
					panic("unexpected stderr " + string(stderr))
				}
				goldenCompare(path, stdout)
			case "renderdir .html":
				dirPath := strings.TrimSuffix(path, ".html") + ".dir"
				stdout, stderr := cmd(nil, "render html directory", dirPath)
				if len(stderr) > 0 {
					panic("unexpected stderr " + string(stderr))
				}
				goldenCompare(path, stdout)
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
					fmt.Println("")
				}
			}
		}()
	}
	if *verbose {
		if *color {
			fmt.Print(ColorGood)
		}
		countExecuted := countTotal - countSkipped
		end := time.Now()
		fmt.Printf("done!   %d tests executed, %d tests skipped in %s\n", countExecuted, countSkipped, end.Sub(start).Truncate(time.Duration(time.Millisecond)))
		if *color {
			fmt.Print(ColorReset)
		}
	}

}
