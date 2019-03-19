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
	"scrigo/ast"
	"scrigo/parser"
)

var packages map[string]*parser.GoPackage

func runScrigoAndGetOutput(src []byte) string {
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
	r := parser.MapReader{"main": src}
	p := parser.New(r, packages, true)
	tree, err := p.Parse("main", ast.ContextNone)
	if err != nil {
		msg := err.Error()
		if msg[0] == ':' {
			return msg[1:]
		}
		return msg
	}
	pkgs := make(map[string]*scrigo.Package, len(packages))
	for n, pkg := range packages {
		pkgs[n] = &scrigo.Package{Name: pkg.Name, Declarations: pkg.Declarations}
	}
	err = scrigo.RunPackageTree(tree, pkgs)
	if err != nil {
		msg := err.Error()
		if msg[0] == ':' {
			return msg[1:]
		}
		return msg
	}
	writer.Close()
	return <-out
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

func runGoAndGetOutput(src []byte) string {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "scrigotesting")
	if err != nil {
		panic(err)
	}
	defer func() {
		if !strings.HasPrefix(tmpDir, os.TempDir()) {
			panic(fmt.Errorf("invalid tmpDir: %q", tmpDir))
		}
		err := os.RemoveAll(tmpDir)
		if err != nil {
			panic(err)
		}
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
		return fmt.Sprintf("%d:%d: %s", line, column, errorMsg)
	}
	return out
}

const testsDir = "examples"

func fatal(a interface{}) {
	log.Fatalf("fatal error: %s", a)
}

func main() {
	verbose := false
	mustRecover := true
	for _, arg := range os.Args {
		if arg == "-v" || arg == "--verbose" {
			verbose = true
		}
		if arg == "-n" || arg == "--no-recover" {
			mustRecover = false
		}
		if arg == "-h" || arg == "--help" {
			fmt.Printf("Usage: %s [--no-recover|-n] [--verbose|-v] [--help|-h]\n", os.Args[0])
			os.Exit(0)
		}
	}
	testDirs, err := ioutil.ReadDir(testsDir)
	if err != nil {
		fatal(err)
	}
	for _, dir := range testDirs {
		func() {
			defer func() {
				if mustRecover {
					if r := recover(); r != nil {
						fmt.Printf("!!! PANIC !!!: %v\n", r)
					}
				}
			}()
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
				if strings.Contains(path, "_ignore_") {
					continue
				}
				src, err := ioutil.ReadFile(path)
				if err != nil {
					fatal(err)
				}
				if verbose {
					fmt.Printf("---------------------------------------------------\n")
					fmt.Print(path + "...")
				}
				scrigoOut := runScrigoAndGetOutput(src)
				goOut := runGoAndGetOutput(src)
				if scrigoOut == goOut {
					if verbose {
						fmt.Println("OK!")
					}
				} else {
					fmt.Printf("ERROR\n\tGo output:      %q\n\tScrigo output:  %q\n", goOut, scrigoOut)
				}
			}
		}()
	}
}
