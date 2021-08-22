// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
)

type properties uint16

const (
	propertyUntyped                        properties = 1 << iota // is untyped
	propertyIsType                                                // is a type
	propertyIsFormatType                                          // is a format type
	propertyIsPackage                                             // is a package
	propertyUniverse                                              // is in the universe scope
	propertyGlobal                                                // is global
	propertyAddressable                                           // is addressable
	propertyIsNative                                              // is native
	propertyHasValue                                              // has a value
	propertyIsMacroDeclaration                                    // is macro declaration
	propertyMacroDeclaredInFileWithExtends                        // is macro declared in file with extends
)

// A typeInfo holds the type checking information. For example, every expression
// in the AST has an associated typeInfo. The typeInfo is also used in the type
// checker scopes to associate the declarations to the type checking
// information.
type typeInfo struct {
	Type              reflect.Type // Type.
	Alias             string       // Alias.
	Properties        properties   // Properties.
	Constant          constant     // Constant value.
	NativePackageName string       // Name of the package. Empty string if non-native.
	MethodType        methodType   // Method type.
	value             interface{}  // value; for packages has type *Package.
	valueType         reflect.Type // When value is a native type holds the original type of value.
}

// methodType represents the type of a method, intended as a combination of a
// method call/value/expression and a receiver type (concrete or interface).
type methodType uint8

const (
	noMethod             methodType = iota // Not a method.
	methodValueConcrete                    // Method value on a concrete receiver.
	methodValueInterface                   // Method value on an interface receiver.
	methodCallConcrete                     // Method call on concrete receiver.
	methodCallInterface                    // Method call on interface receiver.
)

// Nil reports whether it is the predeclared nil.
func (ti *typeInfo) Nil() bool {
	return ti.InUniverse() && ti.Untyped() && !ti.Addressable() && ti.Type == nil
}

// Untyped reports whether it is untyped.
func (ti *typeInfo) Untyped() bool {
	return ti.Properties&propertyUntyped != 0
}

// IsConstant reports whether it is a constant.
func (ti *typeInfo) IsConstant() bool {
	return ti.Constant != nil
}

// IsMacroDeclaration reports whether it is a macro declaration.
func (ti *typeInfo) IsMacroDeclaration() bool {
	return ti.Properties&propertyIsMacroDeclaration != 0
}

// IsUntypedConstant reports whether it is an untyped constant.
func (ti *typeInfo) IsUntypedConstant() bool {
	return ti.Properties&propertyUntyped != 0 && ti.Constant != nil
}

// IsType reports whether it is a type.
func (ti *typeInfo) IsType() bool {
	return ti.Properties&propertyIsType != 0
}

// IsFormatType reports whether it is a format type.
func (ti *typeInfo) IsFormatType() bool {
	return ti.Properties&propertyIsFormatType != 0
}

// IsPackage reports whether it is a package.
func (ti *typeInfo) IsPackage() bool {
	return ti.Properties&propertyIsPackage != 0
}

// InUniverse reports whether it is declared in the universe scope.
func (ti *typeInfo) InUniverse() bool {
	return ti.Properties&propertyUniverse != 0
}

// Global reports whether it is a global.
func (ti *typeInfo) Global() bool {
	return ti.Properties&propertyGlobal != 0
}

// Addressable reports whether it is addressable.
func (ti *typeInfo) Addressable() bool {
	return ti.Properties&propertyAddressable != 0
}

// IsNative reports whether it is native.
func (ti *typeInfo) IsNative() bool {
	return ti.Properties&propertyIsNative != 0
}

// IsBuiltinFunction reports whether it is a builtin function.
func (ti *typeInfo) IsBuiltinFunction() bool {
	return ti.Properties&propertyUniverse != 0 && ti.Properties&propertyUntyped == 0 && ti.Type == nil
}

// Itea reports whether it is itea.
func (ti *typeInfo) Itea() bool {
	return ti.Properties&propertyUniverse != 0 &&
		ti.Properties&propertyUntyped != 0 &&
		ti.Properties&propertyAddressable != 0 && ti.Type == nil
}

