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
	"strings"
	"unicode"
	"unicode/utf8"

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

// errBreak returned from executing the "break" statement.
// It is managed by the innermost "for" statement.
var errBreak = errors.New("break is not in a loop")

// errContinue is returned from the execution of the "break" statement.
// It is managed by the innermost "for" statement.
var errContinue = errors.New("continue is not in a loop")

// scope di variabili
type scope map[string]interface{}

var scopeType = reflect.TypeOf(scope{})

// Execute runs the tree tree and writes the result to wr.
// The variables in vars are defined in the environment during execution.
//
// vars can be:
//
//   - a map with a key of type string
//   - a type with underlying type one of the previous map types
//   - a struct or pointer to struct
//   - a reflect.Value whose concrete value meets one of the previous ones
//   - nil
//
func Execute(wr io.Writer, tree *ast.Tree, version string, vars interface{}, h func(error) bool) error {

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

	if h == nil {
		h = func(err error) bool { return false }
	}

	s := &state{
		scope:       map[string]scope{},
		path:        tree.Path,
		vars:        []scope{builtins, globals, {}},
		version:     version,
		handleError: h,
	}

	extend := getExtendNode(tree)
	if extend == nil {
		err = s.execute(wr, tree.Nodes)
	} else {
		if extend.Tree == nil {
			return errors.New("template/exec: extend node is not expanded")
		}
		s.scope[s.path] = s.vars[2]
		err = s.execute(nil, tree.Nodes)
		if err != nil {
			return err
		}
		s.path = extend.Tree.Path
		vars := scope{}
		for name, v := range s.vars[2] {
			if r, ok := v.(region); ok {
				fc, _ := utf8.DecodeRuneInString(name)
				if unicode.Is(unicode.Lu, fc) && !strings.Contains(name, ".") {
					vars[name] = r
				}
			}
		}
		s.vars = []scope{builtins, globals, vars}
		err = s.execute(wr, extend.Tree.Nodes)
	}

	return err
}

// varsToScope converts variables into a scope.
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
							if ptr := s.rv.Field(i); !ptr.IsNil() {
								value := reflect.Indirect(ptr)
								structs = append(structs, st{elem, value})
							}
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

// region represents a region in a scope.
type region struct {
	path string
	node *ast.Region
}

// state represents the state of execution of a tree.
type state struct {
	scope       map[string]scope
	path        string
	vars        []scope
	version     string
	handleError func(error) bool
}

// errorf builds and returns an execution error.
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

