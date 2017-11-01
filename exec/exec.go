//
// Copyright (c) 2016-2017 Open2b Software Snc. All Rights Reserved.
//

// Package exec fornisce i metodi per eseguire gli alberi dei template.
package exec

import (
	"errors"
	"fmt"
	"html"
	"io"
	"reflect"
	"strconv"

	"open2b/template/ast"
	"open2b/template/types"
)

// HTML viene usato per le stringhe che contengono codice HTML
// affinché show non le sottoponga ad escape.
// Nelle espressioni si comporta come una stringa.
type HTML string

// Stringer è implementato da qualsiasi valore che si comporta come una stringa.
type Stringer interface {
	String() string
}

// Numberer è implementato da qualsiasi valore che si comporta come un numero.
type Numberer interface {
	Number() types.Number
}

var errValue = errors.New("error value")

type Error struct {
	Path string
	Pos  ast.Position
	Err  error
}

func (e *Error) Error() string {
	return fmt.Sprintf("template: %s at %q %s", e.Err, e.Path, e.Pos)
}

var zero = types.NewNumberInt(0)
var numberType = reflect.TypeOf(zero)

// errBreak è ritornato dall'esecuzione dello statement "break".
// Viene gestito dallo statement "for" più interno.
var errBreak = errors.New("break is not in a loop")

// errContinue è ritornato dall'esecuzione dello statement "break".
// Viene gestito dallo statement "for" più interno.
var errContinue = errors.New("continue is not in a loop")

var stringType = reflect.TypeOf("")
var intType = reflect.TypeOf(0)

type Env struct {
	tree  *ast.Tree
	catch func(e *Error) error
}

// NewEnv ritorna un ambiente di esecuzione per l'albero tree.
// La funzione catch gestisce il comportamento in caso di errore.
//
// Quando si verifica un errore di esecuzione, viene chiamata
// la funzione catch con argomento l'errore. Se catch ritorna
// un errore diverso da nil allora l'esecuzione si interrompe e
// l'errore ritornato viene a sua volta ritornato dal metodo Execute.
//
// Se catch è nil allora in caso di errore l'esecuzione si interrompe
// ed il metodo Execute ritorna l'errore.
func NewEnv(tree *ast.Tree, catch func(err *Error) error) *Env {
	if tree == nil {
		panic("template: tree is nil")
	}
	return &Env{tree, catch}
}

// Execute esegue l'albero tree e scrive il risultato su wr.
// Le variabili in vars sono definite nell'ambiente durante l'esecuzione.
func (env *Env) Execute(wr io.Writer, vars map[string]interface{}) error {
	if wr == nil {
		return errors.New("template: wr is nil")
	}
	s := state{
		vars:     []map[string]interface{}{builtin, vars, {}},
		catch:    env.catch,
		treepath: env.tree.Path,
	}
	var err error
	extend := getExtendNode(env.tree)
	if extend == nil {
		s.path = env.tree.Path
		err = s.execute(wr, env.tree.Nodes, nil)
	} else {
		if extend.Tree == nil {
			return errors.New("template: extend node is not expanded")
		}
		// legge le region
		regions := map[string]*ast.Region{}
		for _, node := range env.tree.Nodes {
			if r, ok := node.(*ast.Region); ok {
				regions[r.Name] = r
			}
		}
		s.path = extend.Path
		err = s.execute(wr, extend.Tree.Nodes, regions)
	}
	if err != nil {
		if env.catch != nil {
			if err2, ok := err.(*Error); ok {
				err = env.catch(err2)
			}
		}
		return err
	}
	return nil
}

// state rappresenta lo stato di esecuzione di un albero.
type state struct {
	path     string
	vars     []map[string]interface{}
	catch    func(e *Error) error
	treepath string
}

// errorf costruisce e ritorna un errore di esecuzione.
func (s *state) errorf(node ast.Node, format string, args ...interface{}) error {
	var pos = node.Pos()
	var err = &Error{
		Path: s.path,
		Pos: ast.Position{
			Line:   pos.Line,
			Column: pos.Column,
			Start:  pos.Start,
			End:    pos.End,
		},
		Err: fmt.Errorf(format, args...),
	}
	if s.catch != nil {
		err2 := s.catch(err)
		if err2 == nil {
			err2 = errValue
		}
		return err2
	}
	return err
}

