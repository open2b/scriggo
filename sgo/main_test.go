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

// Test_sgo_gen_fmt tests command 'sgo gen fmt'
func Test_sgo_gen_fmt(t *testing.T) {
	TestEnvironment = true
	tmpDir, err := ioutil.TempDir("", "sgo_test")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	sgo("sgo", "gen", "fmt")

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
