// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// Test_sgc_generate_fmt tests command 'sgc generate fmt'
func Test_sgc_generate_fmt(t *testing.T) {
	TestEnvironment = true
	tmpDir, err := ioutil.TempDir("", "sgc_test")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	os.Args = []string{"sgc", "generate", "fmt"}
	main()

	for _, fileName := range []string{"main.go", "go.mod"} {
		data, err := ioutil.ReadFile(filepath.Join(tmpDir, "fmt", fileName))
		if err != nil {
			t.Fatal(err)
		}
		if len(data) == 0 {
			t.Fatal(fileName + "is empty")
		}
	}

}
