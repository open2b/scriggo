// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"

	"github.com/open2b/scriggo/internal/runtime"
)

// StructOf behaves like reflect.StructOf except when at least one of the
// fields has a Scriggo type; in such case a new Scriggo struct type is
// created and returned as reflect.Type.
func (types *Types) StructOf(fields []reflect.StructField) reflect.Type {
	// Check if there are fields with a Scriggo type.
	var scriggoFields map[int]reflect.StructField
	for i, field := range fields {
		if _, ok := field.Type.(runtime.ScriggoType); ok {
			f := field
			f.Index = []int{i}
			if scriggoFields == nil {
				scriggoFields = map[int]reflect.StructField{i: f}
			} else {
				scriggoFields[i] = f
			}
		}
	}
	if scriggoFields == nil {
		// All the fields have a Go type.
		return reflect.StructOf(fields)
	}
	// Some fields have a Scriggo type.
	goFields := make([]reflect.StructField, len(fields))
	for i, field := range fields {
		goFields[i] = field
		if f, ok := field.Type.(runtime.ScriggoType); ok {
			goFields[i].Type = f.GoType()
		}
	}
	return structType{
		Type:          reflect.StructOf(goFields),
		scriggoFields: types.addFields(scriggoFields),
	}
}

func equalFields(fs1, fs2 map[int]reflect.StructField) bool {
	return reflect.DeepEqual(fs1, fs2)
}

// addFields adds a list of struct fields to the cache if not already present
// or returns the found one. This avoids duplication of struct fields by
// ensuring that every pointer to map returned by this method is equal if and
// only if the underlying map is equal.
func (types *Types) addFields(fields map[int]reflect.StructField) *map[int]reflect.StructField {
	for _, storedFields := range types.structFieldsLists {
		if equalFields(*storedFields, fields) {
			return storedFields
		}
	}
	// Not found.
	newFields := &fields
	types.structFieldsLists = append(types.structFieldsLists, newFields)
	return newFields
}

// structType represents a composite struct type where at least one of the
// fields has a Scriggo type.
type structType struct {
	reflect.Type
	scriggoFields *map[int]reflect.StructField
}

func (x structType) AssignableTo(y reflect.Type) bool {
	return AssignableTo(x, y)
}

func (x structType) ConvertibleTo(y reflect.Type) bool {
	return ConvertibleTo(x, y)
}

func (x structType) Field(i int) reflect.StructField {
	if field, ok := (*x.scriggoFields)[i]; ok {
		return field
	}
	return x.Type.Field(i)
}

func (x structType) FieldByIndex(index []int) reflect.StructField {
	var field reflect.StructField
	t := x.Type
	for i, fi := range index {
		if i > 0 {
			t = field.Type
			if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
				t = t.Elem()
			}
		}
		field = t.Field(fi)
	}
	return field
}

func (x structType) FieldByName(name string) (reflect.StructField, bool) {
	for _, field := range *x.scriggoFields {
		if field.Name == name {
			return field, true
		}
	}
	return x.Type.FieldByName(name)
}

func (x structType) FieldByNameFunc(match func(string) bool) (reflect.StructField, bool) {
	panic("TODO: not implemented") // TODO(Gianluca): to implement.
}

func (x structType) Implements(y reflect.Type) bool {
	return Implements(x, y)
}

func (x structType) Name() string {
	return "" // composite types do not have a name.
}

func (x structType) NumField() int {
	return x.Type.NumField()
}

func (x structType) String() string {
	s := "struct { "
	for i := 0; i < x.NumField(); i++ {
		if i > 0 {
			s += "; "
		}
		field := x.Field(i)
		s += field.Name + " "
		if scriggoField, ok := (*x.scriggoFields)[i]; ok {
			s += scriggoField.Type.String()
		} else {
			s += field.Type.Name()
		}
	}
	s += " }"
	return s
}

// GoType implements the interface runtime.ScriggoType.
func (x structType) GoType() reflect.Type {
	assertNotScriggoType(x.Type)
	return x.Type
}

// Unwrap implements the interface runtime.ScriggoType.
func (x structType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }

// Wrap implements the interface runtime.ScriggoType.
func (x structType) Wrap(v reflect.Value) reflect.Value { return wrap(x, v) }
