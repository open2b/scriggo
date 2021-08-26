// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rogpeppe/go-internal/imports"
)

var BOM = []byte("\ufeff")

const (
	colorInfo  = "\033[1;34m"
	colorBad   = "\033[1;31m"
	colorGood  = "\033[1;32m"
	colorReset = "\033[0m"
)

func main() {

	start := time.Now()

	// Parse the command line arguments.
	var (
		color             = flag.Bool("c", false, "enable colored output. output device must support ANSI escape sequences. require verbose or stats to take effect")
		keepTestingOnFail = flag.Bool("k", false, "keep testing on fail")
		parallel          = flag.Int("l", 4, "number of parallel tests to run")
		pattern           = flag.String("p", "", "executes test whose path is matched by the given pattern. regular expressions are supported, in the syntax of stdlib package 'regexp'")
		stat              = flag.Bool("s", false, "print some stats about executed tests. Parallelism is not influenced.")
		verbose           = flag.Bool("v", false, "verbose. if set, parallelism is set to 1 and keep testing on fail is set to false")
	)

	flag.Parse()
	if *verbose {
		*parallel = 1
		*keepTestingOnFail = false
	}

	if flag.NArg() > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "too many arguments on the command line\n")
		flag.Usage()
		os.Exit(1)
	}

	// Fill the build tags.
	var tags map[string]bool
	{
		goos := os.Getenv("GOOS")
		if goos == "" {
			goos = runtime.GOOS
		}
		goarch := os.Getenv("GOARCH")
		if goarch == "" {
			goarch = runtime.GOARCH
		}
		tags = map[string]bool{
			goos:   true,
			goarch: true,
			"cgo":  cgoEnabled,
		}
		// Add the build tags from the first version of go to the current one.
		goVersion := goBaseVersion(runtime.Version())
		subv, err := strconv.Atoi(goVersion[len("go1."):])
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for i := 1; i <= subv; i++ {
			tags[fmt.Sprintf("go1.%d", i)] = true
		}
	}

	err := buildCmd()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "cannot build cmd: %s\n", err)
		os.Exit(1)
	}

	paths, err := filePaths(*pattern)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(paths) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "specified pattern does not match any path")
		os.Exit(1)
	}

	// Look for the longest path; used when formatting output.
	maxPathLen := 0
	for _, fp := range paths {
		if len(fp) > maxPathLen {
			maxPathLen = len(fp)
		}
	}

	if *verbose || *stat {
		fmt.Printf("%d tests found (including skipped and incompatible ones)\n", len(paths))
	}

	wg := sync.WaitGroup{}
	queue := make(chan bool, *parallel)

	countTotal := int64(0)
	countSkipped := int64(0)
	countIncompatible := int64(0)

	for _, path := range paths {

		wg.Add(1)

		go func(path string) {
			queue <- true
			atomic.AddInt64(&countTotal, 1)
			src, err := ioutil.ReadFile(path)
			if err != nil {
				panic(err)
			}
			ext := filepath.Ext(path)
			// Print output before running the test.
			if *stat && !*verbose {
				percentage := strconv.Itoa(int(math.Floor(float64(countTotal) / float64(len(paths)) * 100)))
				fmt.Printf("\r%s%% (test %d/%d)", percentage, countTotal, len(paths))
			}
			if *verbose {
				if *color {
					fmt.Print(colorInfo)
				}
				percentage := strconv.Itoa(int(math.Floor(float64(countTotal) / float64(len(paths)) * 100)))
				for i := len(percentage); i < 4; i++ {
					percentage = " " + percentage
				}
				fmt.Print("[" + percentage + "%  ] ")
				if *color {
					fmt.Print(colorReset)
				}
				fmt.Print(path)
				for i := len(path); i < maxPathLen+2; i++ {
					fmt.Print(" ")
				}
			}
			if !imports.ShouldBuild(src, tags) {
				atomic.AddInt64(&countIncompatible, 1)
				if *verbose {
					fmt.Println("[ skipped, not compatible ]")
				}
				<-queue
				wg.Done()
				return
			}
			mode, opts, err := readMode(src, ext)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "cannot read mode: %s\n", err)
				os.Exit(1)
			}
			// Skip or run the test.
			if mode == "skip" {
				atomic.AddInt64(&countSkipped, 1)
			} else {
				err = test(src, mode, path, opts, *keepTestingOnFail)
				if err != nil {
					_, _ = fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
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
	if *verbose || *stat {
		if *color {
			fmt.Print(colorGood)
		}
		countExecuted := countTotal - countSkipped - countIncompatible
		totalTime := time.Now().Sub(start).Truncate(time.Millisecond)
		if countIncompatible > 0 {
			fmt.Printf("\ndone!   %d tests executed, %d tests skipped (with %d not compatible) in %s\n",
				countExecuted, countSkipped+countIncompatible, countIncompatible, totalTime)
		} else {
			fmt.Printf("\ndone!   %d tests executed, %d tests skipped in %s\n",
				countExecuted, countSkipped, totalTime)
		}
		if *color {
			fmt.Print(colorReset)
		}
	}

}

// buildCmd builds the 'cmd' command.
func buildCmd() error {
	cmd := exec.Command("go", "build")
	cmd.Dir = "./cmd"
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return errors.New(stderr.String())
	}
	return nil
}

