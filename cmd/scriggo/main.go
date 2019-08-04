// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	_path "path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rogpeppe/go-internal/modfile"
	"github.com/rogpeppe/go-internal/module"
	"github.com/rogpeppe/go-internal/semver"
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
		f := flag.String("f", "", "path of the Scriggofile.")
		v := flag.Bool("v", false, "print the names of packages as the are imported.")
		x := flag.Bool("x", false, "print the commands.")
		o := flag.String("o", "", "output file.")
		flag.Parse()
		var path string
		switch n := len(flag.Args()); n {
		case 0:
		case 1:
			path = flag.Arg(0)
		default:
			flag.Usage()
			exitError(`bad number of arguments`)
		}
		err := build("build", path, buildFlags{f: *f, work: *work, v: *v, x: *x, o: *o})
		if err != nil {
			exitError("%s", err)
		}
		exit(0)
	},
	"install": func() {
		flag.Usage = commandsHelp["install"]
		work := flag.Bool("work", false, "print the name of the temporary work directory and do not delete it when exiting.")
		f := flag.String("f", "", "path of the Scriggofile.")
		v := flag.Bool("v", false, "print the names of packages as the are imported.")
		x := flag.Bool("x", false, "print the commands.")
		flag.Parse()
		var path string
		switch n := len(flag.Args()); n {
		case 0:
		case 1:
			path = flag.Arg(0)
		default:
			flag.Usage()
			exitError(`bad number of arguments`)
		}
		err := build("install", path, buildFlags{f: *f, work: *work, v: *v, x: *x})
		if err != nil {
			exitError("%s", err)
		}
		exit(0)
	},
	"embed": func() {
		flag.Usage = commandsHelp["embed"]
		f := flag.String("f", "", "path of the Scriggofile.")
		v := flag.Bool("v", false, "print the names of packages as the are imported.")
		x := flag.Bool("x", false, "print the commands.")
		o := flag.String("o", "", "write the source to the named file instead of stdout.")
		flag.Parse()
		var path string
		switch n := len(flag.Args()); n {
		case 0:
		case 1:
			path = flag.Arg(0)
		default:
			flag.Usage()
			exitError(`bad number of arguments`)
		}
		err := embed(path, buildFlags{f: *f, v: *v, x: *x, o: *o})
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
func embed(path string, flags buildFlags) (err error) {

	_, err = exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("scriggo: \"go\" executable file not found in $PATH\nIf not installed, " +
			"download and install Go: https://golang.org/dl/\n")
	}

	goos := os.Getenv("GOOS")
	if goos == "" {
		goos = runtime.GOOS
	}

	// Run in module-aware mode.
	if flags.x {
		_, _ = fmt.Fprintln(os.Stderr, "export GO111MODULE=on")
	}
	if err := os.Setenv("GO111MODULE", "on"); err != nil {
		return fmt.Errorf("scriggo: can't set environment variable \"GO111MODULE\" to \"on\": %s", err)
	}

	var modDir string

	if path == "" {
		// path is current directory.
		modDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("scriggo: can't get current directory: %s", err)
		}
	} else if modfile.IsDirectoryPath(path) {
		// path is a local path.
		fi, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				err = fmt.Errorf("scriggo: directory %s does not exist in:\n\t%s", path, modDir)
			}
			return err
		}
		modDir, err = filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("scriggo: can't get absolute path of %s: %s", path, err)
		}
		if !fi.IsDir() {
			return fmt.Errorf("scriggo: %s is not a directory:\n\t%s", path, modDir)
		}
	} else {
		return fmt.Errorf("scriggo: path, if not empty, must be rooted or must start with '.%c' or '..%c'",
			os.PathSeparator, os.PathSeparator)
	}

	// Get the absolute Scriggofile's path.
	var sfPath string
	if flags.f == "" {
		sfPath = filepath.Join(modDir, "Scriggofile")
	} else {
		sfPath, err = filepath.Abs(flags.f)
		if err != nil {
			return fmt.Errorf("scriggo: can't get absolute path of %s: %s", flags.f, err)
		}
	}

	// Read the Scriggofile.
	scriggofile, err := os.Open(sfPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("scriggo: no Scriggofile in:\n\t%s", sfPath)
		}
		return err
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

	// Create the package declarations file.
	out, err := getOutputFlag(flags.o)
	if err != nil {
		return err
	}
	if out != nil {
		defer func() {
			err = out.Close()
		}()
	}
	err = renderPackages(out, modDir, sf, goos, flags)

	return err
}