// execute esegue i nodi nodes.
func (s *state) execute(wr io.Writer, nodes []ast.Node, regions map[string]*ast.Region) error {

	var err error

	for _, n := range nodes {

		switch node := n.(type) {

		case *ast.Text:

			_, err = io.WriteString(wr, node.Text[node.Cut.Left:node.Cut.Right])
			if err != nil {
				return err
			}

		case *ast.Show:

			var expr interface{}
			expr, err = s.eval(node.Expr)
			if err != nil {
				return err
			}

			var str string
			switch node.Context {
			case ast.ContextHTML:
				str = interfaceToHTML(expr)
			case ast.ContextScript:
				str = interfaceToScript(expr)
			}

			_, err := io.WriteString(wr, str)
			if err != nil {
				return err
			}

		case *ast.If:

			if len(node.Then) == 0 && len(node.Else) == 0 {
				continue
			}
			var expr interface{}
			expr, err = s.eval(node.Expr)
			if err != nil {
				return err
			}
			if c, ok := expr.(bool); ok {
				if c {
					if len(node.Then) > 0 {
						err = s.execute(wr, node.Then, nil)
					}
				} else {
					if len(node.Else) > 0 {
						err = s.execute(wr, node.Else, nil)
					}
				}
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("non-bool %s (type %T) used as if condition", node.Expr, node.Expr)
			}

		case *ast.For:

			if len(node.Nodes) == 0 {
				continue
			}
			index := ""
			if node.Index != nil {
				index = node.Index.Name
			}
			ident := node.Ident.Name

			var expr interface{}
			expr, err = s.eval(node.Expr)
			if err != nil {
				return err
			}

			av := reflect.ValueOf(expr)
			if !av.IsValid() {
				continue
			}

			var list []interface{}
			if av.Kind() == reflect.Slice {
				if av.Len() == 0 {
					continue
				}
				list = make([]interface{}, av.Len())
				for i := 0; i < len(list); i++ {
					list[i] = av.Index(i).Interface()
				}
			} else {
				list = []interface{}{av.Interface()}
			}

			s.vars = append(s.vars, nil)
			for i, v := range list {
				vars := map[string]interface{}{ident: v}
				if index != "" {
					vars[index] = i
				}
				s.vars[len(s.vars)-1] = vars
				err = s.execute(wr, node.Nodes, nil)
				if err != nil {
					if err == errBreak {
						break
					}
					if err == errContinue {
						continue
					}
					return err
				}
			}
			s.vars = s.vars[:len(s.vars)-1]

		case *ast.Break:
			return errBreak

		case *ast.Continue:
			return errContinue

		case *ast.Var:

			var vars = s.vars[len(s.vars)-1]
			var name = node.Ident.Name
			if _, ok := vars[name]; ok {
				return fmt.Errorf("variable %q already declared in this block: %#v", name, vars)
			}
			vars[name], err = s.eval(node.Expr)
			if err != nil {
				return err
			}

		case *ast.Assignment:

			var name = node.Ident.Name
			for i := len(s.vars) - 1; i >= 0; i-- {
				var vars = s.vars[i]
				if _, ok := vars[name]; ok {
					if i < 2 {
						if i == 0 && name == "len" {
							return fmt.Errorf("use of builtin len not in function call")
						}
						return fmt.Errorf("cannot assign to %s", name)
					}
					vars[name], err = s.eval(node.Expr)
					if err != nil {
						return err
					}
					break
				}
			}
			return fmt.Errorf("variable %s not declared", name)

		case *ast.Region:

			if regions != nil {
				region := regions[node.Name]
				if region != nil {
					path := s.path
					s.path = s.treepath
					err = s.execute(wr, region.Nodes, nil)
					if err != nil {
						return err
					}
					s.path = path
				}
			}

		case *ast.Include:

			if node.Tree == nil {
				return errors.New("include node is not expanded")
			}
			path := s.path
			s.path = node.Path
			err = s.execute(wr, node.Tree.Nodes, nil)
			if err != nil {
				return err
			}
			s.path = path

		}
	}

	return nil
}