// cmd calls cmd with given stdin, options and arguments.
func cmd(stdin []byte, opts []string, args ...string) (int, []byte, []byte) {
	var cmdArgs []string
	cmdArgs = append(cmdArgs, opts...)
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command("./cmd/cmd", cmdArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = bytes.NewReader(stdin)
	err := cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ProcessState.ExitCode(), stdout.Bytes(), stderr.Bytes()
		}
		panic(err)
	}
	return 0, stdout.Bytes(), stderr.Bytes()
}

var lf = []byte{'\n'}
var errorComment = []byte("// ERROR ")

// errorcheck runs the tests with mode 'errorcheck' on the given source code.
func errorcheck(src []byte, filePath string, opts []string) error {

	ext := filepath.Ext(filePath)

	tests := differentiateSources(string(src))

	// First do a test without error lines.
	var srcWithoutErrors []byte
	for _, line := range bytes.SplitAfter(src, lf) {
		if bytes.Contains(line, errorComment) {
			srcWithoutErrors = append(srcWithoutErrors, '\n')
		} else {
			srcWithoutErrors = append(srcWithoutErrors, line...)
		}
	}
	exitCode, _, stderr := cmd(srcWithoutErrors, opts, "build", ext)
	if exitCode != 0 {
		if p := bytes.Index(stderr, []byte{':'}); p > 0 {
			return fmt.Errorf("%s%s", filePath, stderr[p:])
		}
		return fmt.Errorf("%s: %s", filePath, stderr)
	}

	if len(tests) == 0 {
		return errors.New("no // ERROR comments found")
	}

	for _, test := range tests {
		// Get output from program/templates and check if it matches with the
		// expected error.
		exitCode, stdout, stderr := cmd([]byte(test.src), opts, "build", ext)
		if exitCode == 0 {
			return fmt.Errorf("expecting error '%s' but exit code is 0", test.err)
		}
		if len(stdout) > 0 {
			return errors.New("stdout should be empty")
		}
		if len(stderr) == 0 {
			return fmt.Errorf("expected error '%s', got nothing", test.err)
		}
		re := regexp.MustCompile(test.err)
		stderr = []byte(removePrefixFromError(string(stderr)))
		if !re.Match(stderr) {
			return fmt.Errorf("output does not match with the given regular expression:\n\n\tregexp:    %s\n\toutput:    %s\n", test.err, stderr)
		}
	}

	return nil
}

// mustBeOK fails if at least one of the given streams is not empty or if the
// exit code is not zero.
func mustBeOK(exitCode int, stdout, stderr []byte) error {
	_, err := unwrapStdout(exitCode, stdout, stderr)
	if err != nil {
		return err
	}
	if len(stderr) > 0 {
		return errors.New("unexpected standard output: " + string(stderr))
	}
	return nil
}

