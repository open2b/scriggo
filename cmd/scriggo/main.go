// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rogpeppe/go-internal/modfile"
)

func main() {

	flag.Usage = commandsHelp["scriggo"]

	// No command provided.
	if len(os.Args) == 1 {
		flag.Usage()
		exit(0)
		return
	}

	cmdArg := os.Args[1]

	// Used by flag.Parse.
	os.Args = append(os.Args[:1], os.Args[2:]...)

	cmd, ok := commands[cmdArg]
	if !ok {
		stderr(
			fmt.Sprintf("scriggo %s: unknown command", cmdArg),
			`Run 'scriggo help' for usage.`,
		)
		exit(1)
		return
	}
	cmd()
}

// TestEnvironment is true when testing the scriggo command, false otherwise.
var TestEnvironment = false

// exit causes the current program to exit with the given status code. If
// running in a test environment, every exit call is a no-op.
func exit(status int) {
	if !TestEnvironment {
		os.Exit(status)
	}
}

// stderr prints lines on stderr.
func stderr(lines ...string) {
	for _, l := range lines {
		_, _ = fmt.Fprint(os.Stderr, l+"\n")
	}
}

// exitError prints an error message on stderr with a bold red color and exits
// with status code 1.
func exitError(format string, a ...interface{}) {
	msg := fmt.Errorf(format, a...)
	stderr(msg.Error())
	exit(1)
	return
}

// commandsHelp maps a command name to a function that prints the help for
// that command.
var commandsHelp = map[string]func(){
	"scriggo": func() {
		txtToHelp(helpScriggo)
		flag.PrintDefaults()
	},
	// Help topics.
	"Scriggofile": func() {
		txtToHelp(helpScriggofile)
	},

	// Commands helps.
	"bug": func() {
		stderr(
			`usage: scriggo bug`,
			`Bug opens the default browser and starts a new bug report.`,
			`The report includes useful system information.`,
		)
	},
	"embed": func() {
		txtToHelp(helpEmbed)
	},
	"build": func() {
		txtToHelp(helpBuild)
	},
	"install": func() {
		txtToHelp(helpInstall)
	},
	"stdlib": func() {
		stderr(
			`usage: scriggo stdlib`,
			``,
			`Stdlib prints to the standard output the paths of the packages of the Go`,
			`standard library imported by the instruction 'IMPORT STANDARD LIBRARY' in the`,
			`Scriggofile.`)

	},
	"version": func() {
		stderr(
			`usage: scriggo version`,
		)
	},
}

// commands maps a command name to a function that executes that command.
// Commands are called by command-line using:
//
//		scriggo command
//
var commands = map[string]func(){
	"bug": func() {
		flag.Usage = commandsHelp["bug"]
		panic("TODO: not implemented") // TODO(Gianluca): to implement.
	},
	"build": func() {
		flag.Usage = commandsHelp["install"]
		work := flag.Bool("work", false, "print the name of the temporary work directory and do not delete it when exiting.")
		verbose := flag.Bool("v", false, "print the names of packages as the are imported.")
		flag.Parse()
		if len(flag.Args()) == 0 {
			// No arguments provided: this is not an error.
			flag.Usage()
			return
		}
		if len(flag.Args()) > 1 {
			flag.Usage()
			exitError(`bad number of arguments`)
		}
		err := build(false, *work, *verbose)
		if err != nil {
			exitError("%s", err)
		}
		exit(0)
	},
	"install": func() {
		flag.Usage = commandsHelp["install"]
		verbose := flag.Bool("v", false, "print the names of packages as the are imported.")
		flag.Parse()
		if len(flag.Args()) == 0 {
			// No arguments provided: this is not an error.
			flag.Usage()
			return
		}
		if len(flag.Args()) > 1 {
			flag.Usage()
			exitError(`bad number of arguments`)
		}
		err := build(true, false, *verbose)
		if err != nil {
			exitError("%s", err)
		}
		exit(0)
	},
	"embed": func() {
		flag.Usage = commandsHelp["embed"]
		output := flag.String("o", "", "write the source to the named file instead of stdout.")
		verbose := flag.Bool("v", false, "print the names of packages as they are imported.")
		flag.Parse()
		if len(flag.Args()) == 0 {
			// No arguments provided: this is not an error.
			flag.Usage()
			return
		}
		if len(flag.Args()) > 1 {
			flag.Usage()
			exitError(`bad number of arguments`)
		}
		out, err := getOutputFlag(*output)
		if err != nil {
			exitError("%s", err)
		}
		if out != nil {
			defer func() {
				if err := out.Close(); err != nil {
					exitError("%s", err)
				}
			}()
		}
		err = embed(out, flag.Arg(0), *verbose)
		if err != nil {
			exitError("%s", err)
		}
		exit(0)
	},
	"stdlib": func() {
		flag.Usage = commandsHelp["stdlib"]
		flag.Parse()
		if len(flag.Args()) > 0 {
			flag.Usage()
			exitError(`bad number of arguments`)
		}
		err := stdlib()
		if err != nil {
			exitError("%s", err)
		}
		exit(0)
	},
	"help": func() {
		if len(os.Args) == 1 {
			flag.Usage()
			exit(0)
			return
		}
		topic := os.Args[1]
		help, ok := commandsHelp[topic]
		if !ok {
			_, _ = fmt.Fprintf(os.Stderr, "scriggo help %s: unknown help topic. Run 'scriggo help'.\n", topic)
			exit(1)
			return
		}
		help()
	},
	"version": func() {
		flag.Usage = commandsHelp["version"]
		fmt.Printf("Scriggo module version:            (TODO) \n")     // TODO(Gianluca): use real version.
		fmt.Printf("scriggo tool version:                  (TODO) \n") // TODO(Gianluca): use real version.
		fmt.Printf("Go version used to build scriggo:      %s\n", runtime.Version())
	},
}

