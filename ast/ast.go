// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ast declares the types used to define the template trees.
package ast

import (
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"
)

type OperatorType int

const (
	OperatorEqual          OperatorType = iota // ==
	OperatorNotEqual                           // !=
	OperatorNot                                // !
	OperatorLess                               // <
	OperatorLessOrEqual                        // <=
	OperatorGreater                            // >
	OperatorGreaterOrEqual                     // >=
	OperatorAnd                                // &&
	OperatorOr                                 // ||
	OperatorAddition                           // +
	OperatorSubtraction                        // -
	OperatorMultiplication                     // *
	OperatorDivision                           // /
	OperatorModulo                             // %
)

func (op OperatorType) String() string {
	return []string{"==", "!=", "!", "<", "<=", ">", ">=", "&&", "||", "+", "-", "*", "/", "%"}[op]
}

// Context indicates the context in which a Show node is located.
type Context int

const (
	ContextText Context = iota
	ContextHTML
	ContextTag
	ContextAttribute
	ContextUnquotedAttribute
	ContextCSS
	ContextCSSString
	ContextScript
	ContextScriptString
)

func (ctx Context) String() string {
	switch ctx {
	case ContextText:
		return "Text"
	case ContextHTML:
		return "HTML"
	case ContextTag:
		return "Tag"
	case ContextAttribute:
		return "Attribute"
	case ContextUnquotedAttribute:
		return "UnquotedAttribute"
	case ContextCSS:
		return "CSS"
	case ContextCSSString:
		return "CSSString"
	case ContextScript:
		return "Script"
	case ContextScriptString:
		return "ScriptString"
	}
	panic("invalid context")
}

// Node is an element of the tree.
type Node interface {
	Pos() *Position // node position in the original source
}

// Position is a position of a node in the source.
type Position struct {
	Line   int // line starting from 1
	Column int // column in characters starting from 1
	Start  int // index of the first byte
	End    int // index of the last byte
}

func (p *Position) Pos() *Position {
	return p
}

func (p Position) String() string {
	return strconv.Itoa(p.Line) + ":" + strconv.Itoa(p.Column)
}

type Expression interface {
	isexpr()
	Node
	String() string
}

type expression struct{}

func (e expression) isexpr() {}

// Tree represents a parsed tree.
type Tree struct {
	*Position
	Path    string  // path of the tree.
	Nodes   []Node  // nodes of the first level of the tree.
	Context Context // context.
}

func NewTree(path string, nodes []Node, ctx Context) *Tree {
	if nodes == nil {
		nodes = []Node{}
	}
	tree := &Tree{
		Position: &Position{1, 1, 0, 0},
		Path:     path,
		Nodes:    nodes,
		Context:  ctx,
	}
	return tree
}

type TextCut struct {
	Left  int
	Right int
}

// Text represents a text.
type Text struct {
	*Position         // position in the source.
	Text      []byte  // text.
	Cut       TextCut // cut.
}

func NewText(pos *Position, text []byte) *Text {
	return &Text{pos, text, TextCut{0, len(text)}}
}

func (t Text) String() string {
	return string(t.Text)
}

// URL represents an URL.
type URL struct {
	*Position        // position in the source.
	Tag       string // tag (in lowercase).
	Attribute string // attribute (in lowercase).
	Value     []Node // value nodes.
}

func NewURL(pos *Position, tag, attribute string, value []Node) *URL {
	return &URL{pos, tag, attribute, value}
}

// Var represents a statement {% var identifier = expression %}.
type Var struct {
	*Position             // position in the source.
	Ident     *Identifier // identifier.
	Expr      Expression  // assigned expression..
}

func NewVar(pos *Position, ident *Identifier, expr Expression) *Var {
	return &Var{pos, ident, expr}
}

func (v Var) String() string {
	return fmt.Sprintf("{%% var %v = %v %%}", v.Ident, v.Expr)
}

// Assignment represents a statement {% identifier = expression %}.
type Assignment struct {
	*Position             // position in the source.
	Ident     *Identifier // identifier.
	Expr      Expression  // assigned expression.
}

func NewAssignment(pos *Position, ident *Identifier, expr Expression) *Assignment {
	return &Assignment{pos, ident, expr}
}

func (a Assignment) String() string {
	return fmt.Sprintf("{%% %v = %v %%}", a.Ident, a.Expr)
}

// For represents a statement {% for ... %}.
type For struct {
	*Position             // position in the source.
	Index     *Identifier // index.
	Ident     *Identifier // identifier.
	Expr1     Expression  // left expression of the range or slice on which to iterate.
	Expr2     Expression  // right expression of the range.
	Nodes     []Node      // nodes of the body.
}

func NewFor(pos *Position, index, ident *Identifier, expr1, expr2 Expression, nodes []Node) *For {
	if nodes == nil {
		nodes = []Node{}
	}
	return &For{pos, index, ident, expr1, expr2, nodes}
}

