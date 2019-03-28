// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
)

const minIntAsFloat64 = 1 << 53

type Properties uint8

const (
	PropertyNil         Properties = 1 << (8 - 1 - iota) // is predeclared nil
	PropertyUntyped                                      // is untyped
	PropertyIsConstant                                   // is a constant
	PropertyIsType                                       // is a type
	PropertyIsPackage                                    // is a package
	PropertyIsBuiltin                                    // is a builtin
	PropertyAddressable                                  // is addressable
)

type TypeInfo struct {
	Type       reflect.Type // Type.
	Properties Properties   // Properties.
	Value      interface{}  // Value; for packages has type *Package.
}

// Nil reports whether it is the predeclared nil.
func (ti *TypeInfo) Nil() bool {
	return ti.Properties&PropertyNil != 0
}

// Untyped reports whether it is untyped.
func (ti *TypeInfo) Untyped() bool {
	return ti.Properties&PropertyUntyped != 0
}

// IsConstant reports whether it is a constant.
func (ti *TypeInfo) IsConstant() bool {
	return ti.Properties&PropertyIsConstant != 0
}

// IsUntypedConstant reports whether it is an untyped constant.
func (ti *TypeInfo) IsUntypedConstant() bool {
	return ti.Properties&PropertyUntyped != 0 && ti.Properties&PropertyIsConstant != 0
}

// IsType reports whether it is a type.
func (ti *TypeInfo) IsType() bool {
	return ti.Properties&PropertyIsType != 0
}

// IsPackage reports whether it is a package.
func (ti *TypeInfo) IsPackage() bool {
	return ti.Properties&PropertyIsPackage != 0
}

// IsBuiltin reports whether it is a builtin.
func (ti *TypeInfo) IsBuiltin() bool {
	return ti.Properties&PropertyIsBuiltin != 0
}

// Addressable reports whether it is addressable.
func (ti *TypeInfo) Addressable() bool {
	return ti.Properties&PropertyAddressable != 0
}

var runeType = reflect.TypeOf(rune(0))

// String returns a string representation.
func (ti *TypeInfo) String() string {
	if ti.Nil() {
		return "nil"
	}
	var s string
	if ti.Untyped() {
		s = "untyped "
	}
	if ti.IsConstant() && ti.Type == runeType {
		return s + "rune"
	}
	return s + ti.Type.String()
}

// ShortString returns a short string representation.
func (ti *TypeInfo) ShortString() string {
	if ti.Nil() {
		return "nil"
	}
	if ti.IsConstant() && ti.Type == runeType {
		return "rune"
	}
	return ti.Type.String()
}

// StringWithNumber returns the string representation of ti in the context of a
// function call and return statement.
func (ti *TypeInfo) StringWithNumber(explicitUntyped bool) string {
	if ti.Untyped() && ti.IsNumeric() {
		if explicitUntyped {
			return "untyped number"
		}
		return "number"
	}
	return ti.Type.String()
}

// IsNumeric reports whether it is numeric.
func (ti *TypeInfo) IsNumeric() bool {
	if ti.Nil() {
		return false
	}
	k := ti.Type.Kind()
	return reflect.Int <= k && k <= reflect.Complex128
}

// IsInteger reports whether it is an integer.
func (ti *TypeInfo) IsInteger() bool {
	if ti.Nil() {
		return false
	}
	k := ti.Type.Kind()
	return reflect.Int <= k && k <= reflect.Uintptr
}

//  CanInt64 reports whether it is safe to call Int64.
func (ti *TypeInfo) CanInt64() bool {
	switch v := ti.Value.(type) {
	case int64:
		return true
	case *big.Int:
		return v.IsInt64()
	case float64:
		if -minIntAsFloat64 <= v && v <= minIntAsFloat64 {
			return true
		}
		if _, acc := big.NewFloat(v).Int64(); acc == big.Exact {
			return true
		}
	case *big.Float:
		if _, acc := v.Int64(); acc == big.Exact {
			return true
		}
	case *big.Rat:
		return v.IsInt() && v.Num().IsInt64()
	}
	return false
}

//  CanUint64 reports whether it is safe to call Uint64.
func (ti *TypeInfo) CanUint64() bool {
	switch v := ti.Value.(type) {
	case int64:
		return v >= 0
	case *big.Int:
		return v.IsUint64()
	case float64:
		if v >= 0 {
			if v <= minIntAsFloat64 {
				return true
			}
			if _, acc := big.NewFloat(v).Uint64(); acc == big.Exact {
				return true
			}
		}
	case *big.Float:
		if _, acc := v.Uint64(); acc == big.Exact {
			return true
		}
	case *big.Rat:
		return v.IsInt() && v.Num().IsUint64()
	}
	return false
}

