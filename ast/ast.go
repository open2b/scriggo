//
// Copyright (c) 2016-2017 Open2b Software Snc. All Rights Reserved.
//

// Package ast dichiara i tipi usati per definire gli alberi dei template.
package ast

import (
	"fmt"
	"strconv"
	"sync"

	"open2b/template/types"
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

// Context indica il contesto in cui si trova un nodo Show.
type Context int

const (
	ContextHTML   Context = iota // codice HTML
	ContextScript                // script
	ContextStyle                 // stile
)

// Node è un elemento del tree.
type Node interface {
	Pos() *Position // posizione del nodo nel sorgente originale
}

// Position è una posizione di un nodo nel sorgente.
type Position struct {
	Line   int // linea a partire da 1
	Column int // colonna in caratteri a partire da 1
	Start  int // indice del primo byte
	End    int // indice dell'ultimo byte
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

// Tree rappresenta un albero parsato.
type Tree struct {
	*Position
	Path       string // path dell'albero.
	Nodes      []Node // nodi di primo livello dell'albero.
	IsExpanded bool   // indica se l'albero è stato espanso.
	sync.Mutex        // mutex utilizzato durante l'espansione dell'albero.
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

// Text rappresenta un testo
type Text struct {
	*Position         // posizione nel sorgente.
	Text      string  // testo.
	Cut       TextCut // taglio.
}

func NewText(pos *Position, text string) *Text {
	return &Text{pos, text, TextCut{0, len(text)}}
}

// Var rappresenta uno statement {% var identifier = expression %}
type Var struct {
	*Position             // posizione nel sorgente.
	Ident     *Identifier // identificatore.
	Expr      Expression  // espressione assegnata.
}

func NewVar(pos *Position, ident *Identifier, expr Expression) *Var {
	return &Var{pos, ident, expr}
}

// Assignment rappresenta uno statement {% identifier = expression %}.
type Assignment struct {
	*Position             // posizione nel sorgente.
	Ident     *Identifier // identificatore.
	Expr      Expression  // espressione assegnata.
}

func NewAssignment(pos *Position, ident *Identifier, expr Expression) *Assignment {
	return &Assignment{pos, ident, expr}
}

// For rappresenta uno statement {% for ... %}.
type For struct {
	*Position             // posizione nel sorgente.
	Index     *Identifier // indice.
	Ident     *Identifier // identificatore.
	Expr1     Expression  // espressione sinistra del range o slice su cui iterare
	Expr2     Expression  // espressione destra del range
	Nodes     []Node      // nodi da eseguire per ogni elemento della lista.
}

func NewFor(pos *Position, index, ident *Identifier, expr1, expr2 Expression, nodes []Node) *For {
	if nodes == nil {
		nodes = []Node{}
	}
	return &For{pos, index, ident, expr1, expr2, nodes}
}

// Break rappresenta uno statement {% break %}.
type Break struct {
	*Position // posizione nel sorgente.
}

func NewBreak(pos *Position) *Break {
	return &Break{pos}
}

// Continue rappresenta uno statement {% continue %}.
type Continue struct {
	*Position // posizione nel sorgente.
}

func NewContinue(pos *Position) *Continue {
	return &Continue{pos}
}

// If rappresenta uno statement {% if ... %}.
type If struct {
	*Position            // posizione nel sorgente.
	Expr      Expression // espressione che valutata restituisce true o false.
	Then      []Node     // nodi da eseguire se l'espressione è valutata a vero.
	Else      []Node     // nodi da eseguire se l'espressione è valutata a falso.
}

func NewIf(pos *Position, expr Expression, then []Node, els []Node) *If {
	if then == nil {
		then = []Node{}
	}
	return &If{pos, expr, then, els}
}

// Region rappresenta uno statement {% region ... %}.
type Region struct {
	*Position                // posizione nel sorgente.
	Ident      *Identifier   // nome.
	Parameters []*Identifier // parametri.
	Body       []Node        // corpo.
}

func NewRegion(pos *Position, ident *Identifier, parameters []*Identifier, body []Node) *Region {
	if body == nil {
		body = []Node{}
	}
	return &Region{pos, ident, parameters, body}
}

// ShowRegion rappresenta uno statement {% show <region> %}
type ShowRegion struct {
	*Position              // posizione nel sorgente.
	Import    *Identifier  // nome dell'import.
	Region    *Identifier  // nome della region.
	Arguments []Expression // argomenti.
	Ref       struct {
		Import *Import
		Region *Region
	}
	Context
}

func NewShowRegion(pos *Position, impor, region *Identifier, arguments []Expression, ctx Context) *ShowRegion {
	return &ShowRegion{Position: pos, Import: impor, Region: region, Arguments: arguments, Context: ctx}
}

// ShowPath rappresenta uno statement {% show <path> %}.
type ShowPath struct {
	*Position        // posizione nel sorgente.
	Path      string // path del sorgente da mostrare.
	Ref       struct {
		Tree *Tree
	}
	Context
}

func NewShowPath(pos *Position, path string, ctx Context) *ShowPath {
	return &ShowPath{Position: pos, Path: path, Context: ctx}
}

// Value rappresenta uno statement {{ ... }}
type Value struct {
	*Position            // posizione nel sorgente.
	Expr      Expression // espressione che valutata restituisce il valore da mostrare.
	Context
}

func NewValue(pos *Position, expr Expression, ctx Context) *Value {
	return &Value{pos, expr, ctx}
}

// Extend rappresenta uno statement {% extend ... %}.
type Extend struct {
	*Position        // posizione nel sorgente.
	Path      string // path del file da estendere.
	Ref       struct {
		Tree *Tree // albero del file esteso.
	}
}

func NewExtend(pos *Position, path string) *Extend {
	return &Extend{Position: pos, Path: path}
}

// Import rappresenta uno statement {% import ... %}.
type Import struct {
	*Position             // posizione nel sorgente.
	Ident     *Identifier // identificatore.
	Path      string      // path del file da importato.
	Ref       struct {
		Tree *Tree // albero del file importato.
	}
}

func NewImport(pos *Position, ident *Identifier, path string) *Import {
	return &Import{Position: pos, Ident: ident, Path: path}
}

type Comment struct {
	*Position        // posizione nel sorgente.
	Text      string // testo del commento.
}

func NewComment(pos *Position, text string) *Comment {
	return &Comment{pos, text}
}

type Parentesis struct {
	*Position // posizione nel sorgente.
	expression
	Expr Expression // espressione.
}

func NewParentesis(pos *Position, expr Expression) *Parentesis {
	return &Parentesis{pos, expression{}, expr}
}

func (n *Parentesis) String() string {
	return "(" + n.Expr.String() + ")"
}

type Number struct {
	*Position // posizione nel sorgente.
	expression
	Value types.Number // valore.
}

func NewNumber(pos *Position, value types.Number) *Number {
	return &Number{pos, expression{}, value}
}

func (n *Number) String() string {
	return n.Value.String()
}

type String struct {
	*Position // posizione nel sorgente.
	expression
	Text string // espressione.
}

func NewString(pos *Position, text string) *String {
	return &String{pos, expression{}, text}
}

func (n *String) String() string {
	return strconv.Quote(n.Text)
}

type Identifier struct {
	*Position // posizione nel sorgente.
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
	*Position // posizione nel sorgente.
	expression
	Op   OperatorType // operatore.
	Expr Expression   // espressione.
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
	*Position // posizione nel sorgente.
	expression
	Op    OperatorType // operatore.
	Expr1 Expression   // prima espressione.
	Expr2 Expression   // seconda espressione.
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
	s += n.Op.String()
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
	*Position // posizione nel sorgente.
	expression
	Func Expression   // funzione.
	Args []Expression // parametri.
}

func NewCall(pos *Position, fun Expression, args []Expression) *Call {
	return &Call{pos, expression{}, fun, args}
}

func (n *Call) String() string {
	s := n.Func.String() + "("
	for i, arg := range n.Args {
		s += arg.String()
		if i < len(n.Args)-1 {
			s += ","
		}
	}
	s += ")"
	return s
}

type Index struct {
	*Position // posizione nel sorgente.
	expression
	Expr  Expression // espressione.
	Index Expression // index.
}

func NewIndex(pos *Position, expr Expression, index Expression) *Index {
	return &Index{pos, expression{}, expr, index}
}

func (n *Index) String() string {
	return n.Expr.String() + "[" + n.Index.String() + "]"
}

type Slice struct {
	*Position // posizione nel sorgente.
	expression
	Expr Expression // espressione.
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
	*Position // posizione nel sorgente.
	expression
	Expr  Expression // espressione.
	Ident string     // identificatore.
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
		extend := NewExtend(ClonePosition(n.Position), n.Path)
		if n.Ref.Tree != nil {
			extend.Ref.Tree = CloneTree(n.Ref.Tree)
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
		return NewShowRegion(ClonePosition(n.Position), impor, region, arguments, n.Context)
	case *Import:
		var ident *Identifier
		if n.Ident != nil {
			ident = NewIdentifier(ClonePosition(n.Ident.Position), n.Ident.Name)
		}
		imp := NewImport(ClonePosition(n.Position), ident, n.Path)
		if n.Ref.Tree != nil {
			imp.Ref.Tree = CloneTree(n.Ref.Tree)
		}
		return imp
	case *ShowPath:
		sp := NewShowPath(ClonePosition(n.Position), n.Path, n.Context)
		if n.Ref.Tree != nil {
			sp.Ref.Tree = CloneTree(n.Ref.Tree)
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
