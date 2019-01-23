// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"open2b/template"
)

func main() {

	if len(os.Args) != 2 {
		fmt.Printf("usage: %s filename\n", os.Args[0])
		os.Exit(-1)
	}

	file := os.Args[1]
	fi, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("file %q does not exist\n", file)
		} else {
			fmt.Printf("opening file %q: %s\n", file, err)
		}
		os.Exit(-1)
	}
	src, err := ioutil.ReadAll(fi)
	_ = fi.Close()
	if err != nil {
		fmt.Printf("reading file %q: %s\n", file, err)
		os.Exit(-1)
	}
	err = template.RenderSource(os.Stdout, src, nil, true, template.ContextNone)
	if err != nil {
		fmt.Printf("error: %q\n", err)
	}

}
