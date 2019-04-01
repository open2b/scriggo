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
	"strconv"
	"strings"
	"sync"

	"scrigo"
	"scrigo/parser"
)

var packages map[string]*parser.GoPackage

type outputType uint8

const (
	outputOk outputType = iota
	outputParseCheckerError
	outputRenderingError
)

type output struct {
	outputType outputType
	msg        string
}

func (o output) String() string {
	if o.outputType == outputParseCheckerError {
		return "parsing or checking error: " + o.msg
	}
	if o.outputType == outputRenderingError {
		return "rendering error: " + o.msg
	}
	return o.msg
}

func runScrigoAndGetOutput(src []byte) output {
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

	compiler := scrigo.NewCompiler(nil, packages)
	program, err := compiler.Compile(bytes.NewBuffer(src))
	if err != nil {
		msg := err.Error()
		if msg[0] == ':' {
			return output{
				outputType: outputParseCheckerError,
				msg:        msg[1:],
			}
		}
		return output{
			outputType: outputParseCheckerError,
			msg:        msg,
		}

	}
	err = scrigo.Execute(program)

	if err != nil {
		msg := err.Error()
		if msg[0] == ':' {
			return output{
				outputType: outputRenderingError,
				msg:        msg[1:],
			}
		}
		return output{
			outputType: outputRenderingError,
			msg:        msg,
		}
	}
	writer.Close()
	return output{
		outputType: outputOk,
		msg:        <-out,
	}
}

var golangErrorReg = regexp.MustCompile(`[\w\. ]+:(\d+):(\d+):\s(.*)`)

func extractInfosFromGoErrorMessage(msg string) (int, int, string) {
	match := golangErrorReg.FindStringSubmatch(msg)
	line, _ := strconv.Atoi(match[1])
	column, _ := strconv.Atoi(match[2])
	errorMsg := match[3]
	errorMsg = strings.Replace(errorMsg, "syntax error: ", "", 1)
	return line, column, errorMsg
}

func runGoAndGetOutput(src []byte) output {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "scrigotesting")
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
	if golangErrorReg.MatchString(out) {
		line, column, errorMsg := extractInfosFromGoErrorMessage(out)
		return output{
			outputType: outputParseCheckerError,
			msg:        fmt.Sprintf("%d:%d: %s", line, column, errorMsg),
		}

	}
	return output{
		outputType: outputOk,
		msg:        out,
	}
}

const testsDir = "sources"

func fatal(a interface{}) {
	log.Fatalf("fatal error: %s", a)
}

func main() {
	verbose := false
	mustRecover := true
	pattern := ""
	for i, arg := range os.Args {
		if arg == "-v" || arg == "--verbose" {
			verbose = true
		}
		if arg == "-n" || arg == "--no-recover" {
			mustRecover = false
		}
		if arg == "-h" || arg == "--help" {
			fmt.Printf("Usage: %s [--no-recover|-n] [--verbose|-v] [--help|-h] [--pattern|-p PATTERN]\n", os.Args[0])
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
		fmt.Print("comparing Scrigo with Go..")
	}
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
				scrigoOut := runScrigoAndGetOutput(src)
				goOut := runGoAndGetOutput(src)
				if (scrigoOut.outputType != outputOk || goOut.outputType != outputOk) && (dir.Name() != "errors") {
					fmt.Printf("\nTest %q returned an error, but source is not inside 'errors' directory\n", path)
					fmt.Printf("\nERROR on %q\n\tGo output:      %q\n\tScrigo output:  %q\n", path, goOut, scrigoOut)
					return
				}
				if (scrigoOut.outputType == outputOk) && (goOut.outputType == outputOk) && (dir.Name() == "errors") {
					fmt.Printf("\nTest %q should return error (is inside 'errors' dir), but it doesn't\n", path)
					return
				}
				if scrigoOut == goOut {
					if verbose {
						fmt.Println("OK!")
					}
				} else {
					fmt.Printf("\nERROR on %q\n\tGo output:      %q\n\tScrigo output:  %q\n", path, goOut, scrigoOut)
				}
			}()
		}
	}
	if !verbose {
		fmt.Print("done!\n")
	}
}
