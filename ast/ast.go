//
// Copyright (c) 2016-2018 Open2b Software Snc. All Rights Reserved.
//

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
	ContextCSS
	ContextJavaScript
)

func (ctx Context) String() string {
	switch ctx {
	case ContextText:
		return "Text"
	case ContextHTML:
		return "HTML"
	case ContextCSS:
		return "CSS"
	case ContextJavaScript:
		return "JavaScript"
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
	Node
	isexpr()
	String() string
}

type expression struct{}

func (e expression) isexpr() {}

// Tree represents a parsed tree.
type Tree struct {
	*Position
	Path       string // path of the tree.
	Nodes      []Node // nodes of the first level of the tree.
}

func NewTree(path string, nodes []Node) *Tree {
	if nodes == nil {
		nodes = []Node{}
	}
	tree := &Tree{
		Position: &Position{1, 1, 0, 0},
		Path:     path,
		Nodes:    nodes,
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
	Text      string  // testo.
	Cut       TextCut // taglio.
}

func NewText(pos *Position, text string) *Text {
	return &Text{pos, text, TextCut{0, len(text)}}
}

func (t Text) String() string {
	return t.Text
}

// Var represents a statement {% var identifier = expression %}.
type Var struct {
	*Position             // position in the source.
	Ident     *Identifier // identificatore.
	Expr      Expression  // espressione assegnata.
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
	Ident     *Identifier // identificatore.
	Expr      Expression  // espressione assegnata.
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
	Nodes     []Node      // nodes to be executed for each element of the list.
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

// Region represents a statement {% region ... %}.
type Region struct {
	*Position                // position in the source.
	Ident      *Identifier   // name.
	Parameters []*Identifier // parameters.
	Body       []Node        // body.
}

func NewRegion(pos *Position, ident *Identifier, parameters []*Identifier, body []Node) *Region {
	if body == nil {
		body = []Node{}
	}
	return &Region{pos, ident, parameters, body}
}

// ShowRegion represents a statement {% show <region> %}.
type ShowRegion struct {
	*Position              // position in the source.
	Import    *Identifier  // name of the import.
	Region    *Identifier  // name of the region.
	Arguments []Expression // arguments.
}

func NewShowRegion(pos *Position, impor, region *Identifier, arguments []Expression) *ShowRegion {
	return &ShowRegion{Position: pos, Import: impor, Region: region, Arguments: arguments}
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
	Path      string  // path del file da estendere.
	Context   Context // contesto.
	Tree      *Tree   // albero esteso di extend.
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
	Ident     *Identifier // identificatore.
	Path      string      // path del file da importato.
	Context   Context     // contesto.
	Tree      *Tree       // albero esteso di import.
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
	*Position // position in the source.
	expression
	Expr Expression // expression.
}

func NewParentesis(pos *Position, expr Expression) *Parentesis {
	return &Parentesis{pos, expression{}, expr}
}

func (n *Parentesis) String() string {
	return "(" + n.Expr.String() + ")"
}

type Int struct {
	*Position // position in the source.
	expression
	Value int // value.
}

func NewInt(pos *Position, value int) *Int {
	return &Int{pos, expression{}, value}
}

func (n *Int) String() string {
	return strconv.Itoa(n.Value)
}

type Number struct {
	*Position // position in the source.
	expression
	Value decimal.Decimal // valore.
}

func NewNumber(pos *Position, value decimal.Decimal) *Number {
	return &Number{pos, expression{}, value}
}

func (n *Number) String() string {
	return n.Value.String()
}

type String struct {
	*Position // position in the source.
	expression
	Text string // expression.
}

func NewString(pos *Position, text string) *String {
	return &String{pos, expression{}, text}
}

func (n *String) String() string {
	return strconv.Quote(n.Text)
}

type Identifier struct {
	*Position // position in the source.
	expression
	Name string // nome.
}

func NewIdentifier(pos *Position, name string) *Identifier {
	return &Identifier{pos, expression{}, name}
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
	*Position // position in the source.
	expression
	Op   OperatorType // operator.
	Expr Expression   // expression.
}

func NewUnaryOperator(pos *Position, op OperatorType, expr Expression) *UnaryOperator {
	return &UnaryOperator{pos, expression{}, op, expr}
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
	*Position // position in the source.
	expression
	Op    OperatorType // operator.
	Expr1 Expression   // first expression.
	Expr2 Expression   // second expression.
}

func NewBinaryOperator(pos *Position, op OperatorType, expr1, expr2 Expression) *BinaryOperator {
	return &BinaryOperator{pos, expression{}, op, expr1, expr2}
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
	*Position // position in the source.
	expression
	Func Expression   // function.
	Args []Expression // arguments.
}

func NewCall(pos *Position, fun Expression, args []Expression) *Call {
	return &Call{pos, expression{}, fun, args}
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
	*Position // position in the source.
	expression
	Expr  Expression // expression.
	Index Expression // index.
}

func NewIndex(pos *Position, expr Expression, index Expression) *Index {
	return &Index{pos, expression{}, expr, index}
}

func (n *Index) String() string {
	return n.Expr.String() + "[" + n.Index.String() + "]"
}

type Slice struct {
	*Position // position in the source.
	expression
	Expr Expression // expression.
	Low  Expression // low bound.
	High Expression // high bound.
}

func NewSlice(pos *Position, expr, low, high Expression) *Slice {
	return &Slice{pos, expression{}, expr, low, high}
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
	*Position // position in the source.
	expression
	Expr  Expression // expression.
	Ident string     // identifier.
}

func NewSelector(pos *Position, expr Expression, ident string) *Selector {
	return &Selector{pos, expression{}, expr, ident}
}

func (n *Selector) String() string {
	return n.Expr.String() + "." + n.Ident
}

func CloneTree(tree *Tree) *Tree {
	return CloneNode(tree).(*Tree)
}

func CloneNode(node Node) Node {
	switch n := node.(type) {
	case *Tree:
		var nn = make([]Node, 0, len(n.Nodes))
		for _, n := range n.Nodes {
			nn = append(nn, CloneNode(n))
		}
		return NewTree(n.Path, nn)
	case *Text:
		return NewText(ClonePosition(n.Position), n.Text)
	case *Value:
		return NewValue(ClonePosition(n.Position), CloneExpression(n.Expr), n.Context)
	case *If:
		var then = make([]Node, len(n.Then))
		for i, n2 := range n.Then {
			then[i] = CloneNode(n2)
		}
		var els []Node
		if n.Else != nil {
			els = make([]Node, len(n.Else))
			for i, n2 := range n.Else {
				els[i] = CloneNode(n2)
			}
		}
		return NewIf(ClonePosition(n.Position), CloneExpression(n.Expr), then, els)
	case *For:
		var nodes = make([]Node, len(n.Nodes))
		for i, n2 := range n.Nodes {
			nodes[i] = CloneNode(n2)
		}
		var index, ident *Identifier
		if n.Index != nil {
			index = NewIdentifier(ClonePosition(n.Index.Position), n.Index.Name)
		}
		if n.Ident != nil {
			ident = NewIdentifier(ClonePosition(n.Ident.Position), n.Ident.Name)
		}
		var expr2 Expression
		if n.Expr2 != nil {
			expr2 = CloneExpression(n.Expr2)
		}
		return NewFor(ClonePosition(n.Position), index, ident, CloneExpression(n.Expr1), expr2, nodes)
	case *Break:
		return NewBreak(ClonePosition(n.Position))
	case *Continue:
		return NewContinue(ClonePosition(n.Position))
	case *Extend:
		extend := NewExtend(ClonePosition(n.Position), n.Path, n.Context)
		if n.Tree != nil {
			extend.Tree = CloneTree(n.Tree)
		}
		return extend
	case *Region:
		var ident = NewIdentifier(ClonePosition(n.Ident.Position), n.Ident.Name)
		var parameters []*Identifier
		if n.Parameters != nil {
			parameters = make([]*Identifier, len(n.Parameters))
			for i, p := range n.Parameters {
				parameters[i] = NewIdentifier(ClonePosition(p.Position), p.Name)
			}
		}
		var body = make([]Node, len(n.Body))
		for i, n2 := range n.Body {
			body[i] = CloneNode(n2)
		}
		return NewRegion(ClonePosition(n.Position), ident, parameters, body)
	case *ShowRegion:
		var impor *Identifier
		if n.Import != nil {
			impor = NewIdentifier(ClonePosition(n.Import.Position), n.Import.Name)
		}
		var region = NewIdentifier(ClonePosition(n.Region.Position), n.Region.Name)
		var arguments []Expression
		if n.Arguments != nil {
			arguments = make([]Expression, len(n.Arguments))
			for i, a := range n.Arguments {
				arguments[i] = CloneExpression(a)
			}
		}
		return NewShowRegion(ClonePosition(n.Position), impor, region, arguments)
	case *Import:
		var ident *Identifier
		if n.Ident != nil {
			ident = NewIdentifier(ClonePosition(n.Ident.Position), n.Ident.Name)
		}
		imp := NewImport(ClonePosition(n.Position), ident, n.Path, n.Context)
		if n.Tree != nil {
			imp.Tree = CloneTree(n.Tree)
		}
		return imp
	case *ShowPath:
		sp := NewShowPath(ClonePosition(n.Position), n.Path, n.Context)
		if n.Tree != nil {
			sp.Tree = CloneTree(n.Tree)
		}
		return sp
	case *Comment:
		return NewComment(ClonePosition(n.Position), n.Text)
	case Expression:
		return CloneExpression(n)
	default:
		panic(fmt.Sprintf("unexpected node type %#v", node))
	}
}

func CloneExpression(expr Expression) Expression {
	switch e := expr.(type) {
	case *Parentesis:
		return NewParentesis(ClonePosition(e.Position), CloneExpression(e.Expr))
	case *Int:
		return NewInt(ClonePosition(e.Position), e.Value)
	case *Number:
		return NewNumber(ClonePosition(e.Position), e.Value)
	case *String:
		return NewString(ClonePosition(e.Position), e.Text)
	case *Identifier:
		return NewIdentifier(ClonePosition(e.Position), e.Name)
	case *UnaryOperator:
		return NewUnaryOperator(ClonePosition(e.Position), e.Op, CloneExpression(e.Expr))
	case *BinaryOperator:
		return NewBinaryOperator(ClonePosition(e.Position), e.Op, CloneExpression(e.Expr1), CloneExpression(e.Expr2))
	case *Call:
		var args = make([]Expression, 0, len(e.Args))
		for _, arg := range e.Args {
			args = append(args, CloneExpression(arg))
		}
		return NewCall(ClonePosition(e.Position), CloneExpression(e.Func), args)
	case *Index:
		return NewIndex(ClonePosition(e.Position), CloneExpression(e.Expr), CloneExpression(e.Index))
	case *Selector:
		return NewSelector(ClonePosition(e.Position), CloneExpression(e.Expr), e.Ident)
	default:
		panic(fmt.Sprintf("unexpected node type %#v", expr))
	}
}

func ClonePosition(pos *Position) *Position {
	return &Position{pos.Line, pos.Column, pos.Start, pos.End}
}
