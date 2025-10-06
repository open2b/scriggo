// Copyright 2024 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/native"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// build the template.
func build(dir, o string) error {

	start := time.Now()

	srcDir := dir
	if srcDir == "" {
		srcDir = "."
	}

	publicDir := "public"
	if o != "" {
		st, err := os.Stat(o)
		if !errors.Is(err, fs.ErrNotExist) {
			if err != nil {
				return fmt.Errorf("cannot stat output directory %q: %s", o, err)
			}
			if !st.IsDir() {
				return fmt.Errorf("path %q exists but is not a directory", o)
			}
		}
		publicDir = o
	}
	publicDir, err := filepath.Abs(publicDir)
	if err != nil {
		return err
	}

	dstDir, err := os.MkdirTemp(filepath.Dir(publicDir), "public-temp-*")
	if err != nil {
		return err
	}
	defer func() {
		err = os.RemoveAll(dstDir)
		if err != nil {
			log.Print(err)
		}
	}()

	md := goldmark.New(
		goldmark.WithRendererOptions(html.WithUnsafe()),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithExtensions(extension.Footnote))

	buildOptions := &scriggo.BuildOptions{
		Globals: make(native.Declarations, len(globals)+1),
		MarkdownConverter: func(src []byte, out io.Writer) error {
			return md.Convert(src, out)
		},
	}
	for n, v := range globals {
		buildOptions.Globals[n] = v
	}

	srcFS := os.DirFS(srcDir)
	outputSources := map[string]string{}

	dstBase := filepath.Base(dstDir)
	publicBase := filepath.Base(publicDir)

	err = fs.WalkDir(srcFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path[0] == '.' {
			return nil
		}
		if d.IsDir() {
			// If it is a directory with the same base name as the public directory name,
			// skip it if it is effectively the public directory. It is considered the public
			// directory if the corresponding destination temporary directory also exists.
			if path == publicBase {
				st, err := srcFS.(fs.StatFS).Stat(dstBase)
				if !errors.Is(err, fs.ErrNotExist) {
					if err != nil {
						return err
					}
					if st.IsDir() {
						return fs.SkipAll
					}
				}
			}
			// If it is the destination temporary directory, skip it.
			if path == dstBase {
				return fs.SkipAll
			}
			// Skip directories starting with an underscore.
			if strings.HasPrefix(filepath.Base(path), "_") {
				return fs.SkipDir
			}
			return os.MkdirAll(filepath.Join(dstDir, path), 0700)
		}
		ext := filepath.Ext(path)
		switch ext {
		case ".md", ".html":
			fpath := strings.TrimSuffix(path, ext)
			// Reject multiple templates that would render to the same HTML file.
			if prev, ok := outputSources[fpath]; ok {
				return fmt.Errorf("scriggo: templates %q and %q both render to %q", prev, path, fpath+".html")
			}
			outputSources[fpath] = path
			buildOptions.Globals["filepath"] = fpath
			template, err := scriggo.BuildTemplate(srcFS, path, buildOptions)
			if err != nil {
				return err
			}
			path := filepath.Join(dstDir, fpath) + ".html"
			fi, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return err
			}
			err = template.Run(fi, nil, nil)
			if errCode := fi.Close(); errCode != nil && err == nil {
				err = errCode
			}
		default:
			src, err := srcFS.Open(path)
			if err != nil {
				return err
			}
			defer src.Close()
			path := filepath.Join(dstDir, path)
			err = os.MkdirAll(filepath.Dir(path), 0700)
			if err != nil {
				return err
			}
			dst, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return err
			}
			_, err = io.Copy(dst, src)
			if errClose := dst.Close(); errClose != nil && err == nil {
				err = errClose
			}
		}
		return err
	})
	if err != nil {
		return err
	}

	err = os.RemoveAll(publicDir)
	if err != nil {
		return err
	}
	err = os.Rename(dstDir, publicDir)
	if err != nil {
		return err
	}

	buildTime := time.Since(start)
	_, _ = fmt.Fprintf(os.Stderr, "Build took %s\n", buildTime)

	return nil
}
