// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"open2b/template/ast"

	"github.com/shopspring/decimal"
)

var stringType = reflect.TypeOf("")
var intType = reflect.TypeOf(0)
var float64Type = reflect.TypeOf(0.0)
var boolType = reflect.TypeOf(false)

var zero = decimal.New(0, 0)
var decimalType = reflect.TypeOf(zero)

// eval evaluates an expression and returns its value.
func (r *rendering) eval(exp ast.Expression) (value interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				panic(r)
			}
		}
	}()
	return r.evalExpression(exp), nil
}

// evalExpression evaluates an expression and returns its value.
// In the event of an error, calls panic with the error as parameter.
func (r *rendering) evalExpression(expr ast.Expression) interface{} {
	switch e := expr.(type) {
	case *ast.String:
		return e.Text
	case *ast.Int:
		return e.Value
	case *ast.Number:
		return e.Value
	case *ast.Parentesis:
		return r.evalExpression(e.Expr)
	case *ast.UnaryOperator:
		return r.evalUnaryOperator(e)
	case *ast.BinaryOperator:
		return r.evalBinaryOperator(e)
	case *ast.Identifier:
		return r.evalIdentifier(e)
	case *ast.Call:
		return r.evalCall(e)
	case *ast.Index:
		return r.evalIndex(e)
	case *ast.Slice:
		return r.evalSlice(e)
	case *ast.Selector:
		return r.evalSelector(e)
	default:
		panic(r.errorf(expr, "unexpected node type %#v", expr))
	}
}

// evalUnaryOperator evaluates a unary operator and returns its value.
// On error it calls panic with the error as argument.
func (r *rendering) evalUnaryOperator(node *ast.UnaryOperator) interface{} {
	var e = asBase(r.evalExpression(node.Expr))
	switch node.Op {
	case ast.OperatorNot:
		if b, ok := e.(bool); ok {
			return !b
		}
		panic(r.errorf(node, "invalid operation: ! %s", typeof(e)))
	case ast.OperatorAddition:
		if _, ok := e.(int); ok {
			return e
		}
		if _, ok := e.(decimal.Decimal); ok {
			return e
		}
		panic(r.errorf(node, "invalid operation: + %s", typeof(e)))
	case ast.OperatorSubtraction:
		if n, ok := e.(int); ok {
			return -n
		}
		if n, ok := e.(decimal.Decimal); ok {
			return n.Neg()
		}
		panic(r.errorf(node, "invalid operation: - %s", typeof(e)))
	}
	panic("Unknown Unary Operator")
}