// embed executes the sub commands "embed":
//
//		scriggo embed
//
func embed(out io.Writer, path string, verbose bool) error {

	goos := os.Getenv("GOOS")
	if goos == "" {
		goos = runtime.GOOS
	}

	// Read the Scriggofile.
	content, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file %q does not exists", path)
		}
		return err
	}
	defer content.Close()

	// Parse the Scriggofile.
	sf, err := parseScriggofile(content, goos)
	if err != nil {
		return fmt.Errorf("path %q: %s", path, err)
	}
	err = content.Close()
	if err != nil {
		return err
	}

	_, err = renderPackages(out, sf, goos, verbose)

	return err
}

//func moduleAwareMode() bool {
//	s := os.Getenv("GO111MODULE")
//	if s != "auto" {
//		return s == "on"
//	}
//
//}

// build executes the commands "build" and "install".
func build(install bool, work bool, verbose bool) error {

	_, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("scriggo: \"go\" executable file not found in $PATH\nIf not installed, " +
			"download and install Go: https://golang.org/dl/\n")
	}

	goos := os.Getenv("GOOS")
	if goos == "" {
		goos = runtime.GOOS
	}

	path := flag.Arg(0)

	// isFile reports whether path is a regular file.
	var isFile bool

	if modfile.IsDirectoryPath(path) {
		fi, err := os.Stat(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		isFile = fi != nil && fi.Mode().IsRegular()
	}

	// Read the Scriggofile and the go.mod file.
	var scriggofile io.ReadCloser
	var gomod *modfile.File
	var base string

	if isFile {
		scriggofile, err = os.Open(path)
		if err != nil {
			return fmt.Errorf("scriggo: can't open file %s: %s", path, err)
		}
		gomod = &modfile.File{}
		base = strings.TrimSuffix(path, filepath.Ext(path))
	} else {
		// path is a package path.
		pkgDir, modDir, err := packageDirs(path)
		if err != nil {
			return err
		}
		// Read the Scriggofile.
		file := filepath.Join(pkgDir, "Scriggofile")
		scriggofile, err = os.Open(file)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("scriggo: can't load package %s: no Scriggofile in %s", path, pkgDir)
			}
			return err
		}
		// Read the go.mod file.
		file = filepath.Join(modDir, "go.mod")
		data, err := ioutil.ReadFile(file)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("scriggo: cannot find main module; see 'go help modules'")
			}
			return err
		}
		gomod, err = modfile.ParseLax(file, data, nil)
		if err != nil {
			return err
		}
		// Make absolute the relative rewrite paths.
		for _, replace := range gomod.Replace {
			if p := replace.New.Path; modfile.IsDirectoryPath(p) && !filepath.IsAbs(p) {
				filepath.Join(modDir, replace.New.Path)
			}
		}
		base = filepath.Base(pkgDir)
	}
	defer scriggofile.Close()

	// Parse the Scriggofile.
	sf, err := parseScriggofile(scriggofile, goos)
	if err != nil {
		return err
	}
	err = scriggofile.Close()
	if err != nil {
		return err
	}

	// Create a temporary work directory with the sources of the interpreter.
	// If the options "work" is given, the work directory name will be printed
	// and it will not be deleted after the installation.
	workDir, err := ioutil.TempDir("", "scriggo-build")
	if err != nil {
		return err
	}
	if work {
		_, _ = fmt.Fprintf(os.Stderr, "WORK=%s\n", workDir)
	} else {
		defer func() {
			err = os.RemoveAll(workDir)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scriggo: can't delete the temporary work directory: %s", err)
			}
		}()
	}

	dir := filepath.Join(workDir, base)

	err = os.MkdirAll(dir, dirPerm)
	if err != nil {
		return fmt.Errorf("scriggo: can't make work directory: %s", err)
	}

	// Run in module-aware mode.
	if err = os.Setenv("GO111MODULE", "on"); err != nil {
		return fmt.Errorf("scriggo: can't set environment variable \"GO111MODULE\" to \"on\": %s", err)
	}

	// Create the package declarations file.
	packagesPath := filepath.Join(dir, "packages.go")
	fi, err := os.OpenFile(packagesPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, filePerm)
	if err != nil {
		return err
	}
	defer fi.Close()
	_, err = renderPackages(fi, sf, goos, verbose)
	if err != nil {
		return fmt.Errorf("scriggo: can't render packages: %s", err)
	}
	err = fi.Close()
	if err != nil {
		return fmt.Errorf("scriggo: can't render packages: %s", err)
	}

	// Create the go.mod file.
	gomod.AddModuleStmt("open2b.scriggo/" + base)
	{
		// TODO(marco): to remove.
		goPaths := strings.Split(os.Getenv("GOPATH"), string(os.PathListSeparator))
		if len(goPaths) == 0 {
			panic("scriggo: empty gopath not supported")
		}
		scriggoPath := filepath.Join(goPaths[0], "src/scriggo")
		err = gomod.AddReplace("scriggo", "", scriggoPath, "")
		if err != nil {
			panic("scriggo: can't create go.mod: %s")
		}
	}
	data, err := gomod.Format()
	if err != nil {
		return fmt.Errorf("scriggo: can't create go.mod: %s", err)
	}
	err = ioutil.WriteFile(filepath.Join(dir, "go.mod"), data, filePerm)
	if err != nil {
		return fmt.Errorf("scriggo: %s", err)
	}

	// Create the other installer files main file.
	mainPath := filepath.Join(dir, "main.go")
	err = ioutil.WriteFile(mainPath, makeInterpreterSource(sf.target), filePerm)
	if err != nil {
		return fmt.Errorf("scriggo: can't create go.mod: %s", err)
	}

	// Build or install the package.
	command := "build"
	if install {
		command = "install"
	}
	cmd := exec.Command("go", command)
	cmd.Dir = dir
	stderr := bytes.Buffer{}
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("go %s: %s", command, &stderr)
	}

	return nil
}

