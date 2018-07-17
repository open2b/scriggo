//
// Copyright (c) 2016-2018 Open2b Software Snc. All Rights Reserved.
//

// Package exec fornisce i metodi per eseguire gli alberi dei template.
package exec

import (
	"errors"
	"fmt"
	"io"
	"reflect"

	"open2b/template/ast"
)

type Error struct {
	Path string
	Pos  ast.Position
	Err  error
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s:%s: %s", e.Path, e.Pos, e.Err)
}

type WriterTo interface {
	WriteTo(w io.Writer, ctx ast.Context) (n int, err error)
}

// errBreak è ritornato dall'esecuzione dello statement "break".
// Viene gestito dallo statement "for" più interno.
var errBreak = errors.New("break is not in a loop")

// errContinue è ritornato dall'esecuzione dello statement "break".
// Viene gestito dallo statement "for" più interno.
var errContinue = errors.New("continue is not in a loop")

// scope di variabili
type scope map[string]interface{}

var scopeType = reflect.TypeOf(scope{})

// Execute esegue l'albero tree e scrive il risultato su wr.
// Le variabili in vars sono definite nell'ambiente durante l'esecuzione.
//
// vars può essere:
//
//   - un map con chiave di tipo string
//   - un tipo con underlying type uno dei tipi map precedenti
//   - una struct o puntatore a struct
//   - un reflect.Value il cui valore concreto soddisfa uno dei precedenti
//   - nil
//
func Execute(wr io.Writer, tree *ast.Tree, version string, vars interface{}) error {

	if wr == nil {
		return errors.New("template/exec: wr is nil")
	}
	if tree == nil {
		return errors.New("template/exec: tree is nil")
	}

	globals, err := varsToScope(vars, version)
	if err != nil {
		return err
	}

	s := state{
		scope:   map[string]scope{},
		path:    tree.Path,
		vars:    []scope{builtins, globals, {}},
		version: version,
	}

	extend := getExtendNode(tree)
	if extend == nil {
		err = s.execute(wr, tree.Nodes)
	} else {
		if extend.Ref.Tree == nil {
			return errors.New("template/exec: extend node is not expanded")
		}
		s.scope[s.path] = s.vars[2]
		err = s.execute(nil, tree.Nodes)
		if err != nil {
			return err
		}
		s.path = extend.Path
		s.vars = []scope{builtins, globals, {}}
		err = s.execute(wr, extend.Ref.Tree.Nodes)
	}

	return err
}

// varsToScope converte delle variabili in uno scope.
func varsToScope(vars interface{}, version string) (scope, error) {

	if vars == nil {
		return scope{}, nil
	}

	var rv reflect.Value
	if rv, ok := vars.(reflect.Value); ok {
		vars = rv.Interface()
	}

	if v, ok := vars.(map[string]interface{}); ok {
		return scope(v), nil
	}

	if !rv.IsValid() {
		rv = reflect.ValueOf(vars)
	}
	rt := rv.Type()

	switch rv.Kind() {
	case reflect.Map:
		if rt.ConvertibleTo(scopeType) {
			m := rv.Convert(scopeType).Interface()
			return m.(scope), nil
		}
		if rt.Key().Kind() == reflect.String {
			s := scope{}
			for _, kv := range rv.MapKeys() {
				s[kv.String()] = rv.MapIndex(kv).Interface()
			}
			return s, nil
		}
	case reflect.Struct:
		type st struct {
			rt reflect.Type
			rv reflect.Value
		}
		globals := scope{}
		structs := []st{{rt, rv}}
		var s st
		for len(structs) > 0 {
			s, structs = structs[0], structs[1:]
			nf := s.rv.NumField()
			for i := 0; i < nf; i++ {
				field := s.rt.Field(i)
				if field.PkgPath != "" {
					continue
				}
				if field.Anonymous {
					switch field.Type.Kind() {
					case reflect.Ptr:
						elem := field.Type.Elem()
						if elem.Kind() == reflect.Struct {
							value := reflect.Indirect(s.rv.Field(i))
							structs = append(structs, st{elem, value})
						}
					case reflect.Struct:
						structs = append(structs, st{field.Type, s.rv.Field(i)})
					}
					continue
				}
				value := s.rv.Field(i).Interface()
				var name string
				var ver string
				if tag, ok := field.Tag.Lookup("template"); ok {
					name, ver = parseVarTag(tag)
					if name == "" {
						return nil, fmt.Errorf("template/exec: invalid tag of field %q", field.Name)
					}
					if ver != "" && ver != version {
						continue
					}
				}
				if name == "" {
					name = field.Name
				}
				if _, ok := globals[name]; !ok {
					globals[name] = value
				}
			}
		}
		return globals, nil
	case reflect.Ptr:
		elem := rv.Type().Elem()
		if elem.Kind() == reflect.Struct {
			return varsToScope(reflect.Indirect(rv), version)
		}
	}

	return nil, errors.New("template/exec: unsupported vars type")
}

