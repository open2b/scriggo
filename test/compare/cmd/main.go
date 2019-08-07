// This executable is not meant to be human friendly or to have a nice CLI
// interface. It just aims to simplicity, speed and robustness.

package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"scriggo"
)

//go:generate scriggo embed -v -o packages.go
var packages scriggo.Packages

// TODO(Gianluca): convert mainLoader to a reader, so os.Stdin can be called
// directly.
type mainLoader []byte

func (b mainLoader) Load(path string) (interface{}, error) {
	if path == "main" {
		return bytes.NewReader(b), nil
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
		src, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}
		_, err = scriggo.LoadProgram(scriggo.Loaders(mainLoader(src), packages), nil)
		if err != nil {
			panic(err)
		}
	case "compile script":
		src, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}
		_, err = scriggo.LoadScript(bytes.NewReader(src), packages, nil)
		if err != nil {
			panic(err)
		}
	case "run program":
		src, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}
		program, err := scriggo.LoadProgram(scriggo.Loaders(mainLoader(src), packages), nil)
		if err != nil {
			panic(err)
		}
		err = program.Run(nil)
		if err != nil {
			panic(err)
		}
	case "run script":
		script, err := scriggo.LoadScript(os.Stdin, packages, nil)
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
		prog, err := scriggo.LoadProgram(scriggo.CombinedLoaders{dl, packages}, nil)
		if err != nil {
			panic(err)
		}
		err = prog.Run(nil)
		if err != nil {
			panic(err)
		}
	default:
		panic("invalid argument: %s" + os.Args[1])
	}
}