// Break represents a statement {% break %}.
type Break struct {
	*Position // position in the source.
}

func NewBreak(pos *Position) *Break {
	return &Break{pos}
}

// Continue represents a statement {% continue %}.
type Continue struct {
	*Position // position in the source.
}

func NewContinue(pos *Position) *Continue {
	return &Continue{pos}
}

// If represents a statement {% if ... %}.
type If struct {
	*Position            // position in the source.
	Expr      Expression // expression that once evaluated returns true or false.
	Then      []Node     // nodes to run if the expression is evaluated to true.
	Else      []Node     // nodes to run if the expression is evaluated to false.
}

func NewIf(pos *Position, expr Expression, then []Node, els []Node) *If {
	if then == nil {
		then = []Node{}
	}
	return &If{pos, expr, then, els}
}

// Macro represents a statement {% macro ... %}.
type Macro struct {
	*Position                // position in the source.
	Ident      *Identifier   // name.
	Parameters []*Identifier // parameters.
	Body       []Node        // body.
	IsVariadic bool          // indicates if it is variadic.
	Context    Context       // context.
}

func NewMacro(pos *Position, ident *Identifier, parameters []*Identifier, body []Node, isVariadic bool, ctx Context) *Macro {
	if body == nil {
		body = []Node{}
	}
	return &Macro{pos, ident, parameters, body, isVariadic, ctx}
}

// ShowMacro represents a statement {% show <macro> %}.
type ShowMacro struct {
	*Position              // position in the source.
	Import    *Identifier  // name of the import.
	Macro     *Identifier  // name of the macro.
	Arguments []Expression // arguments.
	Context   Context      // context.
}

func NewShowMacro(pos *Position, impor, macro *Identifier, arguments []Expression, ctx Context) *ShowMacro {
	return &ShowMacro{Position: pos, Import: impor, Macro: macro, Arguments: arguments, Context: ctx}
}

// ShowPath represents a statement {% show <path> %}.
type ShowPath struct {
	*Position         // position in the source.
	Path      string  // path of the source to show.
	Context   Context // context.
	Tree      *Tree   // extended tree of <path>.
}

func NewShowPath(pos *Position, path string, ctx Context) *ShowPath {
	return &ShowPath{Position: pos, Path: path, Context: ctx}
}

// Value represents a statement {{...}}.
type Value struct {
	*Position            // position in the source.
	Expr      Expression // expression that once evaluated returns the value to show.
	Context   Context    // context.
}

func NewValue(pos *Position, expr Expression, ctx Context) *Value {
	return &Value{pos, expr, ctx}
}

func (v Value) String() string {
	return fmt.Sprintf("{{ %v }}", v.Expr)
}

// Extend represents a statement {% extend ... %}.
type Extend struct {
	*Position         // position in the source.
	Path      string  // path to the file to extend.
	Context   Context // context.
	Tree      *Tree   // extended tree of extend.
}

func NewExtend(pos *Position, path string, ctx Context) *Extend {
	return &Extend{Position: pos, Path: path, Context: ctx}
}

func (e Extend) String() string {
	return fmt.Sprintf("{%% extend %v %%}", strconv.Quote(e.Path))
}

// Import represents a statement {% import ... %}.
type Import struct {
	*Position             // position in the source.
	Ident     *Identifier // identifier.
	Path      string      // path of the imported file.
	Context   Context     // context.
	Tree      *Tree       // extended tree of import.
}

func NewImport(pos *Position, ident *Identifier, path string, ctx Context) *Import {
	return &Import{Position: pos, Ident: ident, Path: path, Context: ctx}
}

func (i Import) String() string {
	if i.Ident == nil {
		return fmt.Sprintf("{%% import %v %%}", strconv.Quote(i.Path))
	}

	return fmt.Sprintf("{%% import %v %v %%}", i.Ident, strconv.Quote(i.Path))
}

type Comment struct {
	*Position        // position in the source.
	Text      string // comment text.
}

func NewComment(pos *Position, text string) *Comment {
	return &Comment{pos, text}
}

type Parentesis struct {
	expression
	*Position            // position in the source.
	Expr      Expression // expression.
}

func NewParentesis(pos *Position, expr Expression) *Parentesis {
	return &Parentesis{expression{}, pos, expr}
}

func (n *Parentesis) String() string {
	return "(" + n.Expr.String() + ")"
}

type Int struct {
	expression
	*Position     // position in the source.
	Value     int // value.
}

func NewInt(pos *Position, value int) *Int {
	return &Int{expression{}, pos, value}
}

func (n *Int) String() string {
	return strconv.Itoa(n.Value)
}

type Number struct {
	expression
	*Position                 // position in the source.
	Value     decimal.Decimal // value.
}