// eval valuta una espressione ritornandone il valore.
//
// Se si verifica un errore, chiama s.catch con parametro l'errore.
// Se questa ritorna nil allora ritorna l'errore come valore della
// valutazione, altrimenti ritorna nil e l'errore ritornato da s.catch.
func (s *state) eval(exp ast.Expression) (value interface{}, err error) {
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		if e, ok := r.(error); ok {
	// 			if e == errValue {
	// 				value = errValue
	// 			} else {
	// 				err = e
	// 			}
	// 		} else {
	// 			panic(r)
	// 		}
	// 	}
	// }()
	return s.evalExpression(exp), nil
}

// evalExpression valuta una espressione ritornandone il valore.
// In caso di errore chiama panic con l'errore come parametro.
func (s *state) evalExpression(expr ast.Expression) interface{} {
	switch e := expr.(type) {
	case *ast.String:
		return e.Text
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
		panic(s.errorf(expr, "unexprected node type %#v", expr))
	}
}

// evalUnaryOperator valuta un operatore unario e ne ritorna il valore.
// In caso di errore chiama panic con l'errore come parametro.
func (s *state) evalUnaryOperator(node *ast.UnaryOperator) interface{} {
	var e = asBasic(s.evalExpression(node.Expr))
	switch node.Op {
	case ast.OperatorNot:
		if b, ok := e.(bool); ok {
			return !b
		}
		panic(s.errorf(node, "invalid operation: ! %T", e))
	case ast.OperatorAddition:
		if _, ok := e.(types.Number); ok {
			return e
		}
		panic(s.errorf(node, "invalid operation: + %T", e))
	case ast.OperatorSubtraction:
		if n, ok := e.(types.Number); ok {
			return n.Opposite()
		}
		panic(s.errorf(node, "invalid operation: - %T", e))
	}
	panic("Unknown Unary Operator")
}

// evalBinaryOperator valuta un operatore binario e ne ritorna il valore.
// In caso di errore chiama panic con l'errore come parametro.
func (s *state) evalBinaryOperator(node *ast.BinaryOperator) interface{} {

	var expr1 = asBasic(s.evalExpression(node.Expr1))
	var expr2 = asBasic(s.evalExpression(node.Expr2))

	switch node.Op {

	case ast.OperatorEqual:
		if expr1 == nil || expr2 == nil {
			defer func() {
				if recover() != nil {
					panic(s.errorf(node, "invalid operation: %T == %T", expr1, expr2))
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
			case types.Number:
				if e2, ok := expr2.(types.Number); ok {
					return e1.Compared(e2) == 0
				}
			}
		}
		panic(s.errorf(node, "invalid operation: %T == %T", expr1, expr2))

	case ast.OperatorNotEqual:
		if expr1 == nil || expr2 == nil {
			defer func() {
				if recover() != nil {
					panic(s.errorf(node, "invalid operation: %T != %T", expr1, expr2))
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
			case types.Number:
				if e2, ok := expr2.(types.Number); ok {
					return e1.Compared(e2) != 0
				}
			}
		}
		panic(s.errorf(node, "invalid operation: %T != %T", expr1, expr2))

	case ast.OperatorLess:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T < %T", expr1, expr2))
		}
		expr1, expr2 = htmlToStringType(expr1, expr2)
		switch e1 := expr1.(type) {
		case string:
			if e2, ok := expr2.(string); ok {
				return e1 < e2
			}
		case types.Number:
			if e2, ok := expr2.(types.Number); ok {
				return e1.Compared(e2) < 0
			}
		}
		panic(fmt.Sprintf("invalid operation: %T < %T", expr1, expr2))

	case ast.OperatorLessOrEqual:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T <= %T", expr1, expr2))
		}
		expr1, expr2 = htmlToStringType(expr1, expr2)
		switch e1 := expr1.(type) {
		case string:
			if e2, ok := expr2.(string); ok {
				return e1 <= e2
			}
		case types.Number:
			if e2, ok := expr2.(types.Number); ok {
				return e1.Compared(e2) <= 0
			}
		}
		panic(s.errorf(node, "invalid operation: %T <= %T", expr1, expr2))

	case ast.OperatorGreater:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T > %T", expr1, expr2))
		}
		expr1, expr2 = htmlToStringType(expr1, expr2)
		switch e1 := expr1.(type) {
		case string:
			if e2, ok := expr2.(string); ok {
				return e1 > e2
			}
		case types.Number:
			if e2, ok := expr2.(types.Number); ok {
				return e1.Compared(e2) > 0
			}
		}
		panic(s.errorf(node, "invalid operation: %T > %T", expr1, expr2))

	case ast.OperatorGreaterOrEqual:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T >= %T", expr1, expr2))
		}
		expr1, expr2 = htmlToStringType(expr1, expr2)
		switch e1 := expr1.(type) {
		case string:
			if e2, ok := expr2.(string); ok {
				return e1 >= e2
			}
		case types.Number:
			if e2, ok := expr2.(types.Number); ok {
				return e1.Compared(e2) >= 0
			}
		}
		panic(s.errorf(node, "invalid operation: %T >= %T", expr1, expr2))

	case ast.OperatorAnd:
		if e1, ok := expr1.(bool); ok {
			if e2, ok := expr2.(bool); ok {
				return e1 && e2
			}
		}
		panic(s.errorf(node, "invalid operation: %T && %T", expr1, expr2))

	case ast.OperatorOr:
		if e1, ok := expr1.(bool); ok {
			if e2, ok := expr2.(bool); ok {
				return e1 || e2
			}
		}
		panic(s.errorf(node, "invalid operation: %T || %T", expr1, expr2))

	case ast.OperatorAddition:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T + %T", expr1, expr2))
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
		case types.Number:
			if e2, ok := expr2.(types.Number); ok {
				return e1.Plus(e2)
			}
		}
		panic(s.errorf(node, "invalid operation: %T + %T", expr1, expr2))

	case ast.OperatorSubtraction:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T - %T", expr1, expr2))
		}
		if e1, ok := expr1.(types.Number); ok {
			if e2, ok := expr2.(types.Number); ok {
				return e1.Minus(e2)
			}
		}
		panic(s.errorf(node, "invalid operation: %T - %T", expr1, expr2))

	case ast.OperatorMultiplication:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T * %T", expr1, expr2))
		}
		if e1, ok := expr1.(types.Number); ok {
			if e2, ok := expr2.(types.Number); ok {
				return e1.Multiplied(e2)
			}
		}
		panic(s.errorf(node, "invalid operation: %T * %T", expr1, expr2))

	case ast.OperatorDivision:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T / %T", expr1, expr2))
		}
		if e1, ok := expr1.(types.Number); ok {
			if e2, ok := expr2.(types.Number); ok {
				return e1.Divided(e2)
			}
		}
		panic(s.errorf(node, "invalid operation: %T / %T", expr1, expr2))

	case ast.OperatorModulo:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T %% %T", expr1, expr2))
		}
		if e1, ok := expr1.(types.Number); ok {
			if e2, ok := expr2.(types.Number); ok {
				return e1.Module(e2)
			}
		}
		panic(s.errorf(node, "invalid operation: %T %% %T", expr1, expr2))

	}

	panic("unknown binary operator")
}