// evalBinaryOperator evaluates a binary operator and returns its value.
// On error it calls panic with the error as argument.
func (r *rendering) evalBinaryOperator(node *ast.BinaryOperator) interface{} {

	expr1 := asBase(r.evalExpression(node.Expr1))

	switch node.Op {

	case ast.OperatorEqual:
		expr2 := asBase(r.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			defer func() {
				if recover() != nil {
					panic(r.errorf(node, "invalid operation: %s == %s", typeof(expr1), typeof(expr2)))
				}
			}()
			if expr2 == nil {
				return reflect.ValueOf(expr1).IsNil()
			}
			return reflect.ValueOf(expr2).IsNil()
		} else {
			expr1, expr2 = htmlToStringType(expr1, expr2)
			switch e1 := expr1.(type) {
			case bool:
				if e2, ok := expr2.(bool); ok {
					return e1 == e2
				}
			case string:
				if e2, ok := expr2.(string); ok {
					return e1 == e2
				}
			case int:
				if e2, ok := expr2.(int); ok {
					return e1 == e2
				}
				if e2, ok := expr2.(decimal.Decimal); ok {
					return decimal.New(int64(e1), 0).Cmp(e2) == 0
				}
			case decimal.Decimal:
				if e2, ok := expr2.(decimal.Decimal); ok {
					return e1.Equal(e2)
				}
				if e2, ok := expr2.(int); ok {
					return e1.Cmp(decimal.New(int64(e2), 0)) == 0
				}
			}
		}
		panic(r.errorf(node, "invalid operation: %s == %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorNotEqual:
		expr2 := asBase(r.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			defer func() {
				if recover() != nil {
					panic(r.errorf(node, "invalid operation: %s != %s", typeof(expr1), typeof(expr2)))
				}
			}()
			if expr2 == nil {
				return !reflect.ValueOf(expr1).IsNil()
			}
			return !reflect.ValueOf(expr2).IsNil()
		} else {
			expr1, expr2 = htmlToStringType(expr1, expr2)
			switch e1 := expr1.(type) {
			case bool:
				if e2, ok := expr2.(bool); ok {
					return e1 != e2
				}
			case string:
				if e2, ok := expr2.(string); ok {
					return e1 != e2
				}
			case int:
				if e2, ok := expr2.(int); ok {
					return e1 != e2
				}
				if e2, ok := expr2.(decimal.Decimal); ok {
					return decimal.New(int64(e1), 0).Cmp(e2) != 0
				}
			case decimal.Decimal:
				if e2, ok := expr2.(decimal.Decimal); ok {
					return !e1.Equal(e2)
				}
				if e2, ok := expr2.(int); ok {
					return e1.Cmp(decimal.New(int64(e2), 0)) != 0
				}
			}
		}
		panic(r.errorf(node, "invalid operation: %s != %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorLess:
		expr2 := asBase(r.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(r.errorf(node, "invalid operation: %s < %s", typeof(expr1), typeof(expr2)))
		}
		expr1, expr2 = htmlToStringType(expr1, expr2)
		switch e1 := expr1.(type) {
		case string:
			if e2, ok := expr2.(string); ok {
				return e1 < e2
			}
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 < e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).Cmp(e2) < 0
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.Cmp(e2) < 0
			}
			if e2, ok := expr2.(int); ok {
				return e1.Cmp(decimal.New(int64(e2), 0)) < 0
			}
		}
		panic(fmt.Sprintf("invalid operation: %s < %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorLessOrEqual:
		expr2 := asBase(r.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(r.errorf(node, "invalid operation: %s <= %s", typeof(expr1), typeof(expr2)))
		}
		expr1, expr2 = htmlToStringType(expr1, expr2)
		switch e1 := expr1.(type) {
		case string:
			if e2, ok := expr2.(string); ok {
				return e1 <= e2
			}
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 <= e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).Cmp(e2) <= 0
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.Cmp(e2) <= 0
			}
			if e2, ok := expr2.(int); ok {
				return e1.Cmp(decimal.New(int64(e2), 0)) <= 0
			}
		}
		panic(r.errorf(node, "invalid operation: %s <= %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorGreater:
		expr2 := asBase(r.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(r.errorf(node, "invalid operation: %s > %s", typeof(expr1), typeof(expr2)))
		}
		expr1, expr2 = htmlToStringType(expr1, expr2)
		switch e1 := expr1.(type) {
		case string:
			if e2, ok := expr2.(string); ok {
				return e1 > e2
			}
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 > e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).Cmp(e2) > 0
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.Cmp(e2) > 0
			}
			if e2, ok := expr2.(int); ok {
				return e1.Cmp(decimal.New(int64(e2), 0)) > 0
			}
		}
		panic(r.errorf(node, "invalid operation: %s > %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorGreaterOrEqual:
		expr2 := asBase(r.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(r.errorf(node, "invalid operation: %s >= %s", typeof(expr1), typeof(expr2)))
		}
		expr1, expr2 = htmlToStringType(expr1, expr2)
		switch e1 := expr1.(type) {
		case string:
			if e2, ok := expr2.(string); ok {
				return e1 >= e2
			}
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 >= e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).Cmp(e2) >= 0
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.Cmp(e2) >= 0
			}
			if e2, ok := expr2.(int); ok {
				return e1.Cmp(decimal.New(int64(e2), 0)) >= 0
			}
		}
		panic(r.errorf(node, "invalid operation: %s >= %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorAnd:
		if e1, ok := expr1.(bool); ok {
			if !e1 {
				return false
			}
			expr2 := asBase(r.evalExpression(node.Expr2))
			if e2, ok := expr2.(bool); ok {
				return e1 && e2
			}
			panic(r.errorf(node, "invalid operation: %s && %s", typeof(expr1), typeof(expr2)))
		}
		panic(r.errorf(node, "invalid operation: %s && ...", typeof(expr1)))

	case ast.OperatorOr:
		if e1, ok := expr1.(bool); ok {
			if e1 {
				return true
			}
			expr2 := asBase(r.evalExpression(node.Expr2))
			if e2, ok := expr2.(bool); ok {
				return e1 || e2
			}
			panic(r.errorf(node, "invalid operation: %s || %s", typeof(expr1), typeof(expr2)))
		}
		panic(r.errorf(node, "invalid operation: %s || ...", typeof(expr1)))

	case ast.OperatorAddition:
		expr2 := asBase(r.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(r.errorf(node, "invalid operation: %s + %s", typeof(expr1), typeof(expr2)))
		}
		switch e1 := expr1.(type) {
		case string:
			switch e2 := expr2.(type) {
			case string:
				return e1 + e2
			case HTML:
				return HTML(htmlEscapeString(e1) + string(e2))
			}
		case HTML:
			switch e2 := expr2.(type) {
			case string:
				return HTML(string(e1) + htmlEscapeString(e2))
			case HTML:
				return HTML(string(e1) + string(e2))
			}
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 + e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).Add(e2)
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.Add(e2)
			}
			if e2, ok := expr2.(int); ok {
				return e1.Add(decimal.New(int64(e2), 0))
			}
		}
		panic(r.errorf(node, "invalid operation: %s + %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorSubtraction:
		expr2 := asBase(r.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(r.errorf(node, "invalid operation: %s - %s", typeof(expr1), typeof(expr2)))
		}
		switch e1 := expr1.(type) {
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 - e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).Sub(e2)
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.Sub(e2)
			}
			if e2, ok := expr2.(int); ok {
				return e1.Sub(decimal.New(int64(e2), 0))
			}
		}
		panic(r.errorf(node, "invalid operation: %s - %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorMultiplication:
		expr2 := asBase(r.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(r.errorf(node, "invalid operation: %s * %s", typeof(expr1), typeof(expr2)))
		}
		switch e1 := expr1.(type) {
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 * e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).Mul(e2)
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.Mul(e2)
			}
			if e2, ok := expr2.(int); ok {
				return e1.Mul(decimal.New(int64(e2), 0))
			}
		}
		panic(r.errorf(node, "invalid operation: %s * %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorDivision:
		expr2 := asBase(r.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(r.errorf(node, "invalid operation: %s / %s", typeof(expr1), typeof(expr2)))
		}
		switch e1 := expr1.(type) {
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 / e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).DivRound(e2, 20)
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.DivRound(e2, 20)
			}
			if e2, ok := expr2.(int); ok {
				return e1.DivRound(decimal.New(int64(e2), 0), 20)
			}
		}
		panic(r.errorf(node, "invalid operation: %s / %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorModulo:
		expr2 := asBase(r.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(r.errorf(node, "invalid operation: %s %% %s", typeof(expr1), typeof(expr2)))
		}
		switch e1 := expr1.(type) {
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 % e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).Mod(e2)
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.Mod(e2)
			}
			if e2, ok := expr2.(int); ok {
				return e1.Mod(decimal.New(int64(e2), 0))
			}
		}
		panic(r.errorf(node, "invalid operation: %s %% %s", typeof(expr1), typeof(expr2)))

	}

	panic("unknown binary operator")
}

// evalSelector evaluates a selector expression and returns its value.
func (r *rendering) evalSelector(node *ast.Selector) interface{} {
	v := asBase(r.evalExpression(node.Expr))
	// map
	if v2, ok := v.(map[string]interface{}); ok {
		if v3, ok := v2[node.Ident]; ok {
			return v3
		}
		panic(r.errorf(node, "field %q does not exist", node.Ident))
	}
	rv := reflect.ValueOf(v)
	kind := rv.Kind()
	switch kind {
	case reflect.Map:
		if rv.Type().Key().Kind() == reflect.String {
			v := rv.MapIndex(reflect.ValueOf(node.Ident))
			if !v.IsValid() {
				panic(r.errorf(node, "field %q does not exist", node.Ident))
			}
			return v.Interface()
		}
		panic(r.errorf(node, "unsupported vars type"))
	case reflect.Struct:
		st := reflect.Indirect(rv)
		fields := getStructFields(st)
		index, ok := fields.indexOf[node.Ident]
		if !ok {
			panic(r.errorf(node, "field %q does not exist", node.Ident))
		}
		return st.Field(index).Interface()
	case reflect.Ptr:
		elem := rv.Type().Elem()
		if elem.Kind() == reflect.Struct {
			if rv.IsNil() {
				return rv.Interface()
			}
			st := reflect.Indirect(rv)
			fields := getStructFields(st)
			index, ok := fields.indexOf[node.Ident]
			if !ok {
				panic(r.errorf(node, "field %q does not exist", node.Ident))
			}
			return st.Field(index).Interface()
		}
	}
	panic(r.errorf(node, "type %s cannot have fields", typeof(v)))
}

// evalIndex evaluates an index expression and returns its value.
func (r *rendering) evalIndex(node *ast.Index) interface{} {
	var i int
	switch index := asBase(r.evalExpression(node.Index)).(type) {
	case int:
		i = index
	case decimal.Decimal:
		var err error
		i, err = r.decimalToInt(node, index)
		if err != nil {
			panic(err)
		}
		if i < 0 {
			panic(r.errorf(node, "invalid slice index %s (index must be non-negative)", node.Index))
		}
	default:
		panic(r.errorf(node, "non-integer slice index %s", node.Index))
	}
	if i < 0 {
		panic(r.errorf(node, "invalid slice index %d (index must be non-negative)", i))
	}
	var v = asBase(r.evalExpression(node.Expr))
	if v == nil {
		if r.isBuiltin("nil", node.Expr) {
			panic(r.errorf(node.Expr, "use of untyped nil"))
		} else {
			panic(r.errorf(node, "index out of range"))
		}
	}
	var e = reflect.ValueOf(v)
	if e.Kind() == reflect.Slice {
		if i >= e.Len() {
			panic(r.errorf(node, "index out of range"))
		}
		return e.Index(i).Interface()
	}
	if e.Kind() == reflect.String {
		var p = 0
		for _, c := range e.Interface().(string) {
			if p == i {
				return string(c)
			}
			p++
		}
		panic(r.errorf(node, "index out of range"))
	}
	panic(r.errorf(node, "invalid operation: %s (type %s does not support indexing)", node, typeof(v)))
}

// evalSlice evaluates a slice expression and returns its value.
func (r *rendering) evalSlice(node *ast.Slice) interface{} {
	var ok bool
	var l, h int
	if node.Low != nil {
		n := r.evalExpression(node.Low)
		l, ok = n.(int)
		if !ok {
			panic(r.errorf(node.Low, "invalid slice index %s (type %s)", node.Low, typeof(n)))
		}
		if l < 0 {
			panic(r.errorf(node.Low, "invalid slice index %s (index must be non-negative)", node.Low))
		}
	}
	if node.High != nil {
		n := r.evalExpression(node.High)
		h, ok = n.(int)
		if !ok {
			panic(r.errorf(node.High, "invalid slice index %s (type %s)", node.High, typeof(n)))
		}
		if h < 0 {
			panic(r.errorf(node.High, "invalid slice index %s (index must be non-negative)", node.High))
		}
		if l > h {
			panic(r.errorf(node.Low, "invalid slice index: %d > %d", l, h))
		}
	}
	var v = asBase(r.evalExpression(node.Expr))
	if v == nil {
		if r.isBuiltin("nil", node.Expr) {
			panic(r.errorf(node.Expr, "use of untyped nil"))
		} else {
			panic(r.errorf(node, "slice bounds out of range"))
		}
	}
	var e = reflect.ValueOf(v)
	if e.Kind() == reflect.Slice {
		if node.High == nil {
			h = e.Len()
		} else if h > e.Len() {
			panic(r.errorf(node.High, "slice bounds out of range"))
		}
		return e.Slice(l, h).Interface()
	}
	if e.Kind() == reflect.String {
		str := e.String()
		i := 0
		lb, hb := -1, -1
		for ib := range str {
			if i == l {
				lb = ib
				if node.High == nil {
					hb = len(str)
					break
				}
			}
			if i >= l && i == h {
				hb = ib
				break
			}
			i++
		}
		if lb == -1 {
			panic(r.errorf(node.Low, "slice bounds out of range"))
		}
		if hb == -1 {
			if i < h {
				panic(r.errorf(node.High, "slice bounds out of range"))
			}
			hb = len(str)
		}
		if lb == 0 && hb == len(str) {
			return str
		}
		return str[lb:hb]
	}
	panic(r.errorf(node, "cannot slice %s (type %s)", node.Expr, typeof(v)))
}

// evalIdentifier evaluates an identifier expression and returns its value.
func (r *rendering) evalIdentifier(node *ast.Identifier) interface{} {
	for i := len(r.vars) - 1; i >= 0; i-- {
		if r.vars[i] != nil {
			if i == 0 && node.Name == "len" {
				panic(r.errorf(node, "use of builtin len not in function call"))
			}
			if v, ok := r.vars[i][node.Name]; ok {
				return v
			}
		}
	}
	panic(r.errorf(node, "undefined: %s", node.Name))
}

// evalCall evaluates a call expression and returns its value.
func (r *rendering) evalCall(node *ast.Call) interface{} {

	if r.isBuiltin("len", node.Func) {
		if len(node.Args) == 0 {
			panic(r.errorf(node, "missing argument to len: len()"))
		}
		if len(node.Args) > 1 {
			panic(r.errorf(node, "too many arguments to len: %s", node))
		}
		if r.isBuiltin("nil", node.Args[0]) {
			panic(r.errorf(node, "use of untyped nil"))
		}
		arg := asBase(r.evalExpression(node.Args[0]))
		length := _len(arg)
		if length == -1 {
			panic(r.errorf(node.Args[0], "invalid argument %s (type %s) for len", node.Args[0], typeof(arg)))
		}
		return length
	}

	var f = r.evalExpression(node.Func)

	var fun = reflect.ValueOf(f)
	if !fun.IsValid() {
		panic(r.errorf(node, "cannot call non-function %s (type %s)", node.Func, typeof(f)))
	}
	var typ = fun.Type()
	if typ.Kind() != reflect.Func {
		panic(r.errorf(node, "cannot call non-function %s (type %s)", node.Func, typeof(f)))
	}

	var numIn = typ.NumIn()
	var isVariadic = typ.IsVariadic()

	if (!isVariadic && len(node.Args) != numIn) || (isVariadic && len(node.Args) < numIn-1) {
		have := "("
		for i, arg := range node.Args {
			if i > 0 {
				have += ", "
			}
			have += typeof(r.evalExpression(arg))
		}
		have += ")"
		want := "("
		for i := 0; i < numIn; i++ {
			if i > 0 {
				want += ", "
			}
			if i == numIn-1 && isVariadic {
				want += "..."
			}
			if in := typ.In(i); in.Kind() == reflect.Interface {
				want += "any"
			} else {
				want += typeof(reflect.Zero(in).Interface())
			}
		}
		want += ")"
		if len(node.Args) < numIn {
			panic(r.errorf(node, "not enough arguments in call to %s\n\thave %s\n\twant %s", node.Func, have, want))
		} else {
			panic(r.errorf(node, "too many arguments in call to %s\n\thave %s\n\twant %s", node.Func, have, want))
		}
	}

	var args = make([]reflect.Value, len(node.Args))

	var lastIn = numIn - 1
	var in reflect.Type

	for i := 0; i < len(node.Args); i++ {
		if i < lastIn || !isVariadic {
			in = typ.In(i)
		} else if i == lastIn {
			in = typ.In(lastIn).Elem()
		}

		inKind := in.Kind()
		var arg interface{}
		if i < len(node.Args) {
			arg = asBase(r.evalExpression(node.Args[i]))
		}
		if arg == nil {
			if i < len(node.Args) {
				if in == decimalType {
					panic(r.errorf(node, "cannot use nil as type number in argument to %s", node.Func))
				}
				switch inKind {
				case reflect.Bool, reflect.Int, reflect.String:
					wantType := typeof(reflect.Zero(in).Interface())
					panic(r.errorf(node, "cannot use nil as type %s in argument to %s", wantType, node.Func))
				}
			}
			args[i] = reflect.Zero(in)
		} else {
			if inKind == reflect.Interface {
				args[i] = reflect.ValueOf(arg)
			} else if d, ok := arg.(decimal.Decimal); ok && in == decimalType {
				args[i] = reflect.ValueOf(d)
			} else if d, ok := arg.(decimal.Decimal); ok && inKind == reflect.Int {
				n, err := r.decimalToInt(node.Args[i], d)
				if err != nil {
					panic(err)
				}
				args[i] = reflect.ValueOf(n)
			} else if d, ok := arg.(int); ok && in == decimalType {
				args[i] = reflect.ValueOf(decimal.New(int64(d), 0))
			} else if html, ok := arg.(HTML); ok && inKind == reflect.String {
				args[i] = reflect.ValueOf(string(html))
			} else if reflect.TypeOf(arg).AssignableTo(in) {
				args[i] = reflect.ValueOf(arg)
			} else {
				switch inKind {
				case reflect.Int8,
					reflect.Int16,
					reflect.Int32,
					reflect.Int64,
					reflect.Uint,
					reflect.Uint8,
					reflect.Uint16,
					reflect.Uint32,
					reflect.Uint64,
					reflect.Uintptr,
					reflect.Complex64,
					reflect.Complex128,
					reflect.Ptr,
					reflect.UnsafePointer,
					reflect.Float32,
					reflect.Float64:
					panic(fmt.Errorf("cannot use %s as function parameter type", inKind))
				}
				expectedType := typeof(reflect.Zero(in).Interface())
				panic(r.errorf(node, "cannot use %s (type %s) as type %s in argument to %s", arg, typeof(arg), expectedType, node.Func))
			}
		}
	}

	values := fun.Call(args)

	if len(values) == 2 {
		if !values[1].IsNil() {
			err, ok := values[1].Interface().(error)
			if !ok {
				panic("no-error return value")
			}
			panic(r.errorf(node.Func, "%s", err))
		}
	} else if len(values) != 1 {
		panic("wrong number of return values")
	}

	v := values[0].Interface()

	if d, ok := v.(int); ok {
		v = decimal.New(int64(d), 0)
	}

	return v
}

// isBuiltin indicates if expr is the builtin with the given name.
func (r *rendering) isBuiltin(name string, expr ast.Expression) bool {
	if n, ok := expr.(*ast.Identifier); ok {
		if n.Name != name {
			return false
		}
		for i := len(r.vars) - 1; i >= 0; i-- {
			if r.vars[i] != nil {
				if _, ok := r.vars[i][name]; ok {
					return i == 0
				}
			}
		}
	}
	return false
}

// htmlToStringType returns e1 and e2 with type string instead of HTML.
// If they do not have HTML type they are returned unchanged.
func htmlToStringType(e1, e2 interface{}) (interface{}, interface{}) {
	if e, ok := e1.(HTML); ok {
		e1 = string(e)
	}
	if e, ok := e2.(HTML); ok {
		e2 = string(e)
	}
	return e1, e2
}

// asBase returns the base value of v.
func asBase(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	// number
	case int:
		return vv
	case uint:
		return decimal.NewFromBigInt(new(big.Int).SetUint64(uint64(vv)), 0)
	case int8:
		return int(vv)
	case int16:
		return int(vv)
	case int32:
		return int(vv)
	case int64:
		if strconv.IntSize == 32 {
			return decimal.New(vv, 0)
		}
		return int(vv)
	case uint8:
		return int(vv)
	case uint16:
		return int(vv)
	case uint32:
		if strconv.IntSize == 32 {
			return decimal.New(int64(vv), 0)
		}
		return int(vv)
	case uint64:
		return decimal.NewFromBigInt(new(big.Int).SetUint64(vv), 0)
	case float32:
		return decimal.NewFromFloat32(vv)
	case float64:
		return decimal.NewFromFloat(vv)
	case decimal.Decimal:
		return v
	case Numberer:
		return vv.Number()
	case complex64, complex128, uintptr:
		panic(fmt.Errorf("cannot use %T as implementation type", vv))
	// string
	case string:
		return v
	case HTML:
		return v
	case Stringer:
		return vv.String()
	// bool
	case bool:
		return v
	default:
		rv := reflect.ValueOf(v)
		rt := rv.Type()
		if rt.ConvertibleTo(stringType) {
			return rv.String()
		} else if rt.ConvertibleTo(intType) {
			return rv.Int()
		} else if rt.ConvertibleTo(float64Type) {
			return decimal.NewFromFloat(rv.Float())
		} else if rt.ConvertibleTo(boolType) {
			return rv.Bool()
		}
	}
	return v
}

// structFields represents the fields of a struct.
type structFields struct {
	names   []string
	indexOf map[string]int
}

// structs maintains the association between the field names of a struct,
// as they are called in the template, and the field index in the struct.
var structs = struct {
	fields map[reflect.Type]structFields
	sync.RWMutex
}{map[reflect.Type]structFields{}, sync.RWMutex{}}

// getStructFields returns the fields of the struct st.
func getStructFields(st reflect.Value) structFields {
	typ := st.Type()
	structs.RLock()
	fields, ok := structs.fields[typ]
	structs.RUnlock()
	if !ok {
		structs.Lock()
		if fields, ok = structs.fields[typ]; !ok {
			fields = structFields{
				names:   []string{},
				indexOf: map[string]int{},
			}
			n := typ.NumField()
			for i := 0; i < n; i++ {
				fieldType := typ.Field(i)
				if fieldType.PkgPath != "" {
					continue
				}
				name := fieldType.Name
				if tag, ok := fieldType.Tag.Lookup("template"); ok {
					name = parseVarTag(tag)
					if name == "" {
						structs.Unlock()
						panic(fmt.Errorf("template: invalid tag of field %q", fieldType.Name))
					}
				}
				fields.names = append(fields.names, name)
				fields.indexOf[name] = i
			}
			structs.fields[typ] = fields
		}
		structs.Unlock()
	}
	return fields
}

// parseVarTag parses the tag of a struct field and returns its name.
func parseVarTag(tag string) string {
	sp := strings.SplitN(tag, ",", 2)
	if len(sp) == 0 {
		return ""
	}
	name := sp[0]
	if name == "" {
		return ""
	}
	for _, r := range name {
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return ""
		}
	}
	return name
}