//  CanFloat64 reports whether it is safe to call Float64.
func (ti *TypeInfo) CanFloat64() bool {
	switch v := ti.Value.(type) {
	case int64:
		return true
	case *big.Int:
		if v.IsInt64() || v.IsUint64() {
			return true
		}
		f := (&big.Float{}).SetInt(v)
		if _, acc := f.Float64(); acc == big.Exact {
			return true
		}
	case float64:
		return true
	case *big.Float:
		if f, _ := v.Float64(); !math.IsInf(f, 1) {
			return true
		}
	case *big.Rat:
		f, _ := v.Float64()
		return !math.IsInf(f, 0)
	}
	return false
}

// Int64 returns the value as an int64. If the value can not be represented by
// an int64 the behaviour is undefined.
func (ti *TypeInfo) Int64() int64 {
	switch v := ti.Value.(type) {
	case int64:
		return v
	case *big.Int:
		return v.Int64()
	case float64:
		return int64(v)
	case *big.Float:
		n, _ := v.Int64()
		return n
	case *big.Rat:
		return v.Num().Int64()
	}
	return 0
}

// Uint64 returns the value as an uint64. If the value can not be represented
// by an uint64 the behaviour is undefined.
func (ti *TypeInfo) Uint64() uint64 {
	switch v := ti.Value.(type) {
	case int64:
		return uint64(v)
	case *big.Int:
		return v.Uint64()
	case float64:
		return uint64(v)
	case *big.Float:
		n, _ := v.Uint64()
		return n
	case *big.Rat:
		return v.Num().Uint64()
	}
	return 0
}

// Float64 returns the value as a float64. If the value can not be represented
// by a float64 the behaviour is undefined.
func (ti *TypeInfo) Float64() float64 {
	switch v := ti.Value.(type) {
	case int64:
		return float64(v)
	case *big.Int:
		return float64(v.Int64())
	case float64:
		return v
	case *big.Float:
		n, _ := v.Float64()
		return n
	case *big.Rat:
		n, _ := v.Float64()
		return n
	}
	return 0
}

// TypedValue returns the value represented with type t.
func (ti *TypeInfo) TypedValue(t reflect.Type) interface{} {
	switch t.Kind() {
	case reflect.Bool:
		return ti.Value.(bool)
	case reflect.String:
		return ti.Value.(string)
	case reflect.Int:
		return int(ti.Int64())
	case reflect.Int8:
		return int8(ti.Int64())
	case reflect.Int16:
		return int16(ti.Int64())
	case reflect.Int32:
		return int32(ti.Int64())
	case reflect.Int64:
		return ti.Int64()
	case reflect.Uint:
		return uint(ti.Uint64())
	case reflect.Uint8:
		return uint8(ti.Uint64())
	case reflect.Uint16:
		return uint16(ti.Uint64())
	case reflect.Uint32:
		return uint32(ti.Uint64())
	case reflect.Uint64:
		return ti.Uint64()
	case reflect.Float32:
		return float32(ti.Float64())
	case reflect.Float64:
		return ti.Float64()
	case reflect.Interface:
		v := ti.TypedValue(ti.Type)
		if !ti.Untyped() && ti.Type.Name() != "" { // Defined type.
			nv := reflect.New(ti.Type).Elem()
			switch ti.Type.Kind() {
			case reflect.Invalid:
			case reflect.Bool:
				nv.SetBool(v.(bool))
			case reflect.Int:
				nv.SetInt(int64(v.(int)))
			case reflect.Int8:
				nv.SetInt(int64(v.(int8)))
			case reflect.Int16:
				nv.SetInt(int64(v.(int16)))
			case reflect.Int32:
				nv.SetInt(int64(v.(int32)))
			case reflect.Int64:
				nv.SetInt(v.(int64))
			case reflect.Uint:
				nv.SetUint(uint64(v.(uint)))
			case reflect.Uint8:
				nv.SetUint(uint64(v.(uint8)))
			case reflect.Uint16:
				nv.SetUint(uint64(v.(uint16)))
			case reflect.Uint32:
				nv.SetUint(uint64(v.(uint32)))
			case reflect.Uint64:
				nv.SetUint(v.(uint64))
			case reflect.Float32:
				nv.SetFloat(float64(v.(float32)))
			case reflect.Float64:
				nv.SetFloat(v.(float64))
			case reflect.Complex64, reflect.Complex128:
				nv.SetComplex(v.(complex128))
			case reflect.String:
				nv.SetString(v.(string))
			case reflect.Interface:
				nv.Set(reflect.ValueOf(v))
			default:
				panic(fmt.Errorf("unexpected kind %q (from type %q)", t.Kind(), t))
			}
			v = nv.Interface()
		}
		return v
	}
	panic("unexpected kind")
}
