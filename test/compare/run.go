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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rogpeppe/go-internal/imports"
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

	if len(flag.Args()) > 0 {
		fmt.Fprintf(os.Stderr, "too many arguments on the command line\n")
		flag.Usage()
		os.Exit(1)
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

	if *verbose || *stat {
		fmt.Printf("%d tests found (including skipped and incompatible ones)\n", len(filepaths))
	}

	wg := sync.WaitGroup{}
	queue := make(chan bool, *parallel)

	countTotal := int64(0)
	countSkipped := int64(0)
	countIncompatible := int64(0)

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
			// Print output before running the test.
			if *stat && !*verbose {
				perc := strconv.Itoa(int(math.Floor(float64(countTotal) / float64(len(filepaths)) * 100)))
				fmt.Printf("\r%s%% (test %d/%d)", perc, countTotal, len(filepaths))
			}
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
			if !shouldBuild(src) {
				atomic.AddInt64(&countIncompatible, 1)
				<-queue
				wg.Done()
				return
			}
			mode, opts := readMode(src, ext)
			// Skip or run the test.
			if mode == "skip" {
				atomic.AddInt64(&countSkipped, 1)
			} else {
				test(src, opts, path, mode, ext, *keepTestingOnFail)
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
		totalTime := time.Now().Sub(start).Truncate(time.Duration(time.Millisecond))
		if countIncompatible > 0 {
			fmt.Printf("\ndone!   %d tests executed, %d tests skipped (and %d not compatible) in %s\n", countExecuted, countSkipped, countIncompatible, totalTime)
		} else {
			fmt.Printf("\ndone!   %d tests executed, %d tests skipped in %s\n", countExecuted, countSkipped, totalTime)
		}
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
func cmd(stdin []byte, opts []string, args ...string) (int, []byte, []byte) {
	cmdArgs := []string{}
	cmdArgs = append(cmdArgs, opts...)
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command("./cmd/cmd", cmdArgs...)
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = bytes.NewReader(stdin)
	err := cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ProcessState.ExitCode(), stdout.Bytes(), stderr.Bytes()
		}
	}
	return 0, stdout.Bytes(), stderr.Bytes()
}

// errorcheck run test with mode 'errorcheck' on the given source code.
func errorcheck(src []byte, ext string, opts []string) {
	testWithoutErrorLines(src, ext, opts)
	tests := differentiateSources(string(src))
	if len(tests) == 0 {
		panic("no // ERROR comments found")
	}
	for _, test := range tests {
		// Get output from program/script/templates and check if it matches with
		// expected error.
		arg := map[string]string{
			".go":   "run program",
			".sgo":  "run script",
			".html": "render html",
		}[ext]
		exitCode, stdout, stderr := cmd([]byte(test.src), opts, arg)
		if exitCode == 0 {
			panic(fmt.Errorf("expecting error '%s' but exit code is 0", test.err))
		}
		if len(stdout) > 0 {
			panic("stdout should be empty")
		}
		if len(stderr) == 0 {
			panic(fmt.Errorf("expected error '%s', got nothing", test.err))
		}
		re := regexp.MustCompile(test.err)
		stderr = []byte(removePrefixFromError(string(stderr)))
		if !re.Match(stderr) {
			panic(fmt.Errorf("error does not match:\n\n\texpecting:  %s\n\tgot:        %s", test.err, stderr))
		}
	}
}

// testWithoutErrorLines runs or renders src (using the given options) removing
// all the '// ERROR' comments.
func testWithoutErrorLines(src []byte, ext string, opts []string) {
	arg := map[string]string{
		".go":   "run program",
		".sgo":  "run script",
		".html": "render html",
	}[ext]
	linesWithoutError := []byte{}
	for _, line := range bytes.Split(src, []byte{'\n'}) {
		if !bytes.Contains(line, []byte("// ERROR ")) {
			line = append(line, '\n')
			linesWithoutError = append(linesWithoutError, line...)
		}
	}
	exitCode, _, stderr := cmd([]byte(linesWithoutError), opts, arg)
	if exitCode != 0 {
		panic(fmt.Errorf("unexpected error (maybe you forgot to add an // ERROR comment or the test has some problem): '%s'", stderr))
	}
}