// execute executes nodes.
func (s *state) execute(wr io.Writer, nodes []ast.Node) error {

	var err error

Nodes:
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
				if s.handleError(err) {
					continue
				}
				return err
			}

			err = s.writeTo(wr, expr, node)
			if err != nil {
				return err
			}

		case *ast.If:

			var expr interface{}
			expr, err = s.eval(node.Expr)
			if err != nil {
				if !s.handleError(err) {
					return err
				}
				expr = false
			}
			c, ok := expr.(bool)
			if !ok {
				err = s.errorf(node, "non-bool %s (type %s) used as if condition", node.Expr, typeof(expr))
				if !s.handleError(err) {
					return err
				}
			}
			if c {
				if len(node.Then) > 0 {
					s.vars = append(s.vars, nil)
					err = s.execute(wr, node.Then)
					s.vars = s.vars[:len(s.vars)-1]
					if err != nil {
						return err
					}
				}
			} else if len(node.Else) > 0 {
				s.vars = append(s.vars, nil)
				err = s.execute(wr, node.Else)
				s.vars = s.vars[:len(s.vars)-1]
				if err != nil {
					return err
				}
			}

		case *ast.For:

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
				if s.handleError(err) {
					continue
				}
				return err
			}

			if len(node.Nodes) == 0 {
				continue
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
						err = s.errorf(node, "cannot range over %s (type %s)", node.Expr1, typeof(expr))
						if s.handleError(err) {
							continue
						}
						return err
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
					if s.handleError(err) {
						continue
					}
					return err
				}

				n1, ok := expr.(int)
				if !ok {
					err = s.errorf(node, "left range value is not a integer")
					if s.handleError(err) {
						continue
					}
					return err
				}
				n2, ok := expr2.(int)
				if !ok {
					err = s.errorf(node, "right range value is not a integer")
					if s.handleError(err) {
						continue
					}
					return err
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
				err = s.errorf(node.Ident, "regions not allowed")
				if !s.handleError(err) {
					return err
				}
			}
			name := node.Ident.Name
			if v, ok := s.variable(name); ok {
				if r, ok := v.(region); ok {
					err = s.errorf(node, "%s redeclared\n\tprevious declaration at %s:%s",
						name, r.path, r.node.Pos())
				} else {
					err = s.errorf(node.Ident, "%s redeclared in this file", name)
				}
				if s.handleError(err) {
					continue
				}
				return err
			}
			s.vars[2][name] = region{s.path, node}

		case *ast.Var:

			if s.vars[len(s.vars)-1] == nil {
				s.vars[len(s.vars)-1] = scope{}
			}
			var vars = s.vars[len(s.vars)-1]
			var name = node.Ident.Name
			if v, ok := vars[name]; ok {
				if r, ok := v.(region); ok {
					err = s.errorf(node, "%s redeclared\n\tprevious declaration at %s:%s",
						name, r.path, r.node.Pos())
				} else {
					err = s.errorf(node.Ident, "%s redeclared in this block", name)
				}
				if s.handleError(err) {
					continue
				}
				return err
			}
			v, err := s.eval(node.Expr)
			if err != nil {
				if s.handleError(err) {
					continue
				}
				return err
			}
			vars[name] = v

		case *ast.Assignment:

			var found bool
			var name = node.Ident.Name
			for i := len(s.vars) - 1; i >= 0; i-- {
				vars := s.vars[i]
				if vars != nil {
					if v, ok := vars[name]; ok {
						if _, ok := v.(region); ok || i < 2 {
							if i == 0 && name == "len" {
								err = s.errorf(node, "use of builtin len not in function call")
							} else {
								err = fmt.Errorf("cannot assign to %s", name)
							}
							if s.handleError(err) {
								continue Nodes
							}
							return err
						}
						v, err := s.eval(node.Expr)
						if err != nil {
							if s.handleError(err) {
								continue Nodes
							}
							return err
						}
						vars[name] = v
						found = true
						break
					}
				}
			}
			if !found {
				err = s.errorf(node, "variable %s not declared", name)
				if !s.handleError(err) {
					return err
				}
			}

		case *ast.ShowRegion:

			name := node.Region.Name
			if node.Import != nil {
				name = node.Import.Name + "." + name
			}
			var r region
			if v, ok := s.variable(name); ok {
				if r, ok = v.(region); !ok {
					err = s.errorf(node, "cannot show non-region %s (type %t)", name, typeof(v))
				}
			} else {
				err = s.errorf(node, "region %s not declared", name)
			}
			if err != nil {
				if s.handleError(err) {
					continue
				}
				return err
			}

			haveSize := len(node.Arguments)
			wantSize := len(r.node.Parameters)
			if haveSize != wantSize {
				have := "("
				for i := 0; i < haveSize; i++ {
					if i > 0 {
						have += ","
					}
					if i < wantSize {
						have += r.node.Parameters[i].Name
					} else {
						have += "?"
					}
				}
				have += ")"
				want := "("
				for i, p := range r.node.Parameters {
					if i > 0 {
						want += ","
					}
					want += p.Name
				}
				want += ")"
				name := node.Region.Name
				if node.Import != nil {
					name = node.Import.Name + " " + name
				}
				if haveSize < wantSize {
					err = s.errorf(node, "not enough arguments in show of %s\n\thave %s\n\twant %s", name, have, want)
				} else {
					err = s.errorf(node, "too many arguments in show of %s\n\thave %s\n\twant %s", name, have, want)
				}
				if s.handleError(err) {
					continue
				}
				return err
			}

			var arguments = scope{}
			for i, argument := range node.Arguments {
				arguments[r.node.Parameters[i].Name], err = s.eval(argument)
				if err != nil {
					if s.handleError(err) {
						continue Nodes
					}
					return err
				}
			}
			st := &state{
				scope:       s.scope,
				path:        r.path,
				vars:        []scope{s.vars[0], s.vars[1], s.scope[r.path], arguments},
				version:     s.version,
				handleError: s.handleError,
			}
			err = st.execute(wr, r.node.Body)
			if err != nil {
				return err
			}

		case *ast.Import:

			path := node.Tree.Path
			if _, ok := s.scope[path]; !ok {
				st := &state{
					scope:       s.scope,
					path:        path,
					vars:        []scope{s.vars[0], s.vars[1], {}},
					version:     s.version,
					handleError: s.handleError,
				}
				err = st.execute(nil, node.Tree.Nodes)
				if err != nil {
					return err
				}
				s.scope[path] = st.vars[2]
				for name, r := range st.vars[2] {
					if _, ok := r.(region); !ok {
						continue
					}
					if strings.Index(name, ".") > 0 {
						continue
					}
					if fc, _ := utf8.DecodeRuneInString(name); !unicode.Is(unicode.Lu, fc) {
						continue
					}
					s.vars[2][name] = r
				}
			}

		case *ast.ShowPath:

			s.vars = append(s.vars, nil)
			st := &state{
				scope:       s.scope,
				path:        node.Tree.Path,
				vars:        s.vars,
				version:     s.version,
				handleError: s.handleError,
			}
			err = st.execute(wr, node.Tree.Nodes)
			s.vars = s.vars[:len(s.vars)-1]
			if err != nil {
				return err
			}

		}
	}

	return nil
}

// getExtendNode returns the Extend node of a tree.
// If the node is not present, returns nil.
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
