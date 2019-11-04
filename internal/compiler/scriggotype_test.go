// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"testing"
)

var sliceIntType = reflect.TypeOf([]int{})

func TestScriggoTypeMethods(t *testing.T) {
	cases := []struct {
		input        scriggoType
		expectedName string
	}{
		{
			input: scriggoType{
				definedName: "Int",
				Type:        intType,
			},
			expectedName: "Int",
		},
		{
			input: scriggoType{
				definedName: "SliceInt",
				elem: &scriggoType{
					definedName: "Int",
					Type:        intType,
				},
				Type: sliceIntType,
			},
			expectedName: "SliceInt",
		},
	}
	for _, cas := range cases {
		t.Run(cas.expectedName, func(t *testing.T) {
			gotName := cas.input.Name()
			if gotName != cas.expectedName {
				t.Fatalf("expecting name '%s', got '%s'", cas.expectedName, gotName)
			}
		})
	}
}
