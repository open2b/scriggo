// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package renderer

import (
	"fmt"
	"html"
	"reflect"

	"open2b/template/ast"

	"github.com/shopspring/decimal"
)

const maxInt = int64(^uint(0) >> 1)

// HTML is used for strings that contain HTML so that the show does
// not escape them. In expressions it behaves like a string.
type HTML string

// Stringer is implemented by any value that behaves like a string.
type Stringer interface {
	String() string
}

// Numberer is implemented by any value that behaves like a number.
type Numberer interface {
	Number() decimal.Decimal
}

var stringType = reflect.TypeOf("")
var intType = reflect.TypeOf(0)
var float64Type = reflect.TypeOf(0.0)
var boolType = reflect.TypeOf(false)

var zero = decimal.New(0, 0)
var decimalType = reflect.TypeOf(zero)

// eval evaluates an expression by returning its value.
func (s *state) eval(exp ast.Expression) (value interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				panic(r)
			}
		}
	}()
	return s.evalExpression(exp), nil
}

// evalExpression evaluates an expression and returns its value.
// In the event of an error, calls panic with the error as parameter.
func (s *state) evalExpression(expr ast.Expression) interface{} {
	switch e := expr.(type) {
	case *ast.String:
		return e.Text
	case *ast.Int:
		return e.Value
	case *ast.Number:
		return e.Value
	case *ast.Parentesis:
		return s.evalExpression(e.Expr)
	case *ast.UnaryOperator:
		return s.evalUnaryOperator(e)
	case *ast.BinaryOperator:
		return s.evalBinaryOperator(e)
	case *ast.Identifier:
		return s.evalIdentifier(e)
	case *ast.Call:
		return s.evalCall(e)
	case *ast.Index:
		return s.evalIndex(e)
	case *ast.Slice:
		return s.evalSlice(e)
	case *ast.Selector:
		return s.evalSelector(e)
	default:
		panic(s.errorf(expr, "unexpected node type %#v", expr))
	}
}

// evalUnaryOperator evaluates a unary operator and returns its value.
// On error it calls panic with the error as parameter.
func (s *state) evalUnaryOperator(node *ast.UnaryOperator) interface{} {
	var e = asBase(s.evalExpression(node.Expr))
	switch node.Op {
	case ast.OperatorNot:
		if b, ok := e.(bool); ok {
			return !b
		}
		panic(s.errorf(node, "invalid operation: ! %s", typeof(e)))
	case ast.OperatorAddition:
		if _, ok := e.(int); ok {
			return e
		}
		if _, ok := e.(decimal.Decimal); ok {
			return e
		}
		panic(s.errorf(node, "invalid operation: + %s", typeof(e)))
	case ast.OperatorSubtraction:
		if n, ok := e.(int); ok {
			return -n
		}
		if n, ok := e.(decimal.Decimal); ok {
			return n.Neg()
		}
		panic(s.errorf(node, "invalid operation: - %s", typeof(e)))
	}
	panic("Unknown Unary Operator")
}

