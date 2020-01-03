// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

const header = `// Code generated by sources_gen/main.go. DO NOT EDIT.

// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

func init() {
	sources = map[string][]byte{}
`

const footer = `}
`

func main() {
	dir, err := os.Open(".")
	if err != nil {
		log.Fatalf("cannot open directory: %s", err)
	}
	defer dir.Close()
	names, err := dir.Readdirnames(0)
	if err != nil {
		log.Fatalf("cannot read directory: %s", err)
	}
	sort.Strings(names)
	var b bytes.Buffer
	b.WriteString(header)
	for _, name := range names {
		if name == "sources.go" || name == "run-packages.go" ||
			strings.HasSuffix(name, "_test.go") || !strings.HasSuffix(name, ".go") {
			continue
		}
		src, err := ioutil.ReadFile(name)
		if err != nil {
			log.Fatalf("cannot read file %s: %s", name, err)
		}
		b.WriteString("\tsources[")
		b.WriteString(strconv.Quote(name))
		b.WriteString("] = []byte(`")
		src = bytes.ReplaceAll(src, []byte("`"), []byte("` + \"`\" + `"))
		b.Write(src)
		b.WriteString("`)\n")
	}
	b.WriteString(footer)
	dest, err := os.OpenFile("sources.go", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("cannot open file sources.go: %s", err)
	}
	defer dest.Close()
	_, err = b.WriteTo(dest)
	if err != nil {
		log.Fatalf("cannot write file sources.go: %s", err)
	}
	return
}