// filePaths returns the file paths matching the given pattern.
// If pattern is "", pattern matching is always assumed true.
func filePaths(pattern string) ([]string, error) {
	var paths []string
	var re *regexp.Regexp
	if pattern != "" {
		re = regexp.MustCompile(pattern)
	}
	err := filepath.Walk("testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path[0] == '.' {
			return nil
		}
		switch filepath.Ext(path) {
		case ".go", ".script", ".html", ".css", ".js", ".json", ".md":
		default:
			return nil
		}
		if re != nil && !re.MatchString(path) {
			return nil
		}
		if strings.Contains(path, ".dir"+string(filepath.Separator)) {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

// goldenCompare compares the golden file related to the given path with the
// given data.
func goldenCompare(testPath string, got []byte) error {
	ext := filepath.Ext(testPath)
	goldenPath := strings.TrimSuffix(testPath, ext) + ".golden"
	goldenData, err := ioutil.ReadFile(goldenPath)
	if err != nil {
		return err
	}
	// Remove everything after "//".
	goldenData = regexp.MustCompile(`(?m)^//.*$`).ReplaceAll(goldenData, []byte{})
	expected := bytes.TrimSpace(goldenData)
	got = bytes.TrimSpace(got)
	// Compare all lines, finding all differences.
	{
		expectedLines := strings.Split(string(expected), "\n")
		gotLines := strings.Split(string(got), "\n")
		numLines := len(expectedLines)
		// Find the minimum number of lines.
		if len(gotLines) < numLines {
			numLines = len(gotLines)
		}
		for i := 0; i < numLines; i++ {
			if expectedLines[i] != gotLines[i] {
				return fmt.Errorf("difference at line %d\nexpecting:  %q\ngot:        %q.\n\nFull output: \n------------------------\n%s", i+1, expectedLines[i], gotLines[i], got)
			}
		}
		if len(expectedLines) != len(gotLines) {
			err := fmt.Sprintf("expecting an output of %d lines, got %d lines\n", len(expectedLines), len(gotLines))
			if len(expectedLines) > len(gotLines) {
				err += "expected lines (not returned by the test): \n"
				for i := len(gotLines); i < len(expectedLines); i++ {
					err += fmt.Sprintf("> " + expectedLines[i] + "\n")
				}
			}
			if len(expectedLines) < len(gotLines) {
				err += "additional lines returned by the test (not expected): \n"
				for i := len(expectedLines); i < len(gotLines); i++ {
					err += fmt.Sprintf("> " + gotLines[i] + "\n")
				}
			}
			return errors.New(err)
		}
	}
	// Make an additional compare: any difference not catched by the previous
	// check gets caught here.
	if bytes.Compare(expected, got) != 0 {
		return fmt.Errorf("\n\nexpecting:  %s\ngot:        %s", expected, got)
	}
	return nil
}

// isBuildConstraints reports whether line is a build contrain, as specified at
// https://golang.org/pkg/go/build/#hdr-Build_Constraints.
func isBuildConstraints(line string) bool {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "//") {
		return false
	}
	line = strings.TrimPrefix(line, "//")
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, "+build")
}

// readMode reports the mode (and any options) associated to src.
//
// Mode is specified in programs and scripts using a comment line (starting with
// "//"):
//
//  // mode
//
// templates must encapsulate mode in a line containing just a comment:
//
//  {# readMode #}
//
// After the mode keyword, some arguments may be provided using the syntax
// accepted by function flag.Parse.
//
// 	// run -time 10s
//
// As a special case, the 'skip' comment can be followed by any sequence of
// characters (after a whitespace character) that will be ignored. This is
// useful to put an inline comment to the "skip" comment, explaining the reason
// why a given test cannot be run. For example:
//
//  // skip because feature X is not supported
//  // skip : enable when bug Y will be fixed
//
func readMode(src []byte, ext string) (string, []string, error) {
	if bytes.HasPrefix(src, BOM) {
		src = src[len(BOM):]
	}
	switch ext {
	case ".go", ".script":
		for _, l := range strings.Split(string(src), "\n") {
			l = strings.TrimSpace(l)
			if l == "" {
				continue
			}
			if isBuildConstraints(l) {
				continue
			}
			if !strings.HasPrefix(l, "//") {
				return "", nil, fmt.Errorf("not a valid directive: %s", l)
			}
			l = strings.TrimPrefix(l, "//")
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "skip ") {
				return "skip", []string{}, nil
			}
			s := strings.Split(l, " ")
			return s[0], s[1:], nil
		}
	default:
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
					return "skip", []string{}, nil
				}
				s := strings.Split(l, " ")
				return s[0], s[1:], nil
			}
			return "", nil, fmt.Errorf("not a valid directive: %s", l)
		}
	}
	return "", nil, errors.New("mode not specified")
}

