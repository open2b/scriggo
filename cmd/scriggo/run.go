// Copyright 2021 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/native"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// run executes the sub command "run":
//
//		scriggo run
//
func run(name string, flags buildFlags) (err error) {

	var fsys fs.FS
	if flags.root == "" {
		fsys = os.DirFS(filepath.Dir(name))
		name = filepath.Base(name)
	} else {
		root, err := filepath.Abs(flags.root)
		if err != nil {
			return err
		}
		nameAbs, err := filepath.Abs(name)
		if err != nil {
			return err
		}
		name, err = filepath.Rel(root, nameAbs)
		if err != nil {
			return err
		}
		fsys = os.DirFS(root)
	}

	// Handle "-format" option.
	if flags.f != "" {
		format, err := parseFormat(flags.f)
		if err != nil {
			return err
		}
		fsys = formatFS{FS: fsys, format: format}
	}

	md := goldmark.New(
		goldmark.WithRendererOptions(html.WithUnsafe()),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithExtensions(extension.GFM))

	opts := &scriggo.BuildOptions{
		AllowGoStmt: true,
		Globals:     globals,
		MarkdownConverter: func(src []byte, out io.Writer) error {
			return md.Convert(src, out)
		},
	}
	opts.Globals["filepath"] = strings.TrimSuffix(name, path.Ext(name))

	// Handle "-const" option.
	if flags.consts != "" {
		err = parseConstants(flags.consts, opts.Globals)
		if err != nil {
			return err
		}
	}

	var start time.Time
	if flags.metrics {
		start = time.Now()
	}

	// Build the template.
	template, err := scriggo.BuildTemplate(fsys, name, opts)
	if err != nil {
		return err
	}

	var buildTime time.Duration
	if flags.metrics {
		buildTime = time.Since(start)
	}

	// Handle "-S" option.
	if flags.s >= -1 {
		asm := template.Disassemble(flags.s)
		_, _ = os.Stderr.Write(asm)
	}

	out := os.Stdout
	if flags.o != "" {
		var fi *os.File
		fi, err = openRenderOut(flags.o, path.Base(name))
		if err != nil {
			return err
		}
		defer func() {
			err = fi.Close()
		}()
		out = fi
	}

	if flags.metrics {
		start = time.Now()
	}

	// Run the template.
	err = template.Run(out, nil, nil)

	if flags.metrics {
		runTime := time.Since(start)
		_, _ = fmt.Fprintf(os.Stderr, "\n     %12s  %12s  %12s  %s\n", "Build", "Run", "Total", "File")
		_, _ = fmt.Fprintf(os.Stderr, "     %12s  %12s  %12s  %s\n", "-----", "---", "-----", "----")
		_, _ = fmt.Fprintf(os.Stderr, "     %12s  %12s  %12s  %s\n", buildTime, runTime, buildTime+runTime, name)
	}

	return err
}

// parseFormat parses and returns a format.
func parseFormat(s string) (scriggo.Format, error) {
	switch s {
	case "Text":
		return scriggo.FormatText, nil
	case "HTML":
		return scriggo.FormatHTML, nil
	case "Markdown":
		return scriggo.FormatMarkdown, nil
	case "CSS":
		return scriggo.FormatCSS, nil
	case "JS":
		return scriggo.FormatJS, nil
	case "JSON":
		return scriggo.FormatJSON, nil
	}
	return 0, fmt.Errorf("invalid format %s, format can be Text, HTML, Markdown, CSS, JS or JSON", s)
}

// formatFS is a file system that implements scriggo.FormatFS.
type formatFS struct {
	fs.FS
	format scriggo.Format
}

func (fsys formatFS) Format(name string) (scriggo.Format, error) {
	return fsys.format, nil
}

// openRenderOut opens and returns a file named name, creating or truncating
// it. If it is a directory it opens and returns the file named base in
// the directory.
func openRenderOut(name, base string) (*os.File, error) {
	fi, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		st, err2 := os.Stat(name)
		if err2 != nil {
			return nil, err2
		}
		if !st.IsDir() {
			return nil, err
		}
		fi, err = os.OpenFile(path.Join(name, base), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			return nil, err
		}
	}
	return fi, nil
}

// parseConstants parses s as a sequence of name=value pairs where name is a
// Go identifier (excluded the blank identifier and Go predefined identifiers)
// and value is a string literal, a number literal, true or false. It adds
// each name/value pair to globals.
func parseConstants(s string, globals native.Declarations) error {
	var name string
	s = strings.TrimSpace(s)
	for s != "" {
		i := strings.Index(s, "=")
		if i == -1 {
			return errors.New("missing constant value")
		}
		name, s = s[:i], s[i+1:]
		name = strings.TrimSpace(name)
		if name == "_" || !isIdentifier(name) || isPredeclaredIdentifier(name) {
			return fmt.Errorf("constant name %s cannot be used as identifier", name)
		}
		s = strings.TrimSpace(s)
		var global native.Declaration
		if s == "false" || (strings.HasPrefix(s, "false") && isSpace(s[5])) {
			global = native.UntypedBooleanConst(false)
			s = s[5:]
		} else if s == "true" || (strings.HasPrefix(s, "true") && isSpace(s[4])) {
			global = native.UntypedBooleanConst(false)
			s = s[4:]
		} else if s[0] == '"' {
			if str, ok := parseStringConstant(s); ok {
				if c, err := strconv.Unquote(str); err == nil {
					global = native.UntypedStringConst(c)
					s = s[len(str):]
				}
			}
		} else if s[0] == '`' {
			if j := strings.IndexByte(s[1:], '`'); j != -1 {
				if c, err := strconv.Unquote(s[:j+2]); err == nil {
					global = native.UntypedStringConst(c)
					s = s[j+2:]
				}
			}
		} else {
			var n string
			if j := strings.IndexAny(s, " \t\n\r"); j > 0 {
				n, s = s[:j], s[j+1:]
			} else {
				n, s = s, ""
			}
			if _, _, err := big.ParseFloat(n, 0, 512, big.AwayFromZero); err == nil {
				global = native.UntypedNumericConst(n)
			}
		}
		if global == nil {
			return fmt.Errorf("constant %s has an invalid value", name)
		}
		globals[name] = global
	}
	return nil
}

// isSpace reports whether s is a space.
func isSpace(s byte) bool {
	return s == ' ' || s == '\t' || s == '\n' || s == '\r'
}

// parseStringConstant parses a quoted string with trailing bytes. It returns
// the quoted string without the trailing bytes and true. If the string is not
// closed, it returns and empty string and false.
func parseStringConstant(s string) (string, bool) {
	i := 1
	for i < len(s) {
		j := strings.IndexByte(s[i:], '"')
		if j == -1 {
			return "", false
		}
		var slashes int
		for k := i + j - 1; k >= i && s[k] == '\\'; k-- {
			slashes++
		}
		if slashes%2 == 0 {
			return s[:i+j+1], true
		}
		i += j + 1
	}
	return "", false
}

// isIdentifier reports whether s is an identifier.
func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r != '_' && !unicode.IsLetter(r) && (i == 0 || !unicode.IsDigit(r)) {
			return false
		}
	}
	return !isGoKeyword(s)
}