// evalBinaryOperator evaluates a binary operator and returns its value.
// On error it calls panic with the error as parameter.
func (s *state) evalBinaryOperator(node *ast.BinaryOperator) interface{} {

	expr1 := asBase(s.evalExpression(node.Expr1))

	switch node.Op {

	case ast.OperatorEqual:
		expr2 := asBase(s.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			defer func() {
				if recover() != nil {
					panic(s.errorf(node, "invalid operation: %s == %s", typeof(expr1), typeof(expr2)))
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
		panic(s.errorf(node, "invalid operation: %s == %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorNotEqual:
		expr2 := asBase(s.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			defer func() {
				if recover() != nil {
					panic(s.errorf(node, "invalid operation: %s != %s", typeof(expr1), typeof(expr2)))
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
		panic(s.errorf(node, "invalid operation: %s != %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorLess:
		expr2 := asBase(s.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %s < %s", typeof(expr1), typeof(expr2)))
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
		expr2 := asBase(s.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %s <= %s", typeof(expr1), typeof(expr2)))
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
		panic(s.errorf(node, "invalid operation: %s <= %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorGreater:
		expr2 := asBase(s.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %s > %s", typeof(expr1), typeof(expr2)))
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
		panic(s.errorf(node, "invalid operation: %s > %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorGreaterOrEqual:
		expr2 := asBase(s.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %s >= %s", typeof(expr1), typeof(expr2)))
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
		panic(s.errorf(node, "invalid operation: %s >= %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorAnd:
		if e1, ok := expr1.(bool); ok {
			if !e1 {
				return false
			}
			expr2 := asBase(s.evalExpression(node.Expr2))
			if e2, ok := expr2.(bool); ok {
				return e1 && e2
			}
			panic(s.errorf(node, "invalid operation: %s && %s", typeof(expr1), typeof(expr2)))
		}
		panic(s.errorf(node, "invalid operation: %s && ...", typeof(expr1)))

	case ast.OperatorOr:
		if e1, ok := expr1.(bool); ok {
			if e1 {
				return true
			}
			expr2 := asBase(s.evalExpression(node.Expr2))
			if e2, ok := expr2.(bool); ok {
				return e1 || e2
			}
			panic(s.errorf(node, "invalid operation: %s || %s", typeof(expr1), typeof(expr2)))
		}
		panic(s.errorf(node, "invalid operation: %s || ...", typeof(expr1)))

	case ast.OperatorAddition:
		expr2 := asBase(s.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %s + %s", typeof(expr1), typeof(expr2)))
		}
		switch e1 := expr1.(type) {
		case string:
			switch e2 := expr2.(type) {
			case string:
				return e1 + e2
			case HTML:
				return HTML(html.EscapeString(e1) + string(e2))
			}
		case HTML:
			switch e2 := expr2.(type) {
			case string:
				return HTML(string(e1) + html.EscapeString(e2))
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
		panic(s.errorf(node, "invalid operation: %s + %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorSubtraction:
		expr2 := asBase(s.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %s - %s", typeof(expr1), typeof(expr2)))
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
		panic(s.errorf(node, "invalid operation: %s - %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorMultiplication:
		expr2 := asBase(s.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %s * %s", typeof(expr1), typeof(expr2)))
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
		panic(s.errorf(node, "invalid operation: %s * %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorDivision:
		expr2 := asBase(s.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %s / %s", typeof(expr1), typeof(expr2)))
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
		panic(s.errorf(node, "invalid operation: %s / %s", typeof(expr1), typeof(expr2)))

	case ast.OperatorModulo:
		expr2 := asBase(s.evalExpression(node.Expr2))
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %s %% %s", typeof(expr1), typeof(expr2)))
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
		panic(s.errorf(node, "invalid operation: %s %% %s", typeof(expr1), typeof(expr2)))

	}

	panic("unknown binary operator")
}

func (s *state) evalSelector(node *ast.Selector) interface{} {
	v := asBase(s.evalExpression(node.Expr))
	// map
	if v2, ok := v.(map[string]interface{}); ok {
		if v3, ok := v2[node.Ident]; ok {
			return v3
		}
		panic(s.errorf(node, "field %q does not exist", node.Ident))
	}
	rv := reflect.ValueOf(v)
	kind := rv.Kind()
	switch kind {
	case reflect.Map:
		if rv.Type().Key().Kind() == reflect.String {
			v := rv.MapIndex(reflect.ValueOf(node.Ident))
			if !v.IsValid() {
				panic(s.errorf(node, "field %q does not exist", node.Ident))
			}
			return v.Interface()
		}
		panic(s.errorf(node, "unsupported vars type"))
	case reflect.Struct:
		value, err := getStructField(rv, node.Ident, s.version)
		if err != nil {
			if err == errFieldNotExist {
				err = s.errorf(node, "field %q does not exist", node.Ident)
			}
			panic(err)
		}
		return value
	case reflect.Ptr:
		elem := rv.Type().Elem()
		if elem.Kind() == reflect.Struct {
			if rv.IsNil() {
				return rv.Interface()
			}
			value, err := getStructField(reflect.Indirect(rv), node.Ident, s.version)
			if err != nil {
				if err == errFieldNotExist {
					err = s.errorf(node, "field %q does not exist", node.Ident)
				}
				panic(err)
			}
			return value
		}
	}
	panic(s.errorf(node, "type %s cannot have fields", typeof(v)))
}

func (s *state) evalIndex(node *ast.Index) interface{} {
	var i int
	switch index := asBase(s.evalExpression(node.Index)).(type) {
	case int:
		i = index
	case decimal.Decimal:
		p := index.IntPart()
		if p > maxInt || !decimal.New(p, 0).Equal(index) {
			panic(s.errorf(node, "non-integer slice index %s", node.Index))
		}
		if p < 0 {
			panic(s.errorf(node, "invalid slice index %s (index must be non-negative)", node.Index))
		}
		i = int(p)
	default:
		panic(s.errorf(node, "non-integer slice index %s", node.Index))
	}
	if i < 0 {
		panic(s.errorf(node, "invalid slice index %d (index must be non-negative)", i))
	}
	var v = asBase(s.evalExpression(node.Expr))
	if v == nil {
		if s.isBuiltin("nil", node.Expr) {
			panic(s.errorf(node.Expr, "use of untyped nil"))
		} else {
			panic(s.errorf(node, "index out of range"))
		}
	}
	var e = reflect.ValueOf(v)
	if e.Kind() == reflect.Slice {
		if i >= e.Len() {
			panic(s.errorf(node, "index out of range"))
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
		panic(s.errorf(node, "index out of range"))
	}
	panic(s.errorf(node, "invalid operation: %s (type %s does not support indexing)", node, typeof(v)))
}

func (s *state) evalSlice(node *ast.Slice) interface{} {
	var ok bool
	var l, h int
	if node.Low != nil {
		n := s.evalExpression(node.Low)
		l, ok = n.(int)
		if !ok {
			panic(s.errorf(node.Low, "invalid slice index %s (type %s)", node.Low, typeof(n)))
		}
		if l < 0 {
			panic(s.errorf(node.Low, "invalid slice index %s (index must be non-negative)", node.Low))
		}
	}
	if node.High != nil {
		n := s.evalExpression(node.High)
		h, ok = n.(int)
		if !ok {
			panic(s.errorf(node.High, "invalid slice index %s (type %s)", node.High, typeof(n)))
		}
		if h < 0 {
			panic(s.errorf(node.High, "invalid slice index %s (index must be non-negative)", node.High))
		}
		if l > h {
			panic(s.errorf(node.Low, "invalid slice index: %d > %d", l, h))
		}
	}
	var v = asBase(s.evalExpression(node.Expr))
	if v == nil {
		if s.isBuiltin("nil", node.Expr) {
			panic(s.errorf(node.Expr, "use of untyped nil"))
		} else {
			panic(s.errorf(node, "slice bounds out of range"))
		}
	}
	var e = reflect.ValueOf(v)
	if e.Kind() == reflect.Slice {
		if node.High == nil {
			h = e.Len()
		} else if h > e.Len() {
			panic(s.errorf(node.High, "slice bounds out of range"))
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
			panic(s.errorf(node.Low, "slice bounds out of range"))
		}
		if hb == -1 {
			if i < h {
				panic(s.errorf(node.High, "slice bounds out of range"))
			}
			hb = len(str)
		}
		if lb == 0 && hb == len(str) {
			return str
		}
		return str[lb:hb]
	}
	panic(s.errorf(node, "cannot slice %s (type %s)", node.Expr, typeof(v)))
}

func (s *state) evalIdentifier(node *ast.Identifier) interface{} {
	for i := len(s.vars) - 1; i >= 0; i-- {
		if s.vars[i] != nil {
			if i == 0 && node.Name == "len" {
				panic(s.errorf(node, "use of builtin len not in function call"))
			}
			if v, ok := s.vars[i][node.Name]; ok {
				return v
			}
		}
	}
	panic(s.errorf(node, "undefined: %s", node.Name))
}

func (s *state) evalCall(node *ast.Call) interface{} {

	if s.isBuiltin("len", node.Func) {
		if len(node.Args) == 0 {
			panic(s.errorf(node, "missing argument to len: len()"))
		}
		if len(node.Args) > 1 {
			panic(s.errorf(node, "too many arguments to len: %s", node))
		}
		if s.isBuiltin("nil", node.Args[0]) {
			panic(s.errorf(node, "use of untyped nil"))
		}
		arg := asBase(s.evalExpression(node.Args[0]))
		length := _len(arg)
		if length == -1 {
			panic(s.errorf(node.Args[0], "invalid argument %s (type %s) for len", node.Args[0], typeof(arg)))
		}
		return length
	}

	var f = s.evalExpression(node.Func)

	var fun = reflect.ValueOf(f)
	if !fun.IsValid() {
		panic(s.errorf(node, "cannot call non-function %s (type %s)", node.Func, typeof(f)))
	}
	var typ = fun.Type()
	if typ.Kind() != reflect.Func {
		panic(s.errorf(node, "cannot call non-function %s (type %s)", node.Func, typeof(f)))
	}

	var numIn = typ.NumIn()
	var numArgs = numIn
	if typ.IsVariadic() && numIn < len(node.Args) {
		numArgs = len(node.Args)
	}
	var args = make([]reflect.Value, numArgs)

	for i := 0; i < len(args); i++ {
		var ok bool
		var in reflect.Type
		if typ.IsVariadic() && i >= numIn-1 {
			in = typ.In(numIn - 1).Elem()
		} else {
			in = typ.In(i)
		}
		var arg interface{}
		if i < len(node.Args) {
			arg = asBase(s.evalExpression(node.Args[i]))
		}
		if in == decimalType {
			var a = zero
			if i < len(node.Args) {
				if arg == nil {
					panic(s.errorf(node, "cannot use nil as type decimal in argument to function %s", node.Func))
				}
				switch ar := arg.(type) {
				case decimal.Decimal:
					a = ar
				case int:
					a = decimal.New(int64(ar), 0)
				default:
					panic(s.errorf(node, "cannot use %#v (type %s) as type decimal in argument to function %s", arg, typeof(arg), node.Func))
				}
			}
			args[i] = reflect.ValueOf(a)
			continue
		}
		switch in.Kind() {
		case reflect.Bool:
			var a = false
			if i < len(node.Args) {
				a, ok = arg.(bool)
				if !ok {
					if arg == nil {
						panic(s.errorf(node, "cannot use nil as type bool in argument to function %s", node.Func))
					} else {
						panic(s.errorf(node, "cannot use %#v (type %s) as type bool in argument to function %s", arg, typeof(arg), node.Func))
					}
				}
			}
			args[i] = reflect.ValueOf(a)
		case reflect.Int:
			var a = 0
			if i < len(node.Args) {
				if arg == nil {
					panic(s.errorf(node, "cannot use nil as type int in argument to function %s", node.Func))
				}
				a, ok = arg.(int)
				if !ok {
					panic(s.errorf(node, "cannot use %#v (type %s) as type int in argument to function %s", arg, typeof(arg), node.Func))
				}
			}
			args[i] = reflect.ValueOf(a)
		case reflect.String:
			var a = ""
			if i < len(node.Args) {
				if arg == nil {
					panic(s.errorf(node, "cannot use nil as type string in argument to function %s", node.Func))
				}
				switch v := arg.(type) {
				case string:
					a = v
				case HTML:
					a = string(v)
				default:
					panic(s.errorf(node, "cannot use %#v (type %s) as type string in argument to function %s", arg, typeof(arg), node.Func))
				}
			}
			args[i] = reflect.ValueOf(a)
		case reflect.Interface:
			if arg == nil {
				args[i] = reflect.Zero(in)
			} else {
				args[i] = reflect.ValueOf(arg)
			}
		case reflect.Slice:
			if in == reflect.TypeOf(arg) {
				args[i] = reflect.ValueOf(arg)
			} else {
				args[i] = reflect.Zero(in)
			}
		case reflect.Func:
			if arg == nil {
				args[i] = reflect.Zero(in)
			} else {
				args[i] = reflect.ValueOf(arg)
			}
		default:
			panic(s.errorf(node, "unsupported call argument type %s in function %s", in.Kind(), node.Func))
		}
	}

	vals := func() []reflect.Value {
		defer func() {
			if e := recover(); e != nil {
				panic(s.errorf(node.Func, "%s", e))
			}
		}()
		return fun.Call(args)
	}()

	v := vals[0].Interface()

	if d, ok := v.(int); ok {
		v = decimal.New(int64(d), 0)
	}

	return v
}

// isBuiltin indicates if expr is the builtin with the given name.
func (s *state) isBuiltin(name string, expr ast.Expression) bool {
	if n, ok := expr.(*ast.Identifier); ok {
		if n.Name != name {
			return false
		}
		for i := len(s.vars) - 1; i >= 0; i-- {
			if s.vars[i] != nil {
				if _, ok := s.vars[i][name]; ok {
					return i == 0
				}
			}
		}
	}
	return false
}

// variable returns the value of the variable name in state s.
func (s *state) variable(name string) (interface{}, bool) {
	for i := len(s.vars) - 1; i >= 0; i-- {
		if s.vars[i] != nil {
			if v, ok := s.vars[i][name]; ok {
				return v, true
			}
		}
	}
	return nil, false
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

func asBase(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	// number
	case int:
		return vv
	case float64:
		return decimal.NewFromFloat(vv)
	case decimal.Decimal:
		return v
	case Numberer:
		return vv.Number()
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

func typeof(v interface{}) string {
	if v == nil {
		return ""
	}
	v = asBase(v)
	switch v.(type) {
	case int, decimal.Decimal:
		return "number"
	case string, HTML:
		return "string"
	case bool:
		return "bool"
	default:
		rv := reflect.ValueOf(v)
		rt := rv.Type()
		switch rt.Kind() {
		case reflect.Slice:
			return "slice"
		case reflect.Map, reflect.Ptr:
			return "struct"
		}
	}
	return fmt.Sprintf("(%T)", v)
}
