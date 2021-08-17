// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package types implements functions and types to represent and work with
// Scriggo types.
package types

import (
	"fmt"
	"reflect"

	"github.com/open2b/scriggo/internal/runtime"
)

// Types allows to create and manage types and values, as the reflect package
// does, for a specific compilation and for both the types compiled by Scriggo
// and by gc.
type Types struct {
	// funcParamsStore provides the function parameters necessary to create
	// funcTypes.
	funcParamsStore funcParams

	// structFieldsLists avoid the creation of two different structTypes with
	// the same struct fields.
	structFieldsLists []*map[int]reflect.StructField
}

// NewTypes returns a new instance of Types.
func NewTypes() *Types {
	return &Types{}
}

// New behaves like reflect.New except when t is a Scriggo type; in such case
// it returns an instance of the Go type created with a reflect.New call.
func (types *Types) New(t reflect.Type) reflect.Value {
	if st, ok := t.(runtime.ScriggoType); ok {
		t = st.GoType()
	}
	return reflect.New(t)
}

// Zero is equivalent to reflect.Zero. If t is a Scriggo type it returns the
// zero of the underlying type.
func (types *Types) Zero(t reflect.Type) reflect.Value {
	if st, ok := t.(runtime.ScriggoType); ok {
		t = st.GoType()
	}
	return reflect.Zero(t)
}

// AssignableTo reports whether a value of type x is assignable to type y.
func AssignableTo(x, y reflect.Type) bool {

	// x and y types are identical.
	if x == y {
		return true
	}

	// If y is an interface type, x must implement y.
	k := y.Kind()
	if k == reflect.Interface {
		return Implements(x, y)
	}

	// x and y have different kind.
	if x.Kind() != k {
		return false
	}

	// x and y are both defined types.
	if x.Name() != "" && y.Name() != "" {
		return false
	}

	// x and y are channel types, x is bidirectional, and they have identical element types.
	if k == reflect.Chan && x.ChanDir() == reflect.BothDir && x.Elem() == y.Elem() {
		return true
	}

	// x and y must have identical underlying types.
	return identical(x, y, true, false)
}

// ConvertibleTo reports whether a value of type x is convertible to type y.
func ConvertibleTo(x, y reflect.Type) bool {

	// x and y types are identical.
	if x == y {
		return true
	}

	xk := x.Kind()
	yk := y.Kind()

	// y is a string type and x is an integer type or a slice of bytes or runes.
	if yk == reflect.String {
		if reflect.Int <= xk && xk <= reflect.Uintptr {
			return true
		}
		if xk == reflect.Slice {
			if e := x.Elem(); e.PkgPath() == "" {
				if k := e.Kind(); k == reflect.Uint8 || k == reflect.Int32 {
					return true
				}
			}
		}
	}

	// x and y are integer or floating point types.
	if reflect.Int <= xk && xk <= reflect.Float64 && reflect.Int <= yk && yk <= reflect.Float64 {
		return true
	}

	// x and y are complex types.
	if (xk == reflect.Complex128 || xk == reflect.Complex64) && (yk == reflect.Complex128 || yk == reflect.Complex64) {
		return true
	}

	// x is a string type and y is a slice of bytes or runes.
	if xk == reflect.String && yk == reflect.Slice {
		if e := y.Elem(); e.PkgPath() == "" {
			if k := e.Kind(); k == reflect.Uint8 || k == reflect.Int32 {
				return true
			}
		}
	}

	// x and y are channel types, are not both defined, x is bidirectional, and they have identical element types.
	if xk == reflect.Chan && yk == reflect.Chan && x.ChanDir() == reflect.BothDir &&
		x.Elem() == y.Elem() && (x.Name() == "" || y.Name() == "") {
		return true
	}

	// x is a slice, y is a pointer to array type, and the slice and array have the same element types.
	if xk == reflect.Slice && yk == reflect.Ptr {
		if e := y.Elem(); e.Kind() == reflect.Array && e.Elem() == x.Elem() {
			return true
		}
	}

	// If y is an interface type, x must implement y.
	if yk == reflect.Interface {
		return Implements(x, y)
	}

	// x and y have identical underlying types, ignoring struct tags.
	if identical(x, y, true, true) {
		return true
	}

	// x and y are not defined pointer types, and have identical underlying types, ignoring struct tags.
	if xk == reflect.Ptr && yk == reflect.Ptr &&
		x.Name() == "" && y.Name() == "" && identical(x.Elem(), y.Elem(), true, true) {
		return true
	}

	return false
}

