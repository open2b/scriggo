//
// Copyright (c) 2016 Open2b Software Snc. All Rights Reserved.
//

package ast

import (
	"open2b/decimal"
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

// Context indica il contesto in cui si trova un nodo Show.
type Context int

const (
	ContextHTML      Context = iota // codice HTML
	ContextAttribute                // valore di un attributo
	ContextScript                   // script
	ContextStyle                    // stile
)

// Node è un elemento del tree.
type Node interface {
	Pos() int // posizione in byte del nodo nel sorgente originale
}

// position è una posizione in bytes di un nodo nel sorgente.
type position int

func (p position) Pos() int {
	return int(p)
}

type Expression interface {
	Node
	isexpr()
}

type expression struct{}

func (e expression) isexpr() {}

// Tree rappresenta un albero parsato.
type Tree struct {
	position
	Nodes []Node // nodi di primo livello dell'albero.
}

func NewTree(nodes []Node) *Tree {
	if nodes == nil {
		nodes = []Node{}
	}
	return &Tree{position(0), nodes}
}

// Text rappresenta un testo
type Text struct {
	position        // posizione nel sorgente.
	Text     string // testo.
}

func NewText(pos int, text string) *Text {
	return &Text{position(pos), text}
}

// Var rappresenta uno statement {% var identifier = expression %}
type Var struct {
	position             // posizione nel sorgente.
	Ident    *Identifier // identificatore.
	Expr     Expression  // espressione assegnata.
}

func NewVar(pos int, ident *Identifier, expr Expression) *Var {
	return &Var{position(pos), ident, expr}
}

// Assignment rappresenta uno statement {% identifier = expression %}.
type Assignment struct {
	position             // posizione nel sorgente.
	Ident    *Identifier // identificatore.
	Expr     Expression  // espressione assegnata.
}

func NewAssignment(pos int, ident *Identifier, expr Expression) *Assignment {
	return &Assignment{position(pos), ident, expr}
}

// For rappresenta uno statement {% for ... %}.
type For struct {
	position            // posizione nel sorgente.
	Expr     Expression // espressione che valutata restituisce la lista degli elementi.
	Nodes    []Node     // nodi da eseguire per ogni elemento della lista.
}

func NewFor(pos int, expr Expression, nodes []Node) *For {
	if nodes == nil {
		nodes = []Node{}
	}
	return &For{position(pos), expr, nodes}
}

// If rappresenta uno statement {% if ... %}.
type If struct {
	position            // posizione nel sorgente.
	Expr     Expression // espressione che valutata restituisce true o false.
	Nodes    []Node     // nodi da eseguire se l'espressione è valutata a vero.
}

func NewIf(pos int, expr Expression, nodes []Node) *If {
	if nodes == nil {
		nodes = []Node{}
	}
	return &If{position(pos), expr, nodes}
}

// Show rappresenta uno statement {% show ... %} o {{ ... }}.
type Show struct {
	position            // posizione nel sorgente.
	Expr     Expression // espressione che valutata restituisce il testo da mostrare.
	Text     *Text      // testo esemplificativo
	Context
}

func NewShow(pos int, expr Expression, text *Text, ctx Context) *Show {
	return &Show{position(pos), expr, text, ctx}
}

// Extend rappresenta uno statement {% extend ... %}.
type Extend struct {
	position        // posizione nel sorgente.
	Path     string // path del file da estendere.
}

func NewExtend(pos int, path string) *Extend {
	return &Extend{position(pos), path}
}

// Region rappresenta uno statement {% region ... %}.
type Region struct {
	position        // posizione nel sorgente.
	Name     string // name.
	Nodes    []Node // nodi facenti parte della region.
}

func NewRegion(pos int, name string, nodes []Node) *Region {
	if nodes == nil {
		nodes = []Node{}
	}
	return &Region{position(pos), name, nodes}
}

// Include rappresenta uno statement {% include ... %}.
type Include struct {
	position        // posizione nel sorgente.
	Path     string // path del file da includere.
	Nodes    []Node // nodi del file incluso.
}

func NewInclude(pos int, path string, nodes []Node) *Include {
	if nodes == nil {
		nodes = []Node{}
	}
	return &Include{position(pos), path, nodes}
}

// Snippet rappresenta uno statement {% snippet ... %}.
type Snippet struct {
	position        // posizione nel sorgente.
	Prefix   string // prefisso, vuoto se non è presente.
	Name     string // nome.
}

func NewSnippet(pos int, prefix, name string) *Snippet {
	return &Snippet{position(pos), prefix, name}
}

type Parentesis struct {
	position // posizione nel sorgente.
	expression
	Expr Expression // espressione.
}

func NewParentesis(pos int, expr Expression) *Parentesis {
	return &Parentesis{position(pos), expression{}, expr}
}

type Int32 struct {
	position // posizione nel sorgente.
	expression
	Value int32 // valore.
}

func NewInt32(pos int, value int32) *Int32 {
	return &Int32{position(pos), expression{}, value}
}

type Int64 struct {
	position // posizione nel sorgente.
	expression
	Value int64 // valore.
}

func NewInt64(pos int, value int64) *Int64 {
	return &Int64{position(pos), expression{}, value}
}

type Decimal struct {
	position // posizione nel sorgente.
	expression
	Value decimal.Dec // valore.
}

func NewDecimal(pos int, value decimal.Dec) *Decimal {
	return &Decimal{position(pos), expression{}, value}
}

type String struct {
	position // posizione nel sorgente.
	expression
	Text string // espressione.
}

func NewString(pos int, text string) *String {
	return &String{position(pos), expression{}, text}
}

type Identifier struct {
	position // posizione nel sorgente.
	expression
	Name string // nome.
}

func NewIdentifier(pos int, name string) *Identifier {
	return &Identifier{position(pos), expression{}, name}
}

type Operator interface {
	Expression
	Operator() OperatorType
	Precedence() int
}

type UnaryOperator struct {
	position // posizione nel sorgente.
	expression
	Op   OperatorType // operatore.
	Expr Expression   // espressione.
}

func NewUnaryOperator(pos int, op OperatorType, expr Expression) *UnaryOperator {
	return &UnaryOperator{position(pos), expression{}, op, expr}
}

func (n *UnaryOperator) Operator() OperatorType {
	return n.Op
}

func (n *UnaryOperator) Precedence() int {
	return 6
}

type BinaryOperator struct {
	position // posizione nel sorgente.
	expression
	Op    OperatorType // operatore.
	Expr1 Expression   // prima espressione.
	Expr2 Expression   // seconda espressione.
}

func NewBinaryOperator(pos int, op OperatorType, expr1, expr2 Expression) *BinaryOperator {
	return &BinaryOperator{position(pos), expression{}, op, expr1, expr2}
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
	position // posizione nel sorgente.
	expression
	Func Expression   // funzione.
	Args []Expression // parametri.
}

func NewCall(pos int, fun Expression, args []Expression) *Call {
	return &Call{position(pos), expression{}, fun, args}
}

type Index struct {
	position // posizione nel sorgente.
	expression
	Expr  Expression // espressione.
	Index Expression // index.
}

func NewIndex(pos int, expr Expression, index Expression) *Index {
	return &Index{position(pos), expression{}, expr, index}
}

type Selector struct {
	position // posizione nel sorgente.
	expression
	Expr  Expression // espressione.
	Ident string     // identificatore.
}

func NewSelector(pos int, expr Expression, ident string) *Selector {
	return &Selector{position(pos), expression{}, expr, ident}
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
		return NewTree(nn)
	case *Text:
		return NewText(int(n.position), n.Text)
	case *Show:
		var text *Text
		if n.Text != nil {
			text = NewText(int(n.Text.position), n.Text.Text)
		}
		return NewShow(int(n.position), CloneExpression(n.Expr), text, n.Context)
	case *If:
		var nodes = make([]Node, 0, len(n.Nodes))
		for _, n2 := range n.Nodes {
			nodes = append(nodes, CloneNode(n2))
		}
		return NewIf(int(n.position), CloneExpression(n.Expr), nodes)
	case *For:
		var nodes = make([]Node, 0, len(n.Nodes))
		for _, n2 := range n.Nodes {
			nodes = append(nodes, CloneNode(n2))
		}
		return NewFor(int(n.position), CloneExpression(n.Expr), nodes)
	case *Extend:
		return NewExtend(int(n.position), n.Path)
	case *Region:
		var nodes = make([]Node, 0, len(n.Nodes))
		for _, n2 := range n.Nodes {
			nodes = append(nodes, CloneNode(n2))
		}
		return NewRegion(int(n.position), n.Name, nodes)
	case *Include:
		var nodes = make([]Node, 0, len(n.Nodes))
		for _, n2 := range n.Nodes {
			nodes = append(nodes, CloneNode(n2))
		}
		return NewInclude(int(n.position), n.Path, nodes)
	case Expression:
		return CloneExpression(n)
	default:
		panic("unexpected node type")
	}
}

func CloneExpression(expr Expression) Expression {
	switch e := expr.(type) {
	case *Parentesis:
		return NewParentesis(int(e.position), CloneExpression(e.Expr))
	case *Int32:
		return NewInt32(int(e.position), e.Value)
	case *Int64:
		return NewInt64(int(e.position), e.Value)
	case *Decimal:
		return NewDecimal(int(e.position), e.Value)
	case *String:
		return NewString(int(e.position), e.Text)
	case *Identifier:
		return NewIdentifier(int(e.position), e.Name)
	case *UnaryOperator:
		return NewUnaryOperator(int(e.position), e.Op, CloneExpression(e.Expr))
	case *BinaryOperator:
		return NewBinaryOperator(int(e.position), e.Op, CloneExpression(e.Expr1), CloneExpression(e.Expr2))
	case *Call:
		var args = make([]Expression, 0, len(e.Args))
		for _, arg := range e.Args {
			args = append(args, CloneExpression(arg))
		}
		return NewCall(int(e.position), CloneExpression(e.Func), args)
	case *Index:
		return NewIndex(int(e.position), CloneExpression(e.Expr), CloneExpression(e.Index))
	default:
		panic("unexpected node type")
	}
}
