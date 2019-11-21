// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
)

type Properties uint8

const (
	PropertyUntyped      Properties = 1 << iota // is untyped
	PropertyIsType                              // is a type
	PropertyIsPackage                           // is a package
	PropertyPredeclared                         // is predeclared
	PropertyAddressable                         // is addressable
	PropertyIsPredefined                        // is predefined
	PropertyHasValue                            // has a value
)

// A TypeInfo holds the type checking information. For example, every expression
// in the AST has an associated TypeInfo. The TypeInfo is also used in the type
// checker scopes to associate the declarations to the type checking
// information.
type TypeInfo struct {
	Type              reflect.Type // Type.
	Properties        Properties   // Properties.
	Constant          constant     // Constant value.
	PredefPackageName string       // Name of the package. Empty string if not predefined.
	MethodType        MethodType   // Method type.
	value             interface{}  // value; for packages has type *Package.
	valueType         reflect.Type // When value is a predeclared type holds the original type of value.
}

// MethodType represents the type of a method, intended as a combination of a
// method call/value/expression and a receiver type (concrete or interface).
type MethodType uint8

const (
	NoMethod             MethodType = iota // Not a method.
	MethodValueConcrete                    // Method value on a concrete receiver.
	MethodValueInterface                   // Method value on an interface receiver.
	MethodCallConcrete                     // Method call on concrete receiver.
	MethodCallInterface                    // Method call on interface receiver.
)

// Nil reports whether it is the predeclared nil.
func (ti *TypeInfo) Nil() bool {
	return ti.Predeclared() && ti.Untyped() && ti.Type == nil
}

// Untyped reports whether it is untyped.
func (ti *TypeInfo) Untyped() bool {
	return ti.Properties&PropertyUntyped != 0
}

// IsConstant reports whether it is a constant.
func (ti *TypeInfo) IsConstant() bool {
	return ti.Constant != nil
}

// IsUntypedConstant reports whether it is an untyped constant.
func (ti *TypeInfo) IsUntypedConstant() bool {
	return ti.Properties&PropertyUntyped != 0 && ti.Constant != nil
}

// IsType reports whether it is a type.
func (ti *TypeInfo) IsType() bool {
	return ti.Properties&PropertyIsType != 0
}

// IsPackage reports whether it is a package.
func (ti *TypeInfo) IsPackage() bool {
	return ti.Properties&PropertyIsPackage != 0
}

// Predeclared reports whether it is predeclared.
func (ti *TypeInfo) Predeclared() bool {
	return ti.Properties&PropertyPredeclared != 0
}

// Addressable reports whether it is addressable.
func (ti *TypeInfo) Addressable() bool {
	return ti.Properties&PropertyAddressable != 0
}

// IsPredefined reports whether it is predefined.
func (ti *TypeInfo) IsPredefined() bool {
	return ti.Properties&PropertyIsPredefined != 0
}

// IsBuiltinFunction reports whether it is a builtin function.
func (ti *TypeInfo) IsBuiltinFunction() bool {
	return ti.Properties&PropertyPredeclared != 0 && ti.Properties&PropertyUntyped == 0 && ti.Type == nil
}

// TODO: review.
func (ti *TypeInfo) UntypedNonConstantInteger() bool {
	return ti.IsInteger() && !ti.IsConstant() && ti.Untyped()
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
	// X MARCO: is this change ok or is it obsolete?
	if ti.Type == nil {
		s += "unsigned number"
	} else {
		s += ti.Type.String()
	}
	return s
}

// ShortString returns a short string representation.
func (ti *TypeInfo) ShortString() string {
	if ti.Nil() {
		return "nil"
	}
	if ti.IsConstant() && ti.Type == runeType {
		return "rune"
	}
	// X MARCO: is this change ok or is it obsolete?
	if ti.Type == nil {
		return "unsigned number"
	}
	return ti.Type.String()
}

// StringWithNumber returns the string representation of ti in the context of
// a function call, return statement and type switch assertion.
func (ti *TypeInfo) StringWithNumber(explicitUntyped bool) string {
	if ti.Nil() {
		return "nil"
	}
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

// HasValue reports whether it has a value.
func (ti *TypeInfo) HasValue() bool {
	return ti.Properties&PropertyHasValue != 0
}

// setValue sets the 'value' and 'valueType' fields of 'ti' if this is constant.
// If ti is not constant, setValue is a no-op.
//
// This method should be called at every point where a constant expression is
// used in a non-constant expression or in a statement.
//
// The type of 'value' is determined in the following way:
//
//      - if a ctxType is given, the value takes type from the context. This is the
//      case, for example, of integer constants assigned to float numbers. As a special case,
//      if context type is interface the type of ti is used.
//
//      - if ctxType is nil, the value is implicitly taken from ti. This is the case
//      of a context that does not provide an explicit type, as a variable
//      declaration without type.
//
// The following examples should clarify the use of this method:
//
// 		var i int64 = 20     call setValue on '20'    ctxType = int
//      x + 3                call setValue on '3'     ctxType = typeof(x)
//      x + y                no need to call setValue
//
func (ti *TypeInfo) setValue(ctxType reflect.Type) {
	typ := ctxType
	if ctxType == nil || ctxType.Kind() == reflect.Interface {
		typ = ti.Type
	}
	if ti.IsConstant() {
		switch typ.Kind() {
		case reflect.Bool:
			if ti.Constant.bool() {
				ti.value = int64(1)
			} else {
				ti.value = int64(0)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			ti.value = ti.Constant.int64()
		case reflect.Uint:
			ti.value = int64(int(ti.Constant.uint64()))
		case reflect.Uint8:
			ti.value = int64(int8(ti.Constant.uint64()))
		case reflect.Uint16:
			ti.value = int64(int16(ti.Constant.uint64()))
		case reflect.Uint32:
			ti.value = int64(int32(ti.Constant.uint64()))
		case reflect.Uint64:
			ti.value = int64(ti.Constant.uint64())
		case reflect.Uintptr:
			ti.value = int64(ti.Constant.uint64())
		case reflect.Float32, reflect.Float64:
			ti.value = ti.Constant.float64()
		case reflect.Complex64, reflect.Complex128:
			switch c := ti.Constant.complex128(); typ {
			case complex64Type:
				ti.value = complex64(c)
			case complex128Type:
				ti.value = c
			default:
				rv := reflect.New(typ).Elem()
				rv.SetComplex(c)
				ti.value = rv.Interface()
			}
		case reflect.String:
			ti.value = ti.Constant.string()
		}
		ti.valueType = typ
		ti.Properties |= PropertyHasValue
		return
	}
	if ti.Nil() {
		panic("BUG: cannot call method setValue on a type info representing the predeclared nil") // remove.
	}
}