// runGc runs a Go program using gc and returns its output.
func runGc(path string) (int, []byte, []byte, error) {
	if ext := filepath.Ext(path); ext != ".go" {
		return 0, nil, nil, errors.New("unsupported ext " + ext)
	}
	tmpDir, err := ioutil.TempDir("", "scriggo-gc")
	if err != nil {
		return 0, nil, nil, err
	}
	// Create a temporary directory.
	err = os.MkdirAll(tmpDir, 0755)
	if err != nil {
		return 0, nil, nil, err
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()
	// Copy the test source into the temporary directory.
	src, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, nil, nil, err
	}
	err = ioutil.WriteFile(filepath.Join(tmpDir, "main.go"), src, 0664)
	if err != nil {
		return 0, nil, nil, err
	}
	scriggoAbsPath, err := filepath.Abs("../../../scriggo")
	if err != nil {
		return 0, nil, nil, err
	}
	// Copy the "testpkg" module into the directory.
	{
		data, err := ioutil.ReadFile("testpkg/testpkg.go")
		if err != nil {
			return 0, nil, nil, err
		}
		err = os.MkdirAll(filepath.Join(tmpDir, "testpkg"), 0755)
		if err != nil {
			return 0, nil, nil, err
		}
		err = ioutil.WriteFile(filepath.Join(tmpDir, "testpkg", "testpkg.go"), data, 0664)
		if err != nil {
			return 0, nil, nil, err
		}
		goMod := strings.Join([]string{
			`module testpkg`,
			`replace github.com/open2b/scriggo => ` + scriggoAbsPath,
			`require github.com/open2b/scriggo v0.0.0`,
		}, "\n")
		err = ioutil.WriteFile(filepath.Join(tmpDir, "testpkg", "go.mod"), []byte(goMod), 0664)
		if err != nil {
			return 0, nil, nil, err
		}
	}
	// Create a "go.mod" file inside the testing directory.
	{
		data := strings.Join([]string{
			`module scriggo-gc-test`,
			`replace testpkg => ./testpkg`,
			`replace github.com/open2b/scriggo => ` + scriggoAbsPath,
		}, "\n")
		err := ioutil.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(data), 0664)
		if err != nil {
			return 0, nil, nil, err
		}
	}
	// Create a "go.sum" file inside the testing directory.
	{
		cmd := exec.Command("go", "mod", "tidy")
		cmd.Dir = tmpDir
		stdout := bytes.Buffer{}
		stderr := bytes.Buffer{}
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				return ee.ProcessState.ExitCode(), stdout.Bytes(), stderr.Bytes(), nil
			}
			return 0, nil, nil, err
		}
	}
	// Build the test source.
	{
		cmd := exec.Command("go", "build", "-o", "main.exe", "main.go")
		cmd.Dir = tmpDir
		stdout := bytes.Buffer{}
		stderr := bytes.Buffer{}
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				return ee.ProcessState.ExitCode(), stdout.Bytes(), stderr.Bytes(), nil
			}
			return 0, nil, nil, err
		}
	}
	// Run the test just compiled.
	{
		cmd := exec.Command(filepath.Join(tmpDir, "main.exe"))
		cmd.Dir = tmpDir
		stdout := bytes.Buffer{}
		stderr := bytes.Buffer{}
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				return ee.ProcessState.ExitCode(), stdout.Bytes(), stderr.Bytes(), nil
			}
			return 0, nil, nil, err
		}
		return 0, stdout.Bytes(), stderr.Bytes(), nil
	}
}

// goBaseVersion returns the go base version for v.
//
//		1.15.5 -> 1.15
//
func goBaseVersion(v string) string {
	// Taken from cmd/scriggo/util.go.
	if strings.HasPrefix(v, "devel ") {
		v = v[len("devel "):]
		if i := strings.Index(v, "-"); i >= 0 {
			v = v[:i]
		}
	}
	if i := strings.Index(v, "beta"); i >= 0 {
		v = v[:i]
	}
	if i := strings.Index(v, "rc"); i >= 0 {
		v = v[:i]
	}
	v = v[4:]
	f, err := strconv.ParseFloat(v, 32)
	if err != nil {
		panic(err)
	}
	f = math.Floor(f)
	next := int(f)
	return fmt.Sprintf("go1.%d", next)
}