// Implements reports whether x implements the interface type y.
func Implements(x, y reflect.Type) bool {
	if _, ok := x.(runtime.ScriggoType); ok {
		return y.NumMethod() == 0
	}
	if _, ok := y.(runtime.ScriggoType); ok {
		return true
	}
	// If y has unexported methods, and x is not an interface type, it is not possible to check
	// if x implements y using the x.NumMethod and x.Method methods, because they do not return
	// the unexported methods of x. Therefore, the x.Implements method is used instead.
	return x.Implements(y)
}

// identical reports whether the types x and y, or their underlying types if
// underlying is true, are identical. If ignoreTags is true, the tags are
// ignored.
func identical(x, y reflect.Type, underlying, ignoreTags bool) bool {

	// x and y are identical.
	if x == y {
		return true
	}

	// x and y are not identical, ignoring tags.
	if !underlying && !ignoreTags {
		return false
	}

	// x and y have different kind.
	k := x.Kind()
	if k != y.Kind() {
		return false
	}

	// x and y are not both non-defined types.
	if !underlying && !(x.Name() == "" && y.Name() == "") {
		return false
	}

	// x and y have structurally equivalent underlying type literals.
	switch k {
	case reflect.Array:
		return x.Len() == y.Len() && identical(x.Elem(), y.Elem(), false, ignoreTags)
	case reflect.Slice, reflect.Ptr:
		return identical(x.Elem(), y.Elem(), false, ignoreTags)
	case reflect.Struct:
		n := x.NumField()
		if n != y.NumField() {
			return false
		}
		for i := 0; i < n; i++ {
			f1 := x.Field(i)
			f2 := y.Field(i)
			// Same name.
			if f1.Name != f2.Name {
				return false
			}
			// Same package for non-exported field names.
			if f1.PkgPath != f2.PkgPath {
				return false
			}
			// Identical element types.
			if !identical(f1.Type, f2.Type, false, ignoreTags) {
				return false
			}
			// Same tag, if not ignored.
			if !ignoreTags && f1.Tag != f2.Tag {
				return false
			}
			// Both embedded or both non-embedded.
			if f1.Anonymous != f2.Anonymous {
				return false
			}
			// Same offset.
			if f1.Offset != f2.Offset {
				return false
			}
		}
	case reflect.Func:
		in, out := x.NumIn(), x.NumOut()
		if in != y.NumIn() || out != y.NumOut() || x.IsVariadic() != y.IsVariadic() {
			return false
		}
		for i := 0; i < in; i++ {
			if !identical(x.In(i), y.In(i), false, ignoreTags) {
				return false
			}
		}
		for i := 0; i < out; i++ {
			if !identical(x.Out(i), y.Out(i), false, ignoreTags) {
				return false
			}
		}
	case reflect.Interface:
		xn := x.NumMethod()
		yn := y.NumMethod()
		if xn != yn {
			return false
		}
		for i := 0; i < xn; i++ {
			xm := x.Method(i)
			ym := y.Method(i)
			// Same name.
			if xm.Name != ym.Name {
				return false
			}
			// Same package for non-exported method names.
			if xm.PkgPath != ym.PkgPath {
				return false
			}
			// Identical function types.
			if !identical(xm.Type, ym.Type, false, ignoreTags) {
				return false
			}
		}
	case reflect.Map:
		return identical(x.Key(), y.Key(), false, ignoreTags) && identical(x.Elem(), y.Elem(), false, ignoreTags)
	case reflect.Chan:
		return x.ChanDir() == y.ChanDir() && identical(x.Elem(), y.Elem(), false, ignoreTags)
	}

	return true
}

// TypeOf returns the type of v.
func (types *Types) TypeOf(v reflect.Value) reflect.Type {
	if !v.IsValid() {
		return nil
	}
	if p, ok := v.Interface().(emptyInterfaceProxy); ok {
		return p.sign
	}
	return v.Type()
}

// TODO: this function will be removed when the development of this package is
//  concluded.
func assertNotScriggoType(t reflect.Type) {
	if _, ok := t.(runtime.ScriggoType); ok {
		panic(fmt.Errorf("%v is a Scriggo type!", t))
	}
}

// TODO: every call to a reflect function in the compiler should be checked and
//  eventually converted to a call to a function of this package.
