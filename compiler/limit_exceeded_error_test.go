// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"scriggo/compiler/ast"
	"strconv"
	"testing"
)

func test_builder() *functionBuilder {
	fn := newFunction("", "", reflect.FuncOf(nil, nil, false), "", &ast.Position{})
	return newBuilder(fn, "")
}

func TestRegistersLimit(t *testing.T) {
	cases := []reflect.Kind{
		reflect.Int,
		reflect.String,
		reflect.Float64,
		reflect.Interface,
	}
	for _, kind := range cases {
		t.Run(kind.String(), func(t *testing.T) {

			// This should be ok.
			fb := test_builder()
			for i := 0; i < 126; i++ {
				fb.newRegister(kind)
			}

			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("test should have failed")
				} else {
					if _, ok := r.(*LimitExceededError); ok {
						// Test passed.
					} else {
						t.Fatalf("expecting a LimitExceededError, got error %s (of type %T)", r, r)
					}
				}
			}()

			// This should fail.
			fb = test_builder()
			for i := 0; i < 127; i++ {
				fb.newRegister(kind)
			}

		})
	}
}

func TestConstantsLimit(t *testing.T) {
	cases := []reflect.Kind{
		reflect.Int,
		reflect.String,
		reflect.Float64,
		reflect.Interface,
	}
	for _, kind := range cases {
		t.Run(kind.String(), func(t *testing.T) {

			// This should be ok.
			fb := test_builder()
			for i := 0; i < 255; i++ {
				switch kind {
				case reflect.Int:
					fb.makeIntConstant(int64(i))
				case reflect.String:
					fb.makeStringConstant(strconv.Itoa(i))
				case reflect.Float64:
					fb.makeFloatConstant(float64(i))
				case reflect.Interface:
					fb.makeGeneralConstant(interface{}(i))
				}
			}

			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("test should have failed")
				} else {
					if _, ok := r.(*LimitExceededError); ok {
						// Test passed.
					} else {
						t.Fatalf("expecting a LimitExceededError, got error %s (of type %T)", r, r)
					}
				}
			}()

			// This should fail.
			fb = test_builder()
			for i := 0; i < 256; i++ {
				switch kind {
				case reflect.Int:
					fb.makeIntConstant(int64(i))
				case reflect.String:
					fb.makeStringConstant(strconv.Itoa(i))
				case reflect.Float64:
					fb.makeFloatConstant(float64(i))
				case reflect.Interface:
					fb.makeGeneralConstant(interface{}(i))
				}
			}

		})
	}
}