// MacroDeclaredInExtendingFile reports whether it is a macro declared in file
// with extends.
func (ti *typeInfo) MacroDeclaredInExtendingFile() bool {
	return ti.Properties&propertyMacroDeclaredInFileWithExtends != 0
}

// TypeName returns the name of the type. If it is an alias, it returns the
// name of the alias. Panics if it is not a type.
func (ti *typeInfo) TypeName() string {
	if !ti.IsType() {
		panic("not a type")
	}
	if ti.Alias != "" {
		return ti.Alias
	}
	return ti.Type.Name()
}

var runeType = reflect.TypeOf(rune(0))

// String returns a string representation.
func (ti *typeInfo) String() string {
	if ti.Nil() {
		return "nil"
	}
	var s string
	if ti.Untyped() {
		s = "untyped "
	}
	if ti.Type == nil {
		s += "unsigned number"
	} else {
		s += ti.Type.String()
	}
	return s
}

// ShortString returns a short string representation.
func (ti *typeInfo) ShortString() string {
	if ti.Nil() {
		return "nil"
	}
	if ti.IsConstant() && ti.Type == runeType {
		return "rune"
	}
	if ti.Type == nil {
		return "unsigned number"
	}
	return ti.Type.String()
}

// StringWithNumber returns the string representation of ti in the context of
// a function call, return statement and type switch assertion.
func (ti *typeInfo) StringWithNumber(explicitUntyped bool) string {
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

// IsBoolean reports whether it is boolean.
func (ti *typeInfo) IsBoolean() bool {
	return !ti.Nil() && ti.Type.Kind() == reflect.Bool
}

// IsNumeric reports whether it is numeric.
func (ti *typeInfo) IsNumeric() bool {
	if ti.Nil() {
		return false
	}
	k := ti.Type.Kind()
	return reflect.Int <= k && k <= reflect.Complex128
}

// IsInteger reports whether it is an integer.
func (ti *typeInfo) IsInteger() bool {
	if ti.Nil() {
		return false
	}
	k := ti.Type.Kind()
	return reflect.Int <= k && k <= reflect.Uintptr
}

// HasValue reports whether it has a value.
func (ti *typeInfo) HasValue() bool {
	return ti.Properties&propertyHasValue != 0
}

// setValue sets the value field with the ti's constant represented with the
// type typ. If typ is nil or is an interface type, the constant is
// represented with the type of ti. The valueType field is set with the type
// of value.
//
// If ti is not a constant, setValue does nothing. setValue panics if ti
// represents the predeclared nil.
//
// setValue is called at every point where a constant expression is used in a
// non-constant expression or in a statement. The following examples clarify
// the use of this method:
//
//   var i int64 = 20     call setValue on '20'    ctxType = int
//   x + 3                call setValue on '3'     ctxType = typeof(x)
//   x + y                no need to call setValue
//
func (ti *typeInfo) setValue(typ reflect.Type) {
	if ti.Nil() {
		panic("setValue called on the predeclared nil")
	}
	if !ti.IsConstant() {
		return
	}
	ti.Properties |= propertyHasValue
	ti.valueType = ti.Type
	if typ != nil && typ.Kind() != reflect.Interface {
		ti.valueType = typ
	}
	switch ti.valueType.Kind() {
	case reflect.Bool:
		if ti.Constant.bool() {
			ti.value = int64(1)
		} else {
			ti.value = int64(0)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ti.value = ti.Constant.int64()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		ti.value = int64(int(ti.Constant.uint64()))
	case reflect.Float32, reflect.Float64:
		ti.value = ti.Constant.float64()
	case reflect.Complex64, reflect.Complex128:
		c := ti.Constant.complex128()
		switch ti.valueType {
		case complex64Type:
			ti.value = complex64(c)
		case complex128Type:
			ti.value = c
		default:
			rv := reflect.New(ti.valueType).Elem()
			rv.SetComplex(c)
			ti.value = rv.Interface()
		}
	case reflect.String:
		ti.value = ti.Constant.string()
	}
}