func stdlib() (err error) {
	for _, path := range stdlibPaths {
		_, err = fmt.Println(path)
		if err != nil {
			break
		}
	}
	return err
}

func packageDirs(path string) (string, string, error) {
	type jsonPackage struct {
		Dir        string
		ImportPath string
		Incomplete bool
	}
	// Read package dir.
	cmd := exec.Command("go", "list", "-e", "-json", "-find=true", path)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return "", "", errors.New(stderr.String())
		}
		return "", "", err
	}
	var packages []*jsonPackage
	for dec := json.NewDecoder(stdout); dec.More(); {
		pkg := &jsonPackage{}
		err := dec.Decode(pkg)
		if err != nil {
			return "", "", fmt.Errorf("can't read packages from go list: %v", err)
		}
		packages = append(packages, pkg)
	}
	if len(packages) == 0 || packages[0].Incomplete {
		// Try
		// Call go build to get the error message.
		cmd = exec.Command("go", "build", "-n", path)
		stderr := &bytes.Buffer{}
		cmd.Stderr = stderr
		_ = cmd.Run()
		return "", "", errors.New(stderr.String())
	}
	if len(packages) > 1 {
		err := fmt.Sprintf("too many packages for %q:", path)
		for _, p := range packages {
			err += "\n        " + p.ImportPath
		}
		return "", "", errors.New(err)
	}
	pkg := packages[0]
	stdout.Reset()
	stderr.Reset()
	// Read module dir.
	cmd = exec.Command("go", "list", "-m", "-json")
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	env := os.Environ()
	for i, ev := range env {
		if strings.HasPrefix(ev, "GO111MODULE=") {
			cmd.Env = env
			cmd.Env[i] = "GO111MODULE=on"
			break
		}
	}
	if cmd.Env == nil {
		cmd.Env = append(env, "GO111MODULE=on")
	}
	cmd.Dir = pkg.Dir
	err = cmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return "", "", errors.New(stderr.String())
		}
		return "", "", err
	}
	type jsonModule struct {
		Dir string
	}
	dec := json.NewDecoder(stdout)
	mod := &jsonModule{}
	err = dec.Decode(mod)
	if err != nil {
		return "", "", fmt.Errorf("can't read module from go list: %v", err)
	}
	return pkg.Dir, mod.Dir, nil
}