func (s *state) evalSelector(node *ast.Selector) interface{} {
	v := asBasic(s.evalExpression(node.Expr))
	// map
	if v2, ok := v.(map[string]interface{}); ok {
		if v3, ok := v2[node.Ident]; ok && v3 != nil {
			return v3
		}
		panic(s.errorf(node, "field %q does not exist", node.Ident))
	}
	rv := reflect.ValueOf(v)
	kind := rv.Kind()
	switch kind {
	case reflect.Struct:
		v2 := rv.FieldByName(node.Ident)
		if v2.IsValid() {
			return v2.Interface()
		}
		panic(s.errorf(node, "field %q does not exist", node.Ident))
	case reflect.Ptr:
		elem := rv.Type().Elem()
		if elem.Kind() == reflect.Struct {
			if rv.IsNil() {
				return rv.Interface()
			} else {
				v2 := reflect.Indirect(rv).FieldByName(node.Ident)
				if v2.IsValid() {
					return v2.Interface()
				}
				panic(s.errorf(node, "field %q does not exist", node.Ident))
			}
		}
	}
	panic(s.errorf(node, "type %T cannot have fields", v))
}

func (s *state) evalIndex(node *ast.Index) interface{} {
	n, ok := asBasic(s.evalExpression(node.Index)).(types.Number)
	if !ok {
		panic(s.errorf(node, "non-integer slice index %s", node.Index))
	}
	index, ok := n.Int()
	if !ok {
		panic(s.errorf(node, "non-integer slice index %s", n))
	}
	var e = reflect.ValueOf(s.evalExpression(node.Expr))
	if e.Kind() == reflect.Slice {
		if index < 0 || e.Len() <= index {
			return nil
		}
		return e.Index(index).Interface()
	}
	if e.Kind() == reflect.String {
		var s = e.Interface().(string)
		var p = 0
		for _, c := range s {
			if p == index {
				return string(c)
			}
		}
		return nil
	}
	return nil
}

