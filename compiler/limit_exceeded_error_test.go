// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/open2b/scriggo/compiler/ast"
	"github.com/open2b/scriggo/runtime"
)

func new_test_builder() *functionBuilder {
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

			var i int

			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("test should have failed")
				} else {
					if _, ok := r.(*LimitExceededError); ok {
						// The type of the error is correct. Now check if the
						// test panicked at the correct index.
						if maxRegistersCount != i {
							t.Fatalf("test should have panicked at index %d, but it panicked at index %d", maxRegistersCount, i)
						}
					} else {
						t.Fatalf("expecting a LimitExceededError, got error %s (of type %T)", r, r)
					}
				}
			}()

			fb := new_test_builder()
			for i = 0; i < 1000; i++ {
				fb.newRegister(kind)
			}

		})
	}
}

func TestFunctionsLimit(t *testing.T) {

	var i int

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("test should have failed")
		} else {
			if _, ok := r.(*LimitExceededError); ok {
				// The type of the error is correct. Now check if the
				// test panicked at the correct index.
				if maxScriggoFunctionsCount != i {
					t.Fatalf("test should have panicked at index %d, but it panicked at index %d", maxScriggoFunctionsCount, i)
				}
			} else {
				t.Fatalf("expecting a LimitExceededError, got error %s (of type %T)", r, r)
			}
		}
	}()

	fb := new_test_builder()
	for i = 0; i < 1000; i++ {
		fn := &runtime.Function{
			Pkg:    fb.fn.Pkg,
			File:   fb.fn.File,
			Type:   reflect.FuncOf(nil, nil, false),
			Parent: fb.fn,
		}
		fb.emitLoadFunc(false, fb.addFunction(fn), 1)
	}

}

func TestTypesLimit(t *testing.T) {

	var i int

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("test should have failed")
		} else {
			if _, ok := r.(*LimitExceededError); ok {
				// The type of the error is correct. Now check if the
				// test panicked at the correct index.
				if maxTypesCount != i {
					t.Fatalf("test should have panicked at index %d, but it panicked at index %d", maxTypesCount, i)
				}
			} else {
				t.Fatalf("expecting a LimitExceededError, got error %s (of type %T)", r, r)
			}
		}
	}()

	fb := new_test_builder()
	for i = 0; i < 1000; i++ {
		typ := reflect.ArrayOf(i, intType)
		fb.emitNew(typ, 0)
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

			var i int

			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("test should have failed")
				} else {
					if _, ok := r.(*LimitExceededError); ok {
						// The type of the error is correct. Now check if the
						// test panicked at the correct index.
						var expectedIndex int
						switch kind {
						case reflect.Int:
							expectedIndex = maxIntValuesCount
						case reflect.Float64:
							expectedIndex = maxFloatValuesCount
						case reflect.String:
							expectedIndex = maxStringValuesCount
						case reflect.Interface:
							expectedIndex = maxGeneralValuesCount
						}
						if expectedIndex != i {
							t.Fatalf("test should have panicked at index %d, but it panicked at index %d", expectedIndex, i)
						}
					} else {
						t.Fatalf("expecting a LimitExceededError, got error %s (of type %T)", r, r)
					}
				}
			}()

			fb := new_test_builder()
			switch kind {
			case reflect.Int:
				for i = 0; i < maxIntValuesCount+1; i++ {
					fb.makeIntValue(int64(i))
				}
			case reflect.Float64:
				for i = 0; i < maxFloatValuesCount+1; i++ {
					fb.makeFloatValue(float64(i))
				}
			case reflect.String:
				for i = 0; i < maxStringValuesCount+1; i++ {
					fb.makeStringValue(strconv.Itoa(i))
				}
			case reflect.Interface:
				for i = 0; i < maxGeneralValuesCount+1; i++ {
					fb.makeGeneralValue(reflect.ValueOf(i))
				}
			}

		})
	}
}