// test execute test on the given source code with the specified mode.
// When a test fails, test calls os.Exit with an non-zero error code, unless
// keepTestingOnFail is set.
func test(src []byte, mode, filePath string, opts []string, keepTestingOnFail bool) error {

	defer func() {
		if r := recover(); r != nil {
			absPath, _ := filepath.Abs(filePath)
			_, _ = fmt.Fprintf(os.Stderr, "\n%s: %s\n", absPath, r)
			if keepTestingOnFail {
				return
			}
			os.Exit(1)
		}
	}()

	ext := filepath.Ext(filePath)

	switch mode {
	case "build", "compile":
		return mustBeOK(cmd(src, opts, "build", ext))
	case "errorcheck":
		return errorcheck(src, filePath, opts)
	case "paniccheck":
		switch ext {
		case ".go":
		case ".script":
			return fmt.Errorf("unsupported mode 'paniccheck' for scripts")
		default:
			return fmt.Errorf("unsupported mode 'paniccheck' for templates")
		}
		exitCode, _, stderr := cmd(src, opts, "run", ".go")
		if exitCode == 0 {
			return errors.New("expected panic, got exit code == 0 (success)")
		}
		panicBody := bytes.Index(stderr, []byte("\n\ngoroutine "))
		if panicBody == -1 {
			return fmt.Errorf("expected panic on stderr, got %q", stderr)
		}
		panicHead := stderr[:panicBody]
		return goldenCompare(filePath, panicHead)
	case "run":
		switch ext {
		case ".go":
			scriggoExitCode, scriggoStdout, scriggoStderr := cmd(src, opts, "run", ".go")
			gcExitCode, gcStdout, gcStderr, err := runGc(filePath)
			if err != nil {
				return err
			}
			formatError := func(msg string) error {
				var s strings.Builder
				s.WriteString(fmt.Sprintf("[ Scriggo exit code ] %d\n", scriggoExitCode))
				s.WriteString(fmt.Sprintf("[ Scriggo stdout    ] '%s'\n", scriggoStdout))
				s.WriteString(fmt.Sprintf("[ Scriggo stderr    ] '%s'\n", scriggoStderr))
				s.WriteString(fmt.Sprintf("[ gc exit code      ] %d\n", gcExitCode))
				s.WriteString(fmt.Sprintf("[ gc stdout         ] '%s'\n", gcStdout))
				s.WriteString(fmt.Sprintf("[ gc stderr         ] '%s'\n", gcStderr))
				return fmt.Errorf("%s\n%s", msg, s.String())
			}
			if scriggoExitCode != 0 && gcExitCode != 0 {
				return formatError("scriggo and gc returned a non-zero exit code")
			}
			if scriggoExitCode != 0 {
				return formatError("scriggo returned a non-zero exit code, while gc succeded")
			}
			if gcExitCode != 0 {
				return formatError("gc returned a non-zero exit code, while Scriggo succeded")
			}
			if bytes.Compare(scriggoStdout, gcStdout) != 0 || bytes.Compare(scriggoStderr, gcStderr) != 0 {
				return formatError("Scriggo and gc returned two different stdout/stderr")
			}
			return nil
		case ".script":
			out, err := unwrapStdout(cmd(src, opts, "run", ".script"))
			if err != nil {
				return err
			}
			return goldenCompare(filePath, out)
		}
	case "render":
		out, err := unwrapStdout(cmd(src, opts, "run", ext))
		if err != nil {
			return err
		}
		return goldenCompare(filePath, out)
	case "rundir":
		if ext == ".go" {
			dirPath := strings.TrimSuffix(filePath, ".go") + ".dir"
			if _, err := os.Stat(dirPath); err != nil {
				return err
			}
			out, err := unwrapStdout(cmd(nil, opts, "rundir", ".go", dirPath))
			if err != nil {
				return err
			}
			return goldenCompare(filePath, out)
		}
	case "renderdir":
		dirPath := strings.TrimSuffix(filePath, ext) + ".dir"
		out, err := unwrapStdout(cmd(nil, opts, "rundir", ext, dirPath))
		if err != nil {
			return err
		}
		return goldenCompare(dirPath, out)
	}

	return fmt.Errorf("unsupported mode %q for extension %q", mode, ext)
}

// unwrapStdout unwraps the given streams returning the stdout. Returns an
// error if stderr is not empty or if exit code is not 0.
func unwrapStdout(exitCode int, stdout, stderr []byte) ([]byte, error) {
	if exitCode != 0 {
		return nil, fmt.Errorf("exit code is %d, should be zero. stderr: %s", exitCode, stderr)
	}
	if len(stderr) > 0 {
		return nil, errors.New("unexpected standard error: " + string(stderr))
	}
	return stdout, nil
}