func (s *state) evalSlice(node *ast.Slice) interface{} {
	var l, h int
	if node.Low != nil {
		l = toInt(s.evalExpression(node.Low))
		if l < 0 {
			l = 0
		}
	}
	if node.High != nil {
		h = toInt(s.evalExpression(node.High))
		if h < 0 {
			h = 0
		}
	}
	var e = reflect.ValueOf(s.evalExpression(node.Expr))
	switch e.Kind() {
	case reflect.Slice:
		if node.High == nil || h > e.Len() {
			h = e.Len()
		}
		if h <= l {
			return nil
		}
		return e.Slice(l, h).Interface()
	case reflect.String:
		s := e.String()
		if node.High == nil || h > len(s) {
			h = len(s)
		}
		if h <= l {
			return ""
		}
		if l == 0 && h == len(s) {
			return s
		}
		i := 0
		lb, hb := 0, len(s)
		for ib := range s {
			if i == l {
				lb = ib
				if h == len(s) {
					break
				}
			}
			if i == h {
				hb = ib
				break
			}
			i++
		}
		if lb == 0 && hb == len(s) {
			return s
		}
		return s[lb:hb]
	}
	return nil
}

func (s *state) evalIdentifier(node *ast.Identifier) interface{} {
	for i := len(s.vars) - 1; i >= 0; i-- {
		if v, ok := s.vars[i][node.Name]; ok && (i == 0 || v != nil) {
			if v == errValue {
				panic(v)
			}
			if n, ok := v.(int); ok {
				v = types.NewNumberInt(n)
			}
			return v
		}
	}
	panic(s.errorf(node, "undefined: %s", node.Name))
}

