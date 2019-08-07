package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"scriggo"
)

// packages contains the predefined packages used in tests.
//go:generate scriggo embed -v -o packages.go
var packages scriggo.Packages

type mainLoader []byte

func (b mainLoader) Load(path string) (interface{}, error) {
	if path == "main" {
		return bytes.NewReader(b), nil
	}
	return nil, nil
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
	default:
		panic("invalid argument: %s" + os.Args[1])
	}
}
