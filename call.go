// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"scrigo/ast"
)

// evalCall evaluates a call expression in a single-val context. It returns
// an error if the function
func (r *rendering) evalCall(node *ast.Call) interface{} {
	results, err := r.evalCallN(node, 1)
	if err != nil {
		panic(err)
	}
	return results[0].Interface()
}

// evalCallN evaluates a call expression in n-values context and returns its
// values. It returns an error if n > 0 and the function does not return n
// values.
func (r *rendering) evalCallN(node *ast.Call, n int) ([]reflect.Value, error) {
	// TODO(marco): manage the special case f(g(parameters_of_g)).
	// TODO(marco): manage the special case f(a, b...) in function call.

	if ident, ok := node.Func.(*ast.Identifier); ok {
		var isBuiltin bool
		for i := len(r.vars) - 1; i >= 0; i-- {
			if r.vars[i] != nil {
				if _, ok := r.vars[i][ident.Name]; ok {
					isBuiltin = i == 0
					break
				}
			}
		}
		if isBuiltin {
			switch ident.Name {
			case "append":
				return r.evalAppend(node, n)
			case "copy":
				return r.evalCopy(node, n)
			case "delete":
				return r.evalDelete(node, n)
			case "len":
				return r.evalLen(node, n)
			case "make":
				return r.evalMake(node, n)
			case "new":
				return r.evalNew(node, n)
			case "panic":
				return r.evalPanic(node, n)
			}
		}
	}

	var f, err = r.eval(node.Func)
	if err != nil {
		return nil, err
	}

	if f, ok := f.(function); ok {
		return r.evalCallFunc(node, f, n)
	}

	// Makes a type conversion.
	if typ, ok := f.(reflect.Type); ok {
		v, err := r.convert(node.Args[0], typ)
		if err != nil {
			panic(r.errorf(node.Args[0], "%s", err))
		}
		return []reflect.Value{reflect.ValueOf(v)}, nil
	}

	// Makes a call with reflect.
	fun := reflect.ValueOf(f)

	return r.evalReflectCall(node, fun, n)
}