// state rappresenta lo stato di esecuzione di un albero.
type state struct {
	scope   map[string]scope
	path    string
	vars    []scope
	version string
}

// errorf costruisce e ritorna un errore di esecuzione.
func (s *state) errorf(node ast.Node, format string, args ...interface{}) error {
	var pos = node.Pos()
	if pos == nil {
		return fmt.Errorf(format, args...)
	}
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
	return err
}

// execute esegue i nodi nodes.
func (s *state) execute(wr io.Writer, nodes []ast.Node) error {

	var err error

	for _, n := range nodes {

		switch node := n.(type) {

		case *ast.Text:

			if wr != nil {
				_, err = io.WriteString(wr, node.Text[node.Cut.Left:node.Cut.Right])
				if err != nil {
					return err
				}
			}

		case *ast.Value:

			var expr interface{}
			expr, err = s.eval(node.Expr)
			if err != nil {
				return err
			}

			if e, ok := expr.(WriterTo); ok {

				defer func() {
					if e := recover(); e != nil {
						panic(s.errorf(node, "%s", e))
					}
				}()

				_, err = e.WriteTo(wr, node.Context)
				if err != nil {
					return err
				}

			} else {

				var str string
				switch node.Context {
				case ast.ContextText:
					str = interfaceToText(expr)
				case ast.ContextHTML:
					str = interfaceToHTML(expr)
				case ast.ContextCSS:
					str = interfaceToCSS(expr)
				case ast.ContextJavaScript:
					str = interfaceToJavaScript(expr)
				default:
					panic("template/exec: unknown context")
				}
				_, err = io.WriteString(wr, str)
				if err != nil {
					return err
				}

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
			c, ok := expr.(bool)
			if !ok {
				return s.errorf(node, "non-bool %s (type %T) used as if condition", node.Expr, expr)
			}
			s.vars = append(s.vars, nil)
			if c {
				if len(node.Then) > 0 {
					err = s.execute(wr, node.Then)
				}
			} else {
				if len(node.Else) > 0 {
					err = s.execute(wr, node.Else)
				}
			}
			s.vars = s.vars[:len(s.vars)-1]
			if err != nil {
				return err
			}

		case *ast.For:

			if len(node.Nodes) == 0 {
				continue
			}
			index := ""
			if node.Index != nil {
				index = node.Index.Name
			}
			ident := ""
			if node.Ident != nil {
				ident = node.Ident.Name
			}

			var expr interface{}
			expr, err = s.eval(node.Expr1)
			if err != nil {
				return err
			}

			if node.Expr2 == nil {
				// syntax: for ident in expr
				// syntax: for index, ident in expr

				setScope := func(i int, v interface{}) {
					vars := scope{ident: v}
					if index != "" {
						vars[index] = i
					}
					s.vars[len(s.vars)-1] = vars
				}

				switch vv := expr.(type) {
				case string:
					size := len(vv)
					if size == 0 {
						continue
					}
					s.vars = append(s.vars, nil)
					i := 0
					for _, v := range vv {
						setScope(i, string(v))
						err = s.execute(wr, node.Nodes)
						if err != nil {
							if err == errBreak {
								break
							}
							if err != errContinue {
								return err
							}
						}
						i++
					}
					s.vars = s.vars[:len(s.vars)-1]
				case []string:
					size := len(vv)
					if size == 0 {
						continue
					}
					s.vars = append(s.vars, nil)
					for i := 0; i < size; i++ {
						setScope(i, vv[i])
						err = s.execute(wr, node.Nodes)
						if err != nil {
							if err == errBreak {
								break
							}
							if err != errContinue {
								return err
							}
						}
					}
					s.vars = s.vars[:len(s.vars)-1]
				default:
					av := reflect.ValueOf(expr)
					if !av.IsValid() || av.Kind() != reflect.Slice {
						continue
					}
					size := av.Len()
					if size == 0 {
						continue
					}
					s.vars = append(s.vars, nil)
					for i := 0; i < size; i++ {
						setScope(i, av.Index(i).Interface())
						err = s.execute(wr, node.Nodes)
						if err != nil {
							if err == errBreak {
								break
							}
							if err != errContinue {
								return err
							}
						}
					}
					s.vars = s.vars[:len(s.vars)-1]
				}

			} else {
				// syntax: for index in expr..expr

				expr2, err := s.eval(node.Expr2)
				if err != nil {
					return err
				}

				n1, ok := expr.(int)
				if !ok {
					return s.errorf(node, "left range value is not a integer")
				}
				n2, ok := expr2.(int)
				if !ok {
					return s.errorf(node, "right range value is not a integer")
				}

				step := 1
				if n2 < n1 {
					step = -1
				}

				s.vars = append(s.vars, nil)
				for i := n1; ; i += step {
					s.vars[len(s.vars)-1] = scope{index: i}
					err = s.execute(wr, node.Nodes)
					if err != nil {
						if err == errBreak {
							break
						}
						if err == errContinue {
							continue
						}
						return err
					}
					if i == n2 {
						break
					}
				}
				s.vars = s.vars[:len(s.vars)-1]

			}

		case *ast.Break:
			return errBreak

		case *ast.Continue:
			return errContinue

		case *ast.Region:
			if wr != nil {
				return s.errorf(node.Ident, "regions not allowed", node.Pos())
			}

		case *ast.Var:

			if s.vars[len(s.vars)-1] == nil {
				s.vars[len(s.vars)-1] = scope{}
			}
			var vars = s.vars[len(s.vars)-1]
			var name = node.Ident.Name
			if _, ok := vars[name]; ok {
				return s.errorf(node.Ident, "var %s redeclared in this block", name)
			}
			vars[name], err = s.eval(node.Expr)
			if err != nil {
				return err
			}

		case *ast.Assignment:

			var found bool
			var name = node.Ident.Name
			for i := len(s.vars) - 1; i >= 0; i-- {
				vars := s.vars[i]
				if vars != nil {
					if _, ok := vars[name]; ok {
						if i < 2 {
							if i == 0 && name == "len" {
								return s.errorf(node, "use of builtin len not in function call")
							}
							return fmt.Errorf("cannot assign to %s", name)
						}
						vars[name], err = s.eval(node.Expr)
						if err != nil {
							return err
						}
						found = true
						break
					}
				}
			}
			if !found {
				return s.errorf(node, "variable %s not declared", name)
			}

		case *ast.ShowRegion:

			var region = node.Ref.Region
			var arguments = scope{}
			for i, argument := range node.Arguments {
				arguments[region.Parameters[i].Name], err = s.eval(argument)
				if err != nil {
					return err
				}
			}

			path := s.path
			if node.Ref.Extend != nil {
				// TODO
			} else if node.Ref.Import != nil {
				path = node.Ref.Import.Ref.Tree.Path
			}
			st := state{
				scope:   s.scope,
				path:    path,
				vars:    []scope{s.vars[0], s.vars[1], s.scope[path], arguments},
				version: s.version,
			}
			err = st.execute(wr, region.Body)
			if err != nil {
				return err
			}

		case *ast.Import:

			path := node.Ref.Tree.Path
			if _, ok := s.scope[path]; !ok {
				st := state{
					scope:   s.scope,
					path:    path,
					vars:    []scope{s.vars[0], s.vars[1], {}},
					version: s.version,
				}
				err = st.execute(nil, node.Ref.Tree.Nodes)
				if err != nil {
					return err
				}
				s.scope[path] = s.vars[2]
			}

		case *ast.ShowPath:

			st := state{
				scope:   s.scope,
				path:    node.Ref.Tree.Path,
				vars:    []scope{s.vars[0], s.vars[1], {}},
				version: s.version,
			}
			err = st.execute(wr, node.Ref.Tree.Nodes)
			if err != nil {
				return err
			}

		}
	}

	return nil
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