// mustBeOK fails if at least one of the given streams is not empty or if the
// exit code is not zero.
func mustBeOK(exitCode int, stdout, stderr []byte) {
	_ = unwrapStdout(exitCode, stdout, stderr)
	if len(stderr) > 0 {
		panic("unexpected standard output: " + string(stderr))
	}
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
	// Remove everything after "//".
	goldenData = regexp.MustCompile(`(?m)^//.*$`).ReplaceAll(goldenData, []byte{})
	expected := bytes.TrimSpace(goldenData)
	got = bytes.TrimSpace(got)
	if bytes.Compare(expected, got) != 0 {
		panic(fmt.Errorf("\n\nexpecting:  %s\ngot:        %s", expected, got))
	}
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

// isTestPath reports whether path is a valid test path.
func isTestPath(path string) bool {
	if filepath.Ext(path) != ".go" && filepath.Ext(path) != ".sgo" && filepath.Ext(path) != ".html" {
		return false
	}
	if strings.Contains(path, ".dir"+string(filepath.Separator)) {
		return false
	}
	return true
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
// 	// run -time 10s -mem 34K
//
// As a special case, the 'skip' comment can be followed by any sequence of
// characters (after a whitespace character) that will be ignored. This is
// useful to put an inline comment to the "skip" comment, explaining the reason
// why a given test cannot be run. For example:
//
//  // skip because feature X is not supported
//  // skip : enable when bug Y will be fixed
//
func readMode(src []byte, ext string) (mode string, opts []string) {
	splitOpts := func(s string) (mode string, opts []string) {
		ss := strings.Split(s, " ")
		return ss[0], ss[1:]
	}
	switch ext {
	case ".go", ".sgo":
		for _, l := range strings.Split(string(src), "\n") {
			l = strings.TrimSpace(l)
			if l == "" {
				continue
			}
			if isBuildConstraints(l) {
				continue
			}
			if !strings.HasPrefix(l, "//") {
				panic(fmt.Errorf("not a valid directive: '%s'", l))
			}
			l = strings.TrimPrefix(l, "//")
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "skip ") {
				return "skip", []string{}
			}
			return splitOpts(l)
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
					return "skip", []string{}
				}
				return splitOpts(l)
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
func runGc(path string) (int, []byte, []byte) {
	if ext := filepath.Ext(path); ext != ".go" {
		panic("unsupported ext " + ext)
	}
	tmpDir, err := ioutil.TempDir("", "scriggo-gc")
	if err != nil {
		panic(err)
	}
	// Create a temporary directory.
	err = os.MkdirAll(tmpDir, 0755)
	if err != nil {
		panic(err)
	}
	defer func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			panic(err)
		}
	}()
	// Copy the test source into the temporary directory.
	src, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(filepath.Join(tmpDir, "main.go"), src, 0664)
	if err != nil {
		panic(err)
	}
	// Copy the "testpkg" module into the directory.
	{
		data, err := ioutil.ReadFile("testpkg/testpkg.go")
		if err != nil {
			panic(err)
		}
		err = os.MkdirAll(filepath.Join(tmpDir, "testpkg"), 0755)
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile(filepath.Join(tmpDir, "testpkg", "testpkg.go"), data, 0664)
		if err != nil {
			panic(err)
		}
		goMod := strings.Join([]string{
			`module testpkg`,
		}, "\n")
		err = ioutil.WriteFile(filepath.Join(tmpDir, "testpkg", "go.mod"), []byte(goMod), 0664)
		if err != nil {
			panic(err)
		}
	}
	// Create a "go.mod" file inside the testing directory.
	{
		data := strings.Join([]string{
			`module scriggo-gc-test`,
			`replace testpkg => ./testpkg`,
		}, "\n")
		err := ioutil.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(data), 0664)
		if err != nil {
			panic(err)
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
				return ee.ProcessState.ExitCode(), stdout.Bytes(), stderr.Bytes()
			}
			panic(err)
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
				return ee.ProcessState.ExitCode(), stdout.Bytes(), stderr.Bytes()
			}
			panic(err)
		}
		return 0, stdout.Bytes(), stderr.Bytes()
	}
}

