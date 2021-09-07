// Copyright 2021 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"
)

func BenchmarkRun(b *testing.B) {
	programs, err := build()
	if err != nil {
		b.Fatal(err)
	}
	for _, program := range programs {
		b.Run(program.name, func(b *testing.B) {
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				err := program.code.Run(nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