// stdlibPaths contains the paths of the packages of the Go standard library
// except the packages "database", "plugin", "testing", "runtime/cgo",
// "syscall", "unsafe" and their sub packages.
var stdlibPaths = []string{
	"archive/tar",
	"archive/zip",
	"bufio",
	"bytes",
	"compress/bzip2",
	"compress/flate",
	"compress/gzip",
	"compress/lzw",
	"compress/zlib",
	"container/heap",
	"container/list",
	"container/ring",
	"context",
	"crypto",
	"crypto/aes",
	"crypto/cipher",
	"crypto/des",
	"crypto/dsa",
	"crypto/ecdsa",
	"crypto/elliptic",
	"crypto/hmac",
	"crypto/md5",
	"crypto/rand",
	"crypto/rc4",
	"crypto/rsa",
	"crypto/sha1",
	"crypto/sha256",
	"crypto/sha512",
	"crypto/subtle",
	"crypto/tls",
	"crypto/x509",
	"crypto/x509/pkix",
	"debug/dwarf",
	"debug/elf",
	"debug/gosym",
	"debug/macho",
	"debug/pe",
	"debug/plan9obj",
	"encoding",
	"encoding/ascii85",
	"encoding/asn1",
	"encoding/base32",
	"encoding/base64",
	"encoding/binary",
	"encoding/csv",
	"encoding/gob",
	"encoding/hex",
	"encoding/json",
	"encoding/pem",
	"encoding/xml",
	"errors",
	"expvar",
	"flag",
	"fmt",
	"go/ast",
	"go/build",
	"go/constant",
	"go/doc",
	"go/format",
	"go/importer",
	"go/parser",
	"go/printer",
	"go/scanner",
	"go/token",
	"go/types",
	"hash",
	"hash/adler32",
	"hash/crc32",
	"hash/crc64",
	"hash/fnv",
	"html",
	"html/template",
	"image",
	"image/color",
	"image/color/palette",
	"image/draw",
	"image/gif",
	"image/jpeg",
	"image/png",
	"index/suffixarray",
	"io",
	"io/ioutil",
	"log",
	"log/syslog",
	"math",
	"math/big",
	"math/bits",
	"math/cmplx",
	"math/rand",
	"mime",
	"mime/multipart",
	"mime/quotedprintable",
	"net",
	"net/http",
	"net/http/cgi",
	"net/http/cookiejar",
	"net/http/fcgi",
	"net/http/httptest",
	"net/http/httptrace",
	"net/http/httputil",
	"net/http/pprof",
	"net/mail",
	"net/rpc",
	"net/rpc/jsonrpc",
	"net/smtp",
	"net/textproto",
	"net/url",
	"os",
	"os/exec",
	"os/signal",
	"os/user",
	"path",
	"path/filepath",
	"reflect",
	"regexp",
	"regexp/syntax",
	"runtime",
	"runtime/debug",
	"runtime/pprof",
	"runtime/race",
	"runtime/trace",
	"sort",
	"strconv",
	"strings",
	"sync",
	"sync/atomic",
	"text/scanner",
	"text/tabwriter",
	"text/template",
	"text/template/parse",
	"time",
	"unicode",
	"unicode/utf16",
	"unicode/utf8",
}