// goBaseVersion returns the go base version for v.
//
//		1.12.5 -> 1.12
//
func goBaseVersion(v string) string {
	// Taken from cmd/scriggo/util.go.
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

var buildTags = map[string]bool{}

func init() {
	// Fill builtags.
	goos := os.Getenv("GOOS")
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := os.Getenv("GOARCH")
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	buildTags = map[string]bool{
		goos:   true,
		goarch: true,
	}
	// Add the build tags from the first version of go to the current one.
	cmd := exec.Command("go", "version")
	stdout := &bytes.Buffer{}
	cmd.Stdout = stdout
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
	rawVersion := strings.Split(stdout.String(), " ")[2]
	rawVersion = strings.TrimSpace(rawVersion)
	goVersion := goBaseVersion(rawVersion)
	subv, err := strconv.Atoi(goVersion[len("go1."):])
	if err != nil {
		panic(err)
	}
	for i := 1; i <= subv; i++ {
		buildTags[fmt.Sprintf("go1.%d", i)] = true
	}
}

// shouldBuild reports whether a given source code should build or not.
func shouldBuild(src []byte) bool {
	return imports.ShouldBuild(src, buildTags)
}

// test execute test on the given source code with the specified mode.
// When a test fails, test calls os.Exit with an non-zero error code, unless
// keepTestingOnFail is set.
func test(src []byte, opts []string, path, mode, ext string, keepTestingOnFail bool) {

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
		mustBeOK(cmd(src, opts, "compile program"))
	case "compile .sgo", "build .sgo":
		mustBeOK(cmd(src, opts, "compile script"))
	case "compile .html", "build .html":
		mustBeOK(cmd(src, opts, "compile html"))

	// Error check.
	case "errorcheck .go", "errorcheck .sgo", "errorcheck .html":
		errorcheck(src, ext, opts)

	// Panic check.
	case "paniccheck .go":
		exitCode, _, stderr := cmd(src, opts, "run program")
		if exitCode == 0 {
			panic("expected panic, got exit code == 0 (success)")
		}
		panicBody := bytes.Index(stderr, []byte("\n\ngoroutine "))
		if panicBody == -1 {
			panic(fmt.Errorf("expected panic on stderr, got %q", stderr))
		}
		panicHead := stderr[:panicBody]
		goldenCompare(path, panicHead)

	// Run or render.
	case "run .go":
		scriggoExitCode, scriggoStdout, scriggoStderr := cmd(src, opts, "run program")
		gcExitCode, gcStdout, gcStderr := runGc(path)
		panic := func(msg string) {
			s := strings.Builder{}
			s.WriteString(fmt.Sprintf("[ Scriggo exit code ] %d\n", scriggoExitCode))
			s.WriteString(fmt.Sprintf("[ Scriggo stdout    ] '%s'\n", scriggoStdout))
			s.WriteString(fmt.Sprintf("[ Scriggo stderr    ] '%s'\n", scriggoStderr))
			s.WriteString(fmt.Sprintf("[ gc exit code      ] %d\n", gcExitCode))
			s.WriteString(fmt.Sprintf("[ gc stdout         ] '%s'\n", gcStdout))
			s.WriteString(fmt.Sprintf("[ gc stderr         ] '%s'\n", gcStderr))
			panic(msg + "\n" + s.String())
		}
		if scriggoExitCode != 0 && gcExitCode != 0 {
			panic("scriggo and gc returned a non-zero exit code")
		}
		if scriggoExitCode != 0 {
			panic("scriggo returned a non-zero exit code, while gc succeded")
		}
		if gcExitCode != 0 {
			panic("gc returned a non-zero exit code, while Scriggo succeded")
		}
		if bytes.Compare(scriggoStdout, gcStdout) != 0 || bytes.Compare(scriggoStderr, gcStderr) != 0 {
			panic("Scriggo and gc returned two different stdout/stderr")
		}
	case "rundir .go":
		dirPath := strings.TrimSuffix(path, ".go") + ".dir"
		if _, err := os.Stat(dirPath); err != nil {
			panic(err)
		}
		goldenCompare(
			path,
			unwrapStdout(
				cmd(nil, opts, "run program directory", dirPath),
			),
		)
	case "run .sgo":
		goldenCompare(
			path,
			unwrapStdout(
				cmd(src, opts, "run script"),
			),
		)
	case "render .html":
		goldenCompare(
			path,
			unwrapStdout(
				cmd(src, opts, "render html"),
			),
		)
	case "renderdir .html":
		goldenCompare(
			path,
			unwrapStdout(
				cmd(nil, opts, "render html directory", strings.TrimSuffix(path, ".html")+".dir"),
			),
		)

	default:
		panic(fmt.Errorf("unsupported mode '%s' for test with extension '%s'", mode, ext))
	}

}

// unwrapStdout unwraps the given streams returning the stdout. Panics if stderr
// is not empty or if exit code is not 0.
func unwrapStdout(exitCode int, stdout, stderr []byte) []byte {
	if exitCode != 0 {
		panic(fmt.Errorf("exit code is %d, should be zero. stderr: %s", exitCode, stderr))
	}
	if len(stderr) > 0 {
		panic("unexpected standard error: " + string(stderr))
	}
	return stdout
}