type buildFlags struct {
	work, v, x bool
	f, o       string
}

// build executes the commands "build" and "install".
func build(cmd string, path string, flags buildFlags) error {

	_, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("scriggo: \"go\" executable file not found in $PATH\nIf not installed, " +
			"download and install Go: https://golang.org/dl/\n")
	}

	goos := os.Getenv("GOOS")
	if goos == "" {
		goos = runtime.GOOS
	}

	// Create a temporary work directory with the sources of the interpreter.
	// If the options "work" is given, the work directory name will be printed
	// and it will not be deleted after the installation.
	workDir, err := ioutil.TempDir("", "scriggo-build")
	if err != nil {
		return err
	}
	if flags.work || flags.x {
		_, _ = fmt.Fprintf(os.Stderr, "WORK=%s\n", workDir)
	}
	if !flags.work {
		defer func() {
			err = os.RemoveAll(workDir)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "scriggo: can't delete the temporary work directory:\n\t%s", err)
			}
		}()
	}

	// Run in module-aware mode.
	if flags.x {
		_, _ = fmt.Fprintln(os.Stderr, "export GO111MODULE=on")
	}
	if err := os.Setenv("GO111MODULE", "on"); err != nil {
		return fmt.Errorf("scriggo: can't set environment variable \"GO111MODULE\" to \"on\": %s", err)
	}

	var modPath string // modPath is the module's path, it is empty it is local.
	var modVer string  // modVer is the module's version.
	var modDir string  // modDir is the directory of the module's sources.

	if path == "" {
		// path is current directory.
		modDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("scriggo: can't get current directory: %s", err)
		}
	} else if modfile.IsDirectoryPath(path) {
		// path is a local path.
		fi, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				err = fmt.Errorf("scriggo: directory %s does not exist in:\n\t%s", path, modDir)
			}
			return err
		}
		modDir, err = filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("scriggo: can't get absolute path of %s: %s", path, err)
		}
		if !fi.IsDir() {
			return fmt.Errorf("scriggo: %s is not a directory:\n\t%s", path, modDir)
		}
	} else {
		// path is a module path.
		parts := strings.SplitN(path, "@", 2)
		if err := module.CheckPath(parts[0]); err != nil {
			return fmt.Errorf("scriggo: %s", err)
		}
		modPath = parts[0]
		version := "latest"
		if len(parts) == 2 {
			if !semver.IsValid(parts[1]) {
				return fmt.Errorf("scriggo: invalid module version %s", parts[1])
			}
			version = parts[1]
		}
		// Download the module.
		modDir, modVer, err = downloadModule(modPath, version, workDir, flags)
		if err != nil {
			return err
		}
	}

	// Get the absolute Scriggofile's path.
	var sfPath string
	if flags.f == "" {
		sfPath = filepath.Join(modDir, "Scriggofile")
	} else {
		sfPath, err = filepath.Abs(flags.f)
		if err != nil {
			return fmt.Errorf("scriggo: can't get absolute path of %s: %s", flags.f, err)
		}
	}

	// Parse the Scriggofile.
	fi, err := os.Open(sfPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = fmt.Errorf("scriggo: can't find Scriggofile at:\n\t%s", sfPath)
		}
		return err
	}
	defer fi.Close()
	sf, err := parseScriggofile(fi, goos)
	if err != nil {
		return err
	}
	_ = fi.Close()

	// Parse the go.mod file.
	file := filepath.Join(modDir, "go.mod")
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	goMod, err := modfile.Parse(file, data, nil)
	if err != nil {
		return err
	}
	// Make the replace paths as absolute paths.
	for _, replace := range goMod.Replace {
		if newPath := replace.New.Path; modfile.IsDirectoryPath(newPath) && !filepath.IsAbs(newPath) {
			old := replace.Old
			err = goMod.DropReplace(old.Path, old.Version)
			if err != nil {
				return fmt.Errorf("scriggo: can't remove rewrite from go.mod: %s", err)
			}
			var p string
			if modPath == "" {
				p = filepath.Join(modDir, newPath)
			} else {
				p = _path.Join(modPath, newPath)
			}
			err = goMod.AddReplace(old.Path, old.Version, p, modVer)
			if err != nil {
				return fmt.Errorf("scriggo: can't add rewrite to go.mod: %s", err)
			}
		}
	}
	if modPath == "" {
		// Add a replace for the main module.
		mod := goMod.Module.Mod
		err = goMod.AddReplace(mod.Path, mod.Version, modDir, "")
		if err != nil {
			return fmt.Errorf("scriggo: can't add rewrite to go.mod: %s", err)
		}
	}
	goMod.Cleanup()

	var base string
	if modPath == "" {
		base = filepath.Base(modDir)
	} else {
		base = filepath.Base(modPath)
	}

	dir := filepath.Join(workDir, base)

	if flags.x {
		_, _ = fmt.Fprintf(os.Stderr, "mkdir $WORK%c%s\n", os.PathSeparator, base)
		_, _ = fmt.Fprintf(os.Stderr, "chdir $WORK%c%s\n", os.PathSeparator, base)
	}
	err = os.Mkdir(dir, 0777)
	if err != nil {
		return fmt.Errorf("scriggo: can't make directory %s: %s", dir, err)
	}

	// Create the go.mod file.
	goMod.AddModuleStmt("open2b.scriggo/" + base)
	{
		// TODO(marco): remove this block when Scriggo has the definitive path.
		goPaths := strings.Split(os.Getenv("GOPATH"), string(os.PathListSeparator))
		if len(goPaths) == 0 {
			panic("scriggo: empty gopath not supported")
		}
		scriggoPath := filepath.Join(goPaths[0], "src/scriggo")
		err = goMod.AddReplace("scriggo", "", scriggoPath, "")
		if err != nil {
			panic("scriggo: can't create go.mod: %s")
		}
	}
	data, err = goMod.Format()
	if err != nil {
		return fmt.Errorf("scriggo: can't create go.mod: %s", err)
	}
	err = ioutil.WriteFile(filepath.Join(dir, "go.mod"), data, 0666)
	if err != nil {
		return fmt.Errorf("scriggo: %s", err)
	}

	// Create the package declarations file.
	packagesPath := filepath.Join(dir, "packages.go")
	fi, err = os.OpenFile(packagesPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0666)
	if err != nil {
		return err
	}
	defer fi.Close()
	err = renderPackages(fi, dir, sf, goos, flags)
	if err != nil {
		return fmt.Errorf("scriggo: can't render packages: %s", err)
	}
	err = fi.Close()
	if err != nil {
		return fmt.Errorf("scriggo: can't render packages: %s", err)
	}

	// Create the other installer files main file.
	mainPath := filepath.Join(dir, "main.go")
	err = ioutil.WriteFile(mainPath, makeInterpreterSource(sf.target), 0666)
	if err != nil {
		return fmt.Errorf("scriggo: can't create go.mod: %s", err)
	}

	// Build or install the package.
	var args []string
	if cmd == "build" {
		args = []string{"build", "-o", ""}
		if flags.o == "" {
			args[2], err = filepath.Abs(base)
			if err != nil {
				return fmt.Errorf("scriggo: can't get absolute path of %s: %s", base, err)
			}
			if runtime.GOOS == "windows" {
				args[2] += ".exe"
			}
		} else {
			args[2], err = filepath.Abs(flags.o)
			if err != nil {
				return fmt.Errorf("scriggo: can't get absolute path of %s: %s", flags.o, err)
			}
		}
	} else {
		args = []string{"install"}
	}
	if flags.x {
		_, _ = fmt.Fprint(os.Stderr, "go")
		for _, arg := range args {
			_, _ = fmt.Fprint(os.Stderr, " ", arg)
		}
		_, _ = fmt.Fprintln(os.Stderr)
	}
	_, err = execGoCommand(dir, args...)
	if err != nil {
		return fmt.Errorf("go %s: %s", cmd, err)
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

// downloadModule downloads a module, if not cached, given path and version
// and returns the local directory of its source and the version. workDir is
// the working directory and flags are the command flags.
func downloadModule(path, version, workDir string, flags buildFlags) (string, string, error) {

	// Create the go.mod file for 'go download'.
	dir := filepath.Join(workDir, "download")
	sep := string(os.PathSeparator)
	if flags.x {
		_, _ = fmt.Fprintf(os.Stderr, "mkdir -p $WORK%sdownload\n", sep)
	}
	err := os.Mkdir(dir, 0777)
	if err != nil {
		return "", "", fmt.Errorf("scriggo: can't make diretory %s: %s", dir, err)
	}
	goModPath := filepath.Join(dir, "go.mod")
	goModData := "module scriggo.download\nrequire " + path + " " + version
	if flags.x {
		_, _ = fmt.Fprintf(os.Stderr, "cat >$WORK%sdownload%sgo.mod << 'EOF'\n%s\nEOF\n", sep, sep, goModData)
	}
	err = ioutil.WriteFile(goModPath, []byte(goModData), 0666)
	if err != nil {
		return "", "", err
	}

	// Download the module.
	type jsonModule struct {
		Version string
		Dir     string
	}
	if flags.x {
		_, _ = fmt.Fprintf(os.Stderr, "chdir $WORK%sdownload\n", sep)
		_, _ = fmt.Fprintln(os.Stderr, "go mod download -json")
	}
	out, err := execGoCommand(dir, "mod", "download", "-json")
	if err != nil {
		if e, ok := err.(stdError); ok {
			s := e.Error()
			if strings.Contains(s, "go: errors parsing go.mod:\n") {
				if strings.Contains(s, "invalid module version \"latest\":") {
					return "", "", fmt.Errorf("scriggo: can't find module %s", path)
				} else if strings.Contains(s, "invalid module version \""+version+"\":") {
					return "", "", fmt.Errorf("scriggo: can't find version %s of module %s", version, path)
				}
			}
		}
		return "", "", err
	}

	// Read the module's directory.
	dec := json.NewDecoder(out)
	mod := &jsonModule{}
	err = dec.Decode(mod)
	if err != nil {
		return "", "", fmt.Errorf("scriggo: can't read response from 'go mod download': %v", err)
	}

	return mod.Dir, mod.Version, nil
}

type stdError string

func (e stdError) Error() string {
	return string(e)
}

// execGoCommand executes the command 'go' with dir as current directory and
// args as arguments. Returns the standard output if no error occurs.
func execGoCommand(dir string, args ...string) (out io.Reader, err error) {
	if os.Getenv("GO111MODULE") != "on" {
		panic("GO111MODULE must be 'on'")
	}
	cmd := exec.Command("go", args...)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return nil, stdError(stderr.String())
		}
		return nil, err
	}
	return stdout, nil
}

// stdlibPaths contains the paths of the packages of the Go standard library
// except the packages "database", "plugin", "testing", "runtime/cgo",
// "runtime/race",  "syscall", "unsafe" and their sub packages.
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
