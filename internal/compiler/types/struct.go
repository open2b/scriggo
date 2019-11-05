// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

func StructOf(fields []reflect.StructField) reflect.Type {

	// baseFields contains all the fields with their 'Go' type; if there are not
	// any Scriggo types in fields then baseFields contains exactly the content
	// of fields.
	//
	// This slice is always dense.
	//
	baseFields := make([]reflect.StructField, len(fields))

	// scriggoFields contains the struct fields that cannot be represented by
	// the Go reflect because they are Scriggo types.
	//
	// This is a map (where the key is the struct field index) because can be
	// sparse or empty.
	//
	scriggoFields := map[int]reflect.StructField{}

	for i, field := range fields {

		// If field has a Scriggo type then it will be changed by the next
		// check.
		baseFields[i] = field

		// This field cannot be represented using the reflect: remove the
		// information from the type and add that information to the structType.
		if st, ok := field.Type.(ScriggoType); ok {
			baseFields[i].Type = st.Underlying()
			scriggoFields[i] = field
		}

	}

	// Every field can be represented by the builtin reflect: no need to create
	// a structType.
	if len(scriggoFields) == 0 {
		return reflect.StructOf(fields)
	}

	return structType{
		Type:          StructOf(baseFields),
		scriggoFields: &scriggoFields,
	}

}

type structType struct {
	reflect.Type
	scriggoFields *map[int]reflect.StructField
}

func (x structType) AssignableTo(T reflect.Type) bool {
	return x == T
}

func (x structType) Underlying() reflect.Type {
	return x.Type
}

func (x structType) Field(i int) reflect.StructField {
	if field, ok := (*x.scriggoFields)[i]; ok {
		return field
	}
	return x.Type.Field(i)
}

func (x structType) FieldByIndex(index []int) reflect.StructField {
	panic("TODO: not implemented") // TODO(Gianluca): to implement.
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

func (x structType) NumField() int {
	return x.Type.NumField()
}

func (x structType) String() string {
	s := "struct { "
	for i := 0; i < x.NumField(); i++ {
		field := x.Field(i)
		s += field.Name + " "
		if scriggoField, ok := (*x.scriggoFields)[i]; ok {
			s += scriggoField.Type.String()
		} else {
			s += field.Type.Name()
		}
		if i != x.NumField()-1 {
			s += "; "
		} else {
			s += " "
		}
	}
	s += "}"
	return s
}
