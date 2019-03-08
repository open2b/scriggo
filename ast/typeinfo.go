// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast

import (
	"go/constant"
	"reflect"
)

type UntypedValue struct {
	DefaultType reflect.Kind
	Bool        bool
	String      string
	Number      constant.Value
}

type Properties uint8

const (
	PropertyNil         Properties = 1 << (8 - 1 - iota) // is predeclared nil
	PropertyIsType                                       // is a type
	PropertyIsPackage                                    // is a package
	PropertyIsBuiltin                                    // is a builtin
	PropertyAddressable                                  // is addressable
)

type TypeInfo struct {
	Type       reflect.Type // Type; nil if untyped.
	Properties Properties   // Properties.
	Value      interface{}  // Constant value; nil if not constant.
	Package    *Package     // Package; nil if not a package.
}

// Nil reports whether it is the predeclared nil.
func (ti *TypeInfo) Nil() bool {
	return ti.Properties&PropertyNil != 0
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

// Addressable reports whether it is a addressable.
func (ti *TypeInfo) Addressable() bool {
	return ti.Properties&PropertyAddressable != 0
}

// Kind returns the kind of the type, the default type for untyped
// constants and reflect.Bool for the untyped bool.
//
// Returns reflect.Invalid for the predeclared nil, packages and types.
func (ti *TypeInfo) Kind() reflect.Kind {
	if ti.Nil() || ti.IsPackage() || ti.IsType() {
		return reflect.Invalid
	}
	if ti.Type == nil {
		if ti.Value == nil {
			return reflect.Bool
		}
		return ti.Value.(*UntypedValue).DefaultType
	}
	return ti.Type.Kind()
}

// String returns a string representation.
func (ti *TypeInfo) String() string {
	if ti.Nil() {
		return "nil"
	}
	var s string
	if ti.Type == nil {
		s = "untyped "
	}
	if ti.Value != nil {
		switch v := ti.Value.(type) {
		case bool:
			return s + "bool"
		case string:
			return s + "string"
		case *UntypedValue:
			if v.DefaultType == reflect.Int32 {
				return s + "rune"
			}
			return s + v.DefaultType.String()
		}
	}
	if ti.Type == nil {
		return s + "bool"
	}
	return ti.Type.String()
}

// ShortString returns a short string representation.
func (ti *TypeInfo) ShortString() string {
	switch {
	case ti.Nil():
		return "nil"
	case ti.Value != nil:
		switch v := ti.Value.(type) {
		case bool:
			return "bool"
		case string:
			return "string"
		case *UntypedValue:
			if v.DefaultType == reflect.Int32 {
				return "rune"
			}
			return v.DefaultType.String()
		}
	case ti.Type == nil:
		return "bool"
	}
	return ti.Type.String()
}
