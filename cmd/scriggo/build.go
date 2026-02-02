// Copyright 2025 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/url"
	"os"
	pathPkg "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/ast/astutil"
	"github.com/open2b/scriggo/native"

	"github.com/yuin/goldmark"
)

// build the template.
func build(dir, o string, llms string, consts []string) error {

	var linkReplacer linkDestinationReplacer
	if llms != "" {
		base, err := url.Parse(llms)
		if err != nil {
			return fmt.Errorf("invalid value for -llms flag: %s", err)
		}
		linkReplacer = linkDestinationReplacer{base: base}
	}

	start := time.Now()

	srcDir := dir
	if srcDir == "" {
		srcDir = "."
	}

	publicDir := "public"
	if o != "" {
		publicDir = o
	}
	publicDir, err := filepath.Abs(publicDir)
	if err != nil {
		return err
	}
	err = checkOutDirectory(publicDir)
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

	md := goldmark.New(goldmarkOptions...)

	buildOptions := &scriggo.BuildOptions{
		Globals: make(native.Declarations, len(globals)+len(consts)+1),
		MarkdownConverter: func(src []byte, out io.Writer) error {
			return md.Convert(src, out)
		},
	}
	for n, v := range globals {
		buildOptions.Globals[n] = v
	}
	// Handle "-const" option.
	for _, c := range consts {
		err = parseConstants(c, buildOptions.Globals)
		if err != nil {
			return err
		}
	}

	srcFS := os.DirFS(srcDir)
	outputSources := map[string]string{}

	dstBase := filepath.Base(dstDir)
	publicBase := filepath.Base(publicDir)

	var bufIn, bufOut bytes.Buffer

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
						return fs.SkipDir
					}
				}
			}
			// If it is the destination temporary directory, skip it.
			if path == dstBase {
				return fs.SkipDir
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
			var buildLLMsVersion bool // true if the -llms flag is set and a Markdown template extends an HTML template
			buildOptions.UnexpandedTransformer = nil
			if llms != "" && ext == ".md" {
				buildOptions.UnexpandedTransformer = func(tree *ast.Tree) error {
					for _, node := range tree.Nodes {
						switch n := node.(type) {
						case *ast.Comment:
						case *ast.Extends:
							buildLLMsVersion = strings.HasSuffix(n.Path, ".html")
							return nil
						case *ast.Text:
							if !containsOnlySpaces(n.Text) {
								return nil
							}
						case *ast.Statements:
							if n, ok := n.Nodes[0].(*ast.Extends); ok {
								buildLLMsVersion = strings.HasSuffix(n.Path, ".html")
							}
							return nil
						}
					}
					return nil
				}
			}
			template, err := scriggo.BuildTemplate(srcFS, path, buildOptions)
			if err != nil {
				return err
			}
			dstPath := filepath.Join(dstDir, fpath) + ".html"
			fi, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return err
			}
			err = template.Run(fi, nil, nil)
			errCode := fi.Close()
			if err != nil {
				return err
			}
			if errCode != nil {
				return errCode
			}
			// Build the LLMs version if needed.
			if buildLLMsVersion {
				bufIn.Reset()
				bufOut.Reset()
				// Use a transformer to expand, import, and render Markdown files instead of HTML.
				buildOptions.UnexpandedTransformer = func(tree *ast.Tree) error {
					astutil.Inspect(tree, func(node ast.Node) bool {
						switch n := node.(type) {
						case *ast.Extends:
							n.Path = htmlToMarkdownPath(n.Path)
						case *ast.Import:
							n.Path = htmlToMarkdownPath(n.Path)
						case *ast.Render:
							n.Path = htmlToMarkdownPath(n.Path)
						}
						return true
					})
					return nil
				}
				template, err := scriggo.BuildTemplate(srcFS, path, buildOptions)
				if err != nil {
					return err
				}
				err = template.Run(&bufIn, nil, nil)
				if err != nil {
					return err
				}
				linkReplacer.dir = pathPkg.Dir(path)
				err = linkReplacer.replace(&bufOut, bufIn.Bytes())
				if err != nil {
					return err
				}
				dstPath := filepath.Join(dstDir, fpath) + ".md"
				fi, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE, 0600)
				if err != nil {
					return err
				}
				_, err = bufOut.WriteTo(fi)
				errClose := fi.Close()
				if err != nil {
					return err
				}
				if errClose != nil {
					return err
				}
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

	err = checkOutDirectory(publicDir)
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

// Check that the output directory does not already exist.
func checkOutDirectory(path string) error {
	st, err := os.Stat(path)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat %q: %s", path, err)
	}
	if st.IsDir() {
		return fmt.Errorf("output directory %q already exists", path)
	}
	return fmt.Errorf("path %q exists and is not a directory", path)
}

// htmlToMarkdownPath replaces the ".html" extension in a path with ".md".
// It does nothing if the path does not end with ".html".
func htmlToMarkdownPath(path string) string {
	if p := strings.TrimSuffix(path, ".html"); len(p) < len(path) {
		path = p + ".md"
	}
	return path
}

// containsOnlySpaces reports whether b contains only white space characters
// as intended by Go parser.
func containsOnlySpaces(bytes []byte) bool {
	for _, b := range bytes {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			return false
		}
	}
	return true
}
