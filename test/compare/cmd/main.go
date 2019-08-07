// This executable is not meant to be human friendly or to have a nice CLI
// interface. It just aims to simplicity, speed and robustness.

// In case of success the standard output contains the output of the execution
// of the program/script or the rendering of the template. Else, in case of
// error, the stderr contains the error.

package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"scriggo"
	"scriggo/template"
)

//go:generate scriggo embed -v -o predefPkgs.go
var predefPkgs scriggo.Packages

type stdinLoader struct {
	file *os.File
}

func (b stdinLoader) Load(path string) (interface{}, error) {
	if path == "main" {
		return b.file, nil
	}
	return nil, nil
}

type dirLoader string

func (dl dirLoader) Load(path string) (interface{}, error) {
	if path == "main" {
		main, err := ioutil.ReadFile(filepath.Join(string(dl), "main.go"))
		if err != nil {
			panic(err)
		}
		return bytes.NewReader(main), nil
	}
	data, err := ioutil.ReadFile(filepath.Join(string(dl), path+".go"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func main() {
	switch os.Args[1] {
	case "compile program":
		_, err := scriggo.LoadProgram(scriggo.Loaders(stdinLoader{os.Stdin}, predefPkgs), nil)
		if err != nil {
			panic(err)
		}
	case "compile script":
		src, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}
		_, err = scriggo.LoadScript(bytes.NewReader(src), predefPkgs, nil)
		if err != nil {
			panic(err)
		}
	case "run program":
		program, err := scriggo.LoadProgram(scriggo.Loaders(stdinLoader{os.Stdin}, predefPkgs), nil)
		if err != nil {
			panic(err)
		}
		err = program.Run(nil)
		if err != nil {
			panic(err)
		}
	case "run script":
		script, err := scriggo.LoadScript(os.Stdin, predefPkgs, nil)
		if err != nil {
			panic(err)
		}
		err = script.Run(nil, nil)
		if err != nil {
			panic(err)
		}
	case "run program directory":
		dirPath := os.Args[2]
		dl := dirLoader(dirPath)
		prog, err := scriggo.LoadProgram(scriggo.CombinedLoaders{dl, predefPkgs}, nil)
		if err != nil {
			panic(err)
		}
		err = prog.Run(nil)
		if err != nil {
			panic(err)
		}
	case "render html":
		src, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}
		r := template.MapReader{"/index.html": src}
		templ, err := template.Load("/index.html", r, nil, template.ContextHTML, nil)
		if err != nil {
			panic(err)
		}
		err = templ.Render(os.Stdout, nil, nil)
		if err != nil {
			panic(err)
		}
	case "render html directory":
		dirPath := os.Args[2]
		r := template.DirReader(dirPath)
		templ, err := template.Load("/index.html", r, nil, template.ContextHTML, nil)
		if err != nil {
			panic(err)
		}
		err = templ.Render(os.Stdout, nil, nil)
		if err != nil {
			panic(err)
		}
	case "compile html":
		src, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}
		r := template.MapReader{"/index.html": src}
		_, err = template.Load("/index.html", r, nil, template.ContextHTML, nil)
		if err != nil {
			panic(err)
		}
	default:
		panic("invalid argument: %s" + os.Args[1])
	}
}