// evalCallFunc evaluates a call expression in n-values context and returns
// its values. It returns an error if n > 0 and the function does not return n values.
func (r *rendering) evalCallFunc(node *ast.Call, fun function, n int) ([]reflect.Value, error) {

	var err error

	typ := fun.node.Type

	haveSize := len(node.Args)
	wantSize := len(typ.Parameters)

	var types []reflect.Type
	if wantSize > 0 {
		types = make([]reflect.Type, wantSize)
		for i := wantSize - 1; i >= 0; i-- {
			if t := typ.Parameters[i].Type; t == nil {
				types[i] = types[i+1]
			} else {
				types[i], err = r.evalType(t, noEllipses)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	var args = scope{}
	if params := typ.Parameters; wantSize > 0 {
		if last := params[wantSize-1]; last.Ident != nil {
			var final reflect.Value
			if typ.IsVariadic && last.Ident.Name != "_" {
				t := reflect.SliceOf(types[wantSize-1])
				l := haveSize - wantSize + 1
				final = reflect.MakeSlice(t, l, l)
				args[last.Ident.Name] = final.Interface()
			}
			for i, arg := range node.Args {
				v, err := r.eval(arg)
				if err != nil {
					return nil, err
				}
				var t reflect.Type
				if i < wantSize {
					t = types[i]
				} else {
					t = types[wantSize-1]
				}
				av, err := asAssignableTo(v, t)
				if err != nil {
					if err == errNotAssignable {
						err = r.errorf(arg, "cannot use %s (type %s) as type %s in argument to %s", arg, typeof(v), t, node.Func)
					}
					return nil, err
				}
				if !typ.IsVariadic || i < wantSize-1 {
					if ident := params[i].Ident; ident != nil && ident.Name != "_" {
						args[ident.Name] = av
					}
				} else if final.IsValid() {
					final.Index(i - wantSize + 1).Set(reflect.ValueOf(av))
				}
			}
		}
	}

	var vars []scope
	if ident := fun.node.Ident; ident != nil {
		sc := r.scope[fun.path]
		vars = []scope{r.vars[0], r.vars[1], r.vars[2], fun.upValues, sc, args}
	} else {
		vars = []scope{r.vars[0], r.vars[1], r.vars[2], fun.upValues, args}
	}

	rn := &rendering{
		scope:       r.scope,
		path:        fun.path,
		vars:        vars,
		treeContext: r.treeContext,
		handleError: r.handleError,
		function:    fun,
	}
	err = rn.render(nil, fun.node.Body.Nodes, nil)
	ret, ok := err.(returnError)
	if !ok {
		return nil, err
	}

	result := make([]reflect.Value, len(ret.args))
	for i, value := range ret.args {
		result[i] = reflect.ValueOf(value)
	}

	return result, nil
}

// evalReflectCall evaluates a call expression with reflect in n-values
// context and returns its values. It returns an error if n > 0 and the
// function does not return n values.
func (r *rendering) evalReflectCall(node *ast.Call, fun reflect.Value, n int) ([]reflect.Value, error) {

	if fun.IsNil() {
		return nil, r.errorf(node, "call of nil function")
	}

	typ := fun.Type()

	var numIn = typ.NumIn()
	var isVariadic = typ.IsVariadic()

	var args = make([]reflect.Value, len(node.Args))

	var lastIn = numIn - 1
	var in reflect.Type

	for i := 0; i < len(node.Args); i++ {

		if i < lastIn || !isVariadic {
			in = typ.In(i)
		} else if i == lastIn {
			in = typ.In(lastIn).Elem()
		}

		var arg, err = r.eval(node.Args[i])
		if err != nil {
			return nil, err
		}

		switch a := arg.(type) {
		case HTML:
			if in != htmlType && in == stringType {
				arg = string(a)
			}
		case ConstantNumber:
			arg, err = a.ToType(in)
			if err != nil {
				if e, ok := err.(errConstantOverflow); ok {
					return nil, r.errorf(node.Args[i], "%s", e)
				}
				return nil, r.errorf(node.Args[i], "%s in argument to %s", err, node.Func)
			}
		}

		if arg == nil {
			args[i] = reflect.Zero(in)
		} else {
			args[i] = reflect.ValueOf(arg)
		}

	}

	values, err := func() (_ []reflect.Value, err error) {
		defer func() {
			if e := recover(); e != nil {
				err = r.errorf(node.Func, "%s", e)
			}
		}()
		return fun.Call(args), nil
	}()

	return values, err
}

var errCannotConvert = errors.New("cannot convert")

// convert converts the val of expr to type typ.
//
// TODO(marco): convert must keep the type aliases, as rune and byte, in error messages.
// TODO(marco): implements conversions for custom numbers.
func (r *rendering) convert(expr ast.Expression, typ reflect.Type) (interface{}, error) {
	value := r.evalExpression(expr)
	converted, err := convert(value, typ)
	if err != nil {
		return nil, err
	}
	return converted, nil
}

// convert converts value to type typ.
// TODO (Gianluca): to review.
func convert(value interface{}, typ reflect.Type) (interface{}, error) {
	if value == nil {
		switch typ.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
			reflect.Ptr, reflect.UnsafePointer, reflect.Slice:
			return reflect.Zero(typ).Interface(), nil
		}
		return nil, fmt.Errorf("cannot convert nil to type %s", typ)
	}
	if n, ok := value.(ConstantNumber); ok {
		switch typ {
		case stringType:
			switch n.typ {
			case TypeInt, TypeRune:
				n, err := n.ToInt32()
				if err != nil {
					return "\uFFFD", nil
				}
				return string(n), nil
			}
		case intType:
			return n.ToInt()
		case int64Type:
			return n.ToInt64()
		case int32Type:
			return n.ToInt32()
		case int16Type:
			return n.ToInt16()
		case int8Type:
			return n.ToInt8()
		case uintType:
			return n.ToUint()
		case uint64Type:
			return n.ToUint64()
		case uint32Type:
			return n.ToUint32()
		case uint16Type:
			return n.ToUint16()
		case uint8Type:
			return n.ToUint8()
		default:
			n, err := n.ToTyped()
			if err != nil {
				return nil, err
			}
			rv := reflect.ValueOf(n)
			if rv.Type().ConvertibleTo(typ) {
				return rv.Convert(typ).Interface(), nil
			}
		}
	}
	switch typ {
	case stringType:
		switch v := value.(type) {
		case string, HTML:
			return value, nil
		case int:
			return string(v), nil
		case int64:
			return string(v), nil
		case int32:
			return string(v), nil
		case int16:
			return string(v), nil
		case int8:
			return string(v), nil
		case uint:
			return string(v), nil
		case uint64:
			return string(v), nil
		case uint32:
			return string(v), nil
		case uint16:
			return string(v), nil
		case uint8:
			return string(v), nil
		case []byte:
			return string(v), nil
		case []rune:
			return string(v), nil
		}
	case intType:
		v, err := convert(value, int64Type)
		if err != nil {
			return nil, err
		}
		return int(v.(int64)), nil
	case int64Type:
		switch v := value.(type) {
		case int64:
			return v, nil
		case int32:
			return int64(v), nil
		case int16:
			return int64(v), nil
		case int8:
			return int64(v), nil
		case uint:
			return int64(v), nil
		case uint64:
			return int64(v), nil
		case uint32:
			return int64(v), nil
		case uint16:
			return int64(v), nil
		case uint8:
			return int64(v), nil
		case float64:
			return int64(v), nil
		case float32:
			return int64(v), nil
		}
	case int32Type:
		v, err := convert(value, int64Type)
		if err != nil {
			return nil, err
		}
		return int32(v.(int64)), nil
	case int16Type:
		v, err := convert(value, int64Type)
		if err != nil {
			return nil, err
		}
		return int16(v.(int64)), nil
	case int8Type:
		v, err := convert(value, int64Type)
		if err != nil {
			return nil, err
		}
		return int8(v.(int64)), nil
	case uintType:
		v, err := convert(value, uint64Type)
		if err != nil {
			return nil, err
		}
		return uint(v.(uint64)), nil
	case uint64Type:
		switch v := value.(type) {
		case int64:
			return uint64(v), nil
		case int32:
			return uint64(v), nil
		case int16:
			return uint64(v), nil
		case int8:
			return uint64(v), nil
		case uint:
			return uint64(v), nil
		case uint64:
			return v, nil
		case uint32:
			return uint64(v), nil
		case uint16:
			return uint64(v), nil
		case uint8:
			return uint64(v), nil
		case float64:
			return uint64(v), nil
		case float32:
			return uint64(v), nil
		}
	case uint32Type:
		v, err := convert(value, uint64Type)
		if err != nil {
			return nil, err
		}
		return uint32(v.(uint64)), nil
	case uint16Type:
		v, err := convert(value, uint64Type)
		if err != nil {
			return nil, err
		}
		return uint16(v.(uint64)), nil
	case uint8Type:
		v, err := convert(value, uint64Type)
		if err != nil {
			return nil, err
		}
		return uint8(v.(uint64)), nil
	case float64Type:
		switch v := value.(type) {
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case int32:
			return float64(v), nil
		case int16:
			return float64(v), nil
		case int8:
			return float64(v), nil
		case uint:
			return float64(v), nil
		case uint64:
			return float64(v), nil
		case uint32:
			return float64(v), nil
		case uint16:
			return float64(v), nil
		case uint8:
			return float64(v), nil
		case float32:
			return float64(v), nil
		case ConstantNumber:
			return v.ToFloat64(), nil
		}
	case float32Type:
		v, err := convert(value, float64Type)
		if err != nil {
			return nil, err
		}
		return float32(v.(float64)), nil
	case bytesType:
		switch v := value.(type) {
		case nil:
			return []byte(nil), nil
		case string:
			return []byte(v), nil
		case HTML:
			return []byte(v), nil
		case []byte:
			return v, nil
		}
	case interfaceType:
		return value, nil
	case runesType:
		switch v := value.(type) {
		case nil:
			return []rune(nil), nil
		case string:
			return []rune(v), nil
		case HTML:
			return []rune(v), nil
		case []rune:
			return v, nil
		}
	default:
		// TODO(marco): manage T(nil), now it panics
		rv := reflect.ValueOf(value)
		if rv.Type().ConvertibleTo(typ) {
			return rv.Convert(typ).Interface(), nil
		}
	}
	return nil, errCannotConvert
}

// evalAppend evaluates the append builtin function.
func (r *rendering) evalAppend(node *ast.Call, n int) ([]reflect.Value, error) {

	slice, err := r.eval(node.Args[0])
	if err != nil {
		return nil, err
	}

	t := reflect.TypeOf(slice)

	m := len(node.Args) - 1
	if m == 0 {
		return []reflect.Value{reflect.ValueOf(slice)}, nil
	}

	typ := t.Elem()

	if s, ok := slice.([]interface{}); ok {
		var s2 []interface{}
		l, c := len(s), cap(s)
		p := 0
		if l+m <= c {
			s2 = make([]interface{}, m)
			s = s[:c:c]
		} else {
			s2 = make([]interface{}, l+m)
			copy(s2, s)
			s = s2
			p = l
		}
		for i := 1; i < len(node.Args); i++ {
			v, err := r.eval(node.Args[i])
			if err != nil {
				return nil, err
			}
			if n, ok := v.(ConstantNumber); ok {
				v, err = n.ToType(typ)
				if err != nil {
					if e, ok := err.(errConstantOverflow); ok {
						return nil, r.errorf(node.Args[i], "%s", e)
					}
					return nil, r.errorf(node.Args[i], "%s in argument to %s", err, node.Func)
				}
			}
			s2[p+i-1] = v
		}
		if l+m <= c {
			copy(s[l:], s2)
		}
		return []reflect.Value{reflect.ValueOf(s)}, nil
	}

	sv := reflect.ValueOf(slice)
	var sv2 reflect.Value
	l, c := sv.Len(), sv.Cap()
	p := 0
	if l+n <= c {
		sv2 = reflect.MakeSlice(t, m, m)
		sv = sv.Slice3(0, c, c)
	} else {
		sv2 = reflect.MakeSlice(t, l+m, l+m)
		reflect.Copy(sv2, sv)
		sv = sv2
		p = l
	}
	for i := 1; i < len(node.Args); i++ {
		v, err := r.eval(node.Args[i])
		if err != nil {
			return nil, err
		}
		if n, ok := v.(ConstantNumber); ok {
			v, err = n.ToType(typ)
			if err != nil {
				if e, ok := err.(errConstantOverflow); ok {
					return nil, r.errorf(node.Args[i], "%s", e)
				}
				return nil, r.errorf(node.Args[i], "%s in argument to %s", err, node.Func)
			}
		}
		sv2.Index(p + i - 1).Set(reflect.ValueOf(v))
	}
	if l+m <= c {
		reflect.Copy(sv2.Slice(l, l+m+1), sv2)
	}

	return []reflect.Value{sv}, nil
}

// evalCopy evaluates the copy builtin function.
func (r *rendering) evalCopy(node *ast.Call, n int) ([]reflect.Value, error) {
	dst := r.evalExpression(node.Args[0])
	src := r.evalExpression(node.Args[1])
	switch d := dst.(type) {
	case []interface{}:
		s, ok := src.([]interface{})
		if ok {
			n := copy(d, s)
			return []reflect.Value{reflect.ValueOf(n)}, nil
		}
	case []string:
		s, ok := src.([]string)
		if ok {
			n := copy(d, s)
			return []reflect.Value{reflect.ValueOf(n)}, nil
		}
	case []int:
		s, ok := src.([]int)
		if ok {
			n := copy(d, s)
			return []reflect.Value{reflect.ValueOf(n)}, nil
		}
	case []byte:
		switch s := src.(type) {
		case []byte:
			n := copy(d, s)
			return []reflect.Value{reflect.ValueOf(n)}, nil
		case string:
			n := copy(d, s)
			return []reflect.Value{reflect.ValueOf(n)}, nil
		}
	}
	d := reflect.ValueOf(dst)
	s := reflect.ValueOf(src)
	n = reflect.Copy(d, s)
	return []reflect.Value{reflect.ValueOf(n)}, nil
}

// evalDelete evaluates the delete builtin function.
func (r *rendering) evalDelete(node *ast.Call, n int) ([]reflect.Value, error) {
	m := r.evalExpression(node.Args[0])
	k, err := r.mapIndex(node.Args[1], interfaceType)
	if err != nil {
		return nil, err
	}
	// TODO: gestire il caso delle nil map
	switch v := m.(type) {
	case map[interface{}]interface{}:
		delete(v, k)
	case map[string]string:
		delete(v, k.(string))
	case map[string]int:
		delete(v, k.(string))
	case map[string]interface{}:
		delete(v, k.(string))

		// TODO (Gianluca): how can this error happen?
		// default:
		// 	rv := reflect.ValueOf(v)
		// 	k := reflect.ValueOf(k)
		// 	if k.Type().AssignableTo(rv.Type().Key()) {
		// 		return nil, r.errorf(node, "cannot use %v (type %s) as type %s in delete", k, typeof(k), rv.Type().Key())
		// 	}
		// 	rv.SetMapIndex(k, reflect.Show{})
	}
	return nil, nil
}

// evalLen evaluates the len builtin function.
func (r *rendering) evalLen(node *ast.Call, n int) ([]reflect.Value, error) {
	arg := node.Args[0]
	v := r.evalExpression(arg)
	var length int
	switch s := v.(type) {
	case string:
		length = len(s)
	case HTML:
		length = len(string(s))
	case []interface{}:
		length = len(s)
	case []string:
		length = len(s)
	case []HTML:
		length = len(s)
	case []int:
		length = len(s)
	case []byte:
		length = len(s)
	case []bool:
		length = len(s)
	case map[string]string:
		length = len(s)
	case map[string]int:
		length = len(s)
	case map[string]interface{}:
		length = len(s)
	default:
		var rv = reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice:
			length = rv.Len()
		case reflect.Array:
			length = rv.Len()
		case reflect.Map:
			length = rv.Len()
		}
	}
	return []reflect.Value{reflect.ValueOf(length)}, nil
}

func (r *rendering) evalMake(node *ast.Call, n int) ([]reflect.Value, error) {
	typ, err := r.evalType(node.Args[0], noEllipses)
	if err != nil {
		return nil, err
	}
	switch typ.Kind() {
	case reflect.Slice:
		// make([]T, len)
		// make([]T, len, cap)
		rawLength := r.evalExpression(node.Args[1])
		if cn, ok := rawLength.(ConstantNumber); ok {
			rawLength, err = cn.ToType(intType)
			if err != nil {
				return nil, err
			}
		}
		length := rawLength.(int)
		var capacity int
		if len(node.Args) >= 3 {
			// make([]T, len, cap)
			rawCap := r.evalExpression(node.Args[2])
			if cn, ok := rawCap.(ConstantNumber); ok {
				rawCap, err = cn.ToType(intType)
				if err != nil {
					return nil, err
				}
			}
			capacity = rawCap.(int)
		} else {
			capacity = length
		}
		return []reflect.Value{reflect.MakeSlice(typ, length, capacity)}, nil
	case reflect.Map:
		var size int
		if len(node.Args) == 1 {
			// make(map[T1]T2)
			return []reflect.Value{reflect.MakeMap(typ)}, nil
		}
		// make(map[T1]T2, size)
		rawSize := r.evalExpression(node.Args[1])
		if cn, ok := rawSize.(ConstantNumber); ok {
			rawSize, err = cn.ToType(intType)
			if err != nil {
				return nil, err
			}
		}
		size = rawSize.(int)
		return []reflect.Value{reflect.MakeMapWithSize(typ, size)}, nil
	}
	return nil, nil
}

// evalNew evaluates the new builtin function.
func (r *rendering) evalNew(node *ast.Call, n int) ([]reflect.Value, error) {
	typ, err := r.evalType(node.Args[0], noEllipses)
	if err != nil {
		return nil, err
	}
	return []reflect.Value{reflect.New(typ)}, nil
}

// ErrorPanic is the error representing the panic condition.
type ErrorPanic struct {
	err  error    // error message.
	node ast.Node // ast node which caused panic.
}

func (e ErrorPanic) Error() string {
	msg := `panic: {{err}}

goroutine [running]:
	{{panicPos}}
`
	repl := strings.NewReplacer(
		"{{err}}", e.err.Error(),
		"{{panicPos}}", e.node.Pos().String(),
	)
	return repl.Replace(msg)
}

// evalPanic evaluates the panic builtin function.
func (r *rendering) evalPanic(node *ast.Call, n int) ([]reflect.Value, error) {
	v, err := r.eval(node.Args[0])
	if err != nil {
		return nil, err
	}
	// TODO (Gianluca): how to convert v to string properly? This conversion
	// does not work for constant numbers.
	return nil, ErrorPanic{fmt.Errorf("%v", v), node}
}