func (s *state) evalCall(node *ast.Call) interface{} {

	var f = s.evalExpression(node.Func)
	var fun = reflect.ValueOf(f)
	if !fun.IsValid() {
		panic(s.errorf(node, "cannot call non-function %s (type %T)", node.Func, f))
	}
	var typ = fun.Type()
	if typ.Kind() != reflect.Func {
		panic(s.errorf(node, "cannot call non-function %s (type %T)", node.Func, f))
	}

	var numIn = typ.NumIn()
	var args = make([]reflect.Value, numIn)

	for i := 0; i < numIn; i++ {
		var ok bool
		var in = typ.In(i)
		if in == numberType {
			var a = zero
			if i < len(node.Args) {
				arg := asBasic(s.evalExpression(node.Args[i]))
				if arg == nil {
					panic(s.errorf(node, "cannot use nil as type number in argument to function %s", node.Func))
				}
				a, ok = arg.(types.Number)
				if !ok {
					panic(s.errorf(node, "cannot use %#v (type %T) as type number in argument to function %s", arg, arg, node.Func))
				}
			}
			args[i] = reflect.ValueOf(a)
			continue
		}
		switch in.Kind() {
		case reflect.Bool:
			var a = false
			if i < len(node.Args) {
				arg := s.evalExpression(node.Args[i])
				a, ok = arg.(bool)
				if !ok {
					if arg == nil {
						panic(s.errorf(node, "cannot use nil as type bool in argument to function %s", node.Func))
					} else {
						panic(s.errorf(node, "cannot use %#v (type %T) as type bool in argument to function %s", arg, arg, node.Func))
					}
				}
			}
			args[i] = reflect.ValueOf(a)
		case reflect.Int:
			var a = 0
			if i < len(node.Args) {
				arg := asBasic(s.evalExpression(node.Args[i]))
				if arg == nil {
					panic(s.errorf(node, "cannot use nil as type number in argument to function %s", node.Func))
				}
				var n types.Number
				n, ok = arg.(types.Number)
				if !ok {
					panic(s.errorf(node, "cannot use %#v (type %T) as type number in argument to function %s", arg, arg, node.Func))
				}
				a, ok = n.Int()
				if !ok {
					panic(s.errorf(node, "cannot use %s as integer in argument to function %s", n, node.Func))
				}
			}
			args[i] = reflect.ValueOf(a)
		case reflect.String:
			var a = ""
			if i < len(node.Args) {
				arg := asBasic(s.evalExpression(node.Args[i]))
				if arg == nil {
					panic(s.errorf(node, "cannot use nil as type string in argument to function %s", node.Func))
				}
				a, ok = arg.(string)
				if !ok {
					rv := reflect.ValueOf(arg)
					if rv.Type().ConvertibleTo(stringType) {
						a = rv.String()
					} else {
						panic(s.errorf(node, "cannot use %#v (type %T) as type string in argument to function %s", arg, arg, node.Func))
					}
				}
			}
			args[i] = reflect.ValueOf(a)
		case reflect.Interface:
			var arg interface{}
			if i < len(node.Args) {
				arg = s.evalExpression(node.Args[i])
			}
			if arg == nil {
				args[i] = reflect.Zero(in)
			} else {
				args[i] = reflect.ValueOf(arg)
			}
		case reflect.Slice:
			var arg interface{}
			if i < len(node.Args) {
				arg = s.evalExpression(node.Args[i])
			}
			if in == reflect.TypeOf(arg) {
				args[i] = reflect.ValueOf(arg)
			} else {
				args[i] = reflect.Zero(in)
			}
		case reflect.Func:
			var arg interface{}
			if i < len(node.Args) {
				arg = s.evalExpression(node.Args[i])
			}
			if arg == nil {
				args[i] = reflect.Zero(in)
			} else {
				args[i] = reflect.ValueOf(arg)
			}
		default:
			panic(s.errorf(node, "unsupported call argument type %s in function %s", in.Kind(), node.Func))
		}
	}

	var vals = fun.Call(args)
	v := vals[0].Interface()

	if n, ok := v.(int); ok {
		v = types.NewNumberInt(n)
	}

	return v
}

// htmlToStringType ritorna e1 e e2 con tipo string al posto di HTML.
// Se non hanno tipo HTML vengono ritornate invariate.
func htmlToStringType(e1, e2 interface{}) (interface{}, interface{}) {
	if e, ok := e1.(HTML); ok {
		e1 = string(e)
	}
	if e, ok := e2.(HTML); ok {
		e2 = string(e)
	}
	return e1, e2
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case types.Number:
		i, err := strconv.Atoi(n.Rounded(0, "Down").String())
		if err != nil {
			return 0
		}
		return i
	case string:
		i, err := strconv.Atoi(n)
		if err != nil {
			return 0
		}
		return i
	}
	return 0
}

// getExtendNode ritorna il nodo Extend di un albero.
// Se il nodo non è presente ritorna nil.
func getExtendNode(tree *ast.Tree) *ast.Extend {
	if len(tree.Nodes) == 0 {
		return nil
	}
	if node, ok := tree.Nodes[0].(*ast.Extend); ok {
		return node
	}
	if len(tree.Nodes) > 1 {
		if node, ok := tree.Nodes[1].(*ast.Extend); ok {
			return node
		}
	}
	return nil
}

func asBasic(v interface{}) interface{} {
	if v == nil {
		return v
	}
	switch vv := v.(type) {
	case string:
		return v
	case HTML:
		return v
	case types.Number:
		return v
	case bool:
		return v
	case Stringer:
		return vv.String()
	case Numberer:
		return vv.Number()
	default:
		rv := reflect.ValueOf(v)
		rt := rv.Type()
		if rt.ConvertibleTo(stringType) {
			return rv.String()
		} else if rt.ConvertibleTo(intType) {
			return rv.Int()
		}
	}
	return v
}