func NewNumber(pos *Position, value decimal.Decimal) *Number {
	return &Number{expression{}, pos, value}
}

func (n *Number) String() string {
	return n.Value.String()
}

type String struct {
	expression
	*Position        // position in the source.
	Text      string // expression.
}

func NewString(pos *Position, text string) *String {
	return &String{expression{}, pos, text}
}

func (n *String) String() string {
	return strconv.Quote(n.Text)
}

type Identifier struct {
	expression
	*Position        // position in the source.
	Name      string // name.
}

func NewIdentifier(pos *Position, name string) *Identifier {
	return &Identifier{expression{}, pos, name}
}

func (n *Identifier) String() string {
	return n.Name
}

type Operator interface {
	Expression
	Operator() OperatorType
	Precedence() int
}

type UnaryOperator struct {
	expression
	*Position              // position in the source.
	Op        OperatorType // operator.
	Expr      Expression   // expression.
}

func NewUnaryOperator(pos *Position, op OperatorType, expr Expression) *UnaryOperator {
	return &UnaryOperator{expression{}, pos, op, expr}
}

func (n *UnaryOperator) String() string {
	s := n.Op.String()
	if e, ok := n.Expr.(Operator); ok && e.Precedence() <= n.Precedence() {
		s += "(" + n.Expr.String() + ")"
	} else {
		s += n.Expr.String()
	}
	return s
}

func (n *UnaryOperator) Operator() OperatorType {
	return n.Op
}

func (n *UnaryOperator) Precedence() int {
	return 6
}

type BinaryOperator struct {
	expression
	*Position              // position in the source.
	Op        OperatorType // operator.
	Expr1     Expression   // first expression.
	Expr2     Expression   // second expression.
}

func NewBinaryOperator(pos *Position, op OperatorType, expr1, expr2 Expression) *BinaryOperator {
	return &BinaryOperator{expression{}, pos, op, expr1, expr2}
}

func (n *BinaryOperator) String() string {
	var s string
	if e, ok := n.Expr1.(Operator); ok && e.Precedence() <= n.Precedence() {
		s += "(" + n.Expr1.String() + ")"
	} else {
		s += n.Expr1.String()
	}
	s += " " + n.Op.String() + " "
	if e, ok := n.Expr2.(Operator); ok && e.Precedence() <= n.Precedence() {
		s += "(" + n.Expr2.String() + ")"
	} else {
		s += n.Expr2.String()
	}
	return s
}

func (n *BinaryOperator) Operator() OperatorType {
	return n.Op
}

func (n *BinaryOperator) Precedence() int {
	switch n.Op {
	case OperatorMultiplication, OperatorDivision, OperatorModulo:
		return 5
	case OperatorAddition, OperatorSubtraction:
		return 4
	case OperatorEqual, OperatorNotEqual, OperatorLess, OperatorLessOrEqual,
		OperatorGreater, OperatorGreaterOrEqual:
		return 3
	case OperatorAnd:
		return 2
	case OperatorOr:
		return 1
	}
	panic("invalid operator type")
}

type Call struct {
	expression
	*Position              // position in the source.
	Func      Expression   // function.
	Args      []Expression // arguments.
}

func NewCall(pos *Position, fun Expression, args []Expression) *Call {
	return &Call{expression{}, pos, fun, args}
}

func (n *Call) String() string {
	s := n.Func.String() + "("
	for i, arg := range n.Args {
		s += arg.String()
		if i < len(n.Args)-1 {
			s += ", "
		}
	}
	s += ")"
	return s
}

type Index struct {
	expression
	*Position            // position in the source.
	Expr      Expression // expression.
	Index     Expression // index.
}

func NewIndex(pos *Position, expr Expression, index Expression) *Index {
	return &Index{expression{}, pos, expr, index}
}

func (n *Index) String() string {
	return n.Expr.String() + "[" + n.Index.String() + "]"
}

type Slice struct {
	expression
	*Position            // position in the source.
	Expr      Expression // expression.
	Low       Expression // low bound.
	High      Expression // high bound.
}

func NewSlice(pos *Position, expr, low, high Expression) *Slice {
	return &Slice{expression{}, pos, expr, low, high}
}

func (n *Slice) String() string {
	s := n.Expr.String() + "["
	if n.Low != nil {
		s += n.Low.String()
	}
	s += ":"
	if n.High != nil {
		s += n.High.String()
	}
	s += "]"
	return s
}

type Selector struct {
	expression
	*Position            // position in the source.
	Expr      Expression // expression.
	Ident     string     // identifier.
}

func NewSelector(pos *Position, expr Expression, ident string) *Selector {
	return &Selector{expression{}, pos, expr, ident}
}

func (n *Selector) String() string {
	return n.Expr.String() + "." + n.Ident
}
