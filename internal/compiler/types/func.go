// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"

	"github.com/open2b/scriggo/internal/runtime"
)

// FuncOf behaves like reflect.FuncOf except when at least one of the
// parameters is a Scriggo type; in such case a new Scriggo function type is
// created and returned as reflect.Type.
func (types *Types) FuncOf(in, out []reflect.Type, variadic bool) reflect.Type {

	// If at least one parameter of the function that is going to be created
	// is a Scriggo type then such function must be represented with a Scriggo
	// type. If not, the resulting function can be represented by the reflect
	// implementation of reflect.Type that can be created with reflect.FuncOf.

	if !containsScriggoType(in) && !containsScriggoType(out) {
		return reflect.FuncOf(in, out, variadic)
	}

	inGo := make([]reflect.Type, len(in))
	outGo := make([]reflect.Type, len(out))

	for i := range in {
		if st, ok := in[i].(runtime.ScriggoType); ok {
			inGo[i] = st.GoType()
		} else {
			inGo[i] = in[i]
		}
	}
	for i := range out {
		if st, ok := out[i].(runtime.ScriggoType); ok {
			outGo[i] = st.GoType()
		} else {
			outGo[i] = out[i]
		}
	}

	return funcType{
		in:   types.funcParamsStore.deduplicate(in),
		out:  types.funcParamsStore.deduplicate(out),
		Type: reflect.FuncOf(inGo, outGo, variadic),
	}

}

// funcParams stores and provides the function parameters necessary to create
// a funcType, ensuring that two parameters list have the same pointer if and
// only if their elements are the same.
//
// This consequently ensures that the comparison between two pointers to
// parameters list returns a consistent result.
type funcParams struct {
	params []*[]reflect.Type
}

// deduplicate deduplicates the given parameters by returning a pointer to a
// slice of parameters that contains the same elements.
func (store *funcParams) deduplicate(params []reflect.Type) *[]reflect.Type {
	for _, storedParams := range store.params {
		if equalReflectTypes(*storedParams, params) {
			return storedParams
		}
	}
	// Make a copy of params before getting the pointer.
	newParams := make([]reflect.Type, len(params))
	for i := range params {
		newParams[i] = params[i]
	}
	newParamsPtr := &newParams
	store.params = append(store.params, newParamsPtr)
	return newParamsPtr
}

// equalReflectTypes reports whether ts1 and ts2 are equal, i.e. they contain
// the same elements at the same position.
func equalReflectTypes(ts1, ts2 []reflect.Type) bool {
	if len(ts1) != len(ts2) {
		return false
	}
	for i := range ts1 {
		if ts1[i] != ts2[i] {
			return false
		}
	}
	return true
}

// containsScriggoType reports whether types contains at least one Scriggo
// type.
func containsScriggoType(types []reflect.Type) bool {
	for _, t := range types {
		if _, ok := t.(runtime.ScriggoType); ok {
			return true
		}
	}
	return false
}

type funcType struct {
	reflect.Type

	// At least one of in and out contains at least one Scriggo type,
	// otherwise there's no need to create a funcType. in and out must be
	// pointers because a funcType value must be comparable.
	in, out *[]reflect.Type
}

func (x funcType) AssignableTo(y reflect.Type) bool {
	return AssignableTo(x, y)
}

func (x funcType) ConvertibleTo(y reflect.Type) bool {
	return ConvertibleTo(x, y)
}

func (x funcType) Implements(y reflect.Type) bool {
	return Implements(x, y)
}

func (x funcType) In(i int) reflect.Type {
	if x.in != nil {
		return (*x.in)[i]
	}
	return x.Type.In(i)
}

func (x funcType) Kind() reflect.Kind {
	return reflect.Func
}

func (x funcType) Name() string {
	return "" // composite types do not have a name.
}

func (x funcType) Out(i int) reflect.Type {
	if x.out != nil {
		return (*x.out)[i]
	}
	return x.Type.Out(i)
}

// TODO: does this function handle variadic functions properly?
func (x funcType) String() string {
	s := "func("
	for i, t := range *x.in {
		if i > 0 {
			s += ", "
		}
		s += t.String()
	}
	s += ") "
	if len(*x.out) >= 2 {
		s += "("
	}
	for i, t := range *x.out {
		if i > 0 {
			s += ", "
		}
		s += t.String()
	}
	if len(*x.out) >= 2 {
		s += ")"
	}
	return s
}

// GoType implements the interface runtime.ScriggoType.
func (x funcType) GoType() reflect.Type {
	assertNotScriggoType(x.Type)
	return x.Type
}

// Unwrap implements the interface runtime.ScriggoType.
func (x funcType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }

// Wrap implements the interface runtime.ScriggoType.
func (x funcType) Wrap(v reflect.Value) reflect.Value { return wrap(x, v) }
