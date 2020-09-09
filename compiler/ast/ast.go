// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ast declares the types used to define the template trees.
//
// For example, the source in an "articles.html" file:
//
//    {% for article in articles %}
//    <div>{{ article.title }}</div>
//    {% end %}
//
// is represented with the tree:
//
// 		ast.NewTree("articles.txt", []ast.Node{
//			ast.NewFor(
//				&ast.Position{Line: 1, Column: 1, Start: 0, End: 69},
//				nil,
//				ast.NewIdentifier(&ast.Position{Line: 1, Column: 8, Start: 7, End: 13}, "article"),
//				ast.NewIdentifier(&ast.Position{Line: 1, Column: 19, Start: 18, End: 25}, "articles"),
//				nil,
//				[]ast.Node{
//					ast.NewText(&ast.Position{Line: 1, Column: 30, Start: 29, End: 34}, []byte("\n<div>"), ast.Cut{1,0}),
//					ast.NewShow(
//						&ast.Position{Line: 2, Column: 6, Start: 35, End: 53},
//						ast.NewSelector(
//							&ast.Position{Line: 2, Column: 16, Start: 38, End: 50},
//							ast.NewIdentifier(
//								&ast.Position{Line: 2, Column: 9, Start: 38, End: 44},
//								"article",
//							),
//							"title"),
//						ast.ContextHTML),
//					ast.NewText(&ast.Position{Line: 2, Column: 25, Start: 54, End: 60}, []byte("</div>\n"), ast.Cut{})),
//				},
//			),
//		}, ast.ContextHTML)
//
package ast

import (
	"fmt"
	"reflect"
	"strconv"
)

// expandedPrint is set to true in tests to print completely a composite
// literal.
var expandedPrint = false

// OperatorType represents an operator type in an unary and binary expression.
type OperatorType int

const (
	OperatorEqual          OperatorType = iota // ==
	OperatorNotEqual                           // !=
	OperatorLess                               // <
	OperatorLessEqual                          // <=
	OperatorGreater                            // >
	OperatorGreaterEqual                       // >=
	OperatorNot                                // !
	OperatorBitAnd                             // &
	OperatorBitOr                              // |
	OperatorAnd                                // &&
	OperatorOr                                 // ||
	OperatorAddition                           // +
	OperatorSubtraction                        // -
	OperatorMultiplication                     // *
	OperatorDivision                           // /
	OperatorModulo                             // %
	OperatorXor                                // ^
	OperatorAndNot                             // &^
	OperatorLeftShift                          // <<
	OperatorRightShift                         // >>
	OperatorReceive                            // <-
	OperatorAddress                            // &
	OperatorPointer                            // *
	OperatorRelaxedAnd                         // and
	OperatorRelaxedOr                          // or
	OperatorRelaxedNot                         // not
)

func (op OperatorType) String() string {
	return []string{"==", "!=", "<", "<=", ">", ">=", "!", "&", "|", "&&", "||",
		"+", "-", "*", "/", "%", "^", "&^", "<<", ">>", "<-", "&", "*", "and", "or", "not"}[op]
}

type AssignmentType int

const (
	AssignmentSimple         AssignmentType = iota // =
	AssignmentDeclaration                          // :=
	AssignmentAddition                             // +=
	AssignmentSubtraction                          // -=
	AssignmentMultiplication                       // *=
	AssignmentDivision                             // /=
	AssignmentModulo                               // %=
	AssignmentAnd                                  // &=
	AssignmentOr                                   // |=
	AssignmentXor                                  // ^=
	AssignmentAndNot                               // &^=
	AssignmentLeftShift                            // <<=
	AssignmentRightShift                           // >>=
	AssignmentIncrement                            // ++
	AssignmentDecrement                            // --
)

// A Language represents a source language.
type Language int

const (
	LanguageText Language = iota
	LanguageHTML
	LanguageCSS
	LanguageJavaScript
	LanguageJSON
	LanguageGo // TODO(marco): move in first position.
)

func (lang Language) String() string {
	switch lang {
	case LanguageGo:
		return "Go"
	case LanguageText:
		return "text"
	case LanguageHTML:
		return "HTML"
	case LanguageCSS:
		return "CSS"
	case LanguageJavaScript:
		return "JavaScript"
	case LanguageJSON:
		return "JSON"
	}
	panic("invalid language")
}

// Context indicates the context in which a value statement must be valuated.
type Context int

const (
	ContextText Context = iota
	ContextHTML
	ContextCSS
	ContextJavaScript
	ContextJSON
	ContextGo // TODO(marco): move in first position.
	ContextTag
	ContextAttribute
	ContextUnquotedAttribute
	ContextCSSString
	ContextJavaScriptString
	ContextJSONString
)

func (ctx Context) String() string {
	switch ctx {
	case ContextGo:
		return "Go"
	case ContextText:
		return "text"
	case ContextHTML:
		return "HTML"
	case ContextCSS:
		return "CSS"
	case ContextJavaScript:
		return "JavaScript"
	case ContextJSON:
		return "JSON"
	case ContextTag:
		return "tag"
	case ContextAttribute:
		return "attribute"
	case ContextUnquotedAttribute:
		return "unquoted attribute"
	case ContextCSSString:
		return "CSS string"
	case ContextJavaScriptString:
		return "JavaScript string"
	case ContextJSONString:
		return "JSON string"
	}
	panic("invalid context")
}

type LiteralType int

const (
	StringLiteral LiteralType = iota
	RuneLiteral
	IntLiteral
	FloatLiteral
	ImaginaryLiteral
)

type ChanDirection int

const (
	NoDirection ChanDirection = iota
	ReceiveDirection
	SendDirection
)

var directionString = [3]string{"no direction", "receive", "send"}

func (dir ChanDirection) String() string { return directionString[dir] }

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

// Expression node represents an expression.
type Expression interface {
	Parenthesis() int
	SetParenthesis(int)
	Node
	String() string
}

// expression represents an expression.
type expression struct {
	parenthesis int
}

// Parenthesis returns the number of parenthesis around the expression.
func (e *expression) Parenthesis() int {
	return e.parenthesis
}

// SetParenthesis sets the number of parenthesis around the expression.
func (e *expression) SetParenthesis(n int) {
	e.parenthesis = n
}

// Tree node represents a tree.
type Tree struct {
	*Position
	Path     string   // path of the tree.
	Nodes    []Node   // nodes of the first level of the tree.
	Language Language // source language.
}

func NewTree(path string, nodes []Node, language Language) *Tree {
	if nodes == nil {
		nodes = []Node{}
	}
	tree := &Tree{
		Position: &Position{1, 1, 0, 0},
		Path:     path,
		Nodes:    nodes,
		Language: language,
	}
	return tree
}

// Cut, in a Text node, indicates how many bytes must be cut from the left and
// the right of the text before rendering the Text node.
type Cut struct {
	Left  int
	Right int
}

// Text node represents a text in the source.
type Text struct {
	*Position        // position in the source.
	Text      []byte // text.
	Cut       Cut    // cut.
}

func NewText(pos *Position, text []byte, cut Cut) *Text {
	return &Text{pos, text, cut}
}

func (n *Text) String() string {
	return string(n.Text)
}

// URL node represents an URL in an attribute value. Show nodes that are
// children of an URL node are rendered accordingly.
type URL struct {
	*Position         // position in the source.
	Tag       string  // tag (in lowercase).
	Attribute string  // attribute (in lowercase).
	Value     []Node  // value nodes.
	Context   Context // context.
}

func NewURL(pos *Position, tag, attribute string, value []Node, ctx Context) *URL {
	return &URL{pos, tag, attribute, value, ctx}
}

// Block node represents a block { ... } with his own scope.
type Block struct {
	*Position
	Nodes []Node
}

// NewBlock returns a new block statement.
func NewBlock(pos *Position, nodes []Node) *Block {
	return &Block{pos, nodes}
}

// Package node represents a package.
type Package struct {
	*Position
	Name         string // name.
	Declarations []Node
}

func NewPackage(pos *Position, name string, nodes []Node) *Package {
	return &Package{pos, name, nodes}
}

// Assignment node represents an assignment statement.
type Assignment struct {
	*Position                // position in the source.
	Lhs       []Expression   // left hand variables.
	Type      AssignmentType // type.
	Rhs       []Expression   // assigned values (nil for increment and decrement).
}

func NewAssignment(pos *Position, lhs []Expression, typ AssignmentType, rhs []Expression) *Assignment {
	return &Assignment{pos, lhs, typ, rhs}
}

func (n *Assignment) String() string {
	var s string
	for i, v := range n.Lhs {
		if i > 0 {
			s += ", "
		}
		s += v.String()
	}
	switch n.Type {
	case AssignmentSimple:
		s += " = "
	case AssignmentDeclaration:
		s += " := "
	case AssignmentAddition:
		s += " += "
	case AssignmentSubtraction:
		s += " -= "
	case AssignmentMultiplication:
		s += " *= "
	case AssignmentDivision:
		s += " /= "
	case AssignmentModulo:
		s += " %= "
	case AssignmentIncrement:
		s += "++"
	case AssignmentDecrement:
		s += "--"
	}
	if n.Rhs != nil {
		for i, value := range n.Rhs {
			if i > 0 {
				s += ", "
			}
			s += value.String()
		}
	}
	return s
}

// Parameter node represents a parameter in a function type, literal or
// declaration.
type Parameter struct {
	Ident *Identifier // name, can be nil.
	Type  Expression  // type.
}

func NewParameter(ident *Identifier, typ Expression) *Parameter {
	return &Parameter{ident, typ}
}

func (n *Parameter) String() string {
	if n.Ident == nil {
		return n.Type.String()
	}
	if n.Type == nil {
		return n.Ident.Name
	}
	return n.Ident.Name + " " + n.Type.String()
}

// FuncType node represents a function type.
type FuncType struct {
	expression
	*Position               // position in the source.
	Parameters []*Parameter // parameters.
	Result     []*Parameter // result.
	IsVariadic bool         // reports whether it is variadic.
	Reflect    reflect.Type // reflect type.
}

func NewFuncType(pos *Position, parameters []*Parameter, result []*Parameter, isVariadic bool) *FuncType {
	return &FuncType{expression{}, pos, parameters, result, isVariadic, nil}
}

func (n *FuncType) String() string {
	s := "func("
	for i, param := range n.Parameters {
		if i > 0 {
			s += ", "
		}
		s += param.String()
	}
	s += ")"
	if len(n.Result) > 0 {
		if n.Result[0].Ident == nil {
			s += " " + n.Result[0].Type.String()
		} else {
			s += " ("
			for i, res := range n.Result {
				if i > 0 {
					s += ", "
				}
				s += res.String()
			}
			s += ")"
		}
	}
	return s
}

// Func node represents a function declaration or literal.
type Func struct {
	expression
	*Position
	Ident  *Identifier // name, nil for function literals.
	Type   *FuncType   // type.
	Body   *Block      // body.
	Upvars []Upvar     // Upvars of func.
}

// Upvar represents a variable defined outside function body. Even package level
// variables (predefined or not) are considered upvars.
type Upvar struct {

	// PredefinedPkg is the name of the predefined package which holds a
	// predefined Upvar. If Upvar is not a predefined Upvar then PredefinedName
	// is an empty string.
	PredefinedPkg string

	// PredefinedName is the name of the predefined declaration of a predefined
	// Upvar. If Upvar is not a predefined Upvar then PredefinedName is an empty
	// string.
	PredefinedName string

	// PredefinedValue is the value of the predefined variable Upvar. If Upvar
	// is not a predefined then Upvar is nil.
	PredefinedValue *reflect.Value

	// Declaration is the ast node where Upvar is defined. If Upvar is a
	// predefined var then Declaration is nil.
	Declaration Node

	// Index indexes the Upvars slice of the parent function.
	// As a special case, Index is -1 when the Upvar declaration node is a
	// sibling of the function declaration node.
	//
	// Consider this example:
	//
	// 		var A
	// 		func g() {
	// 			func f() {
	// 				_ = A
	// 			}
	// 		}
	//
	// g has one upvar (A) with index -1 (node which declares A is a sibling of
	// the declaration of g)
	// f has one upvar (A) with index 0, which is the index of A in the Upvars slice of g.
	//
	// Another example:
	//
	// 		func g() {
	// 			var A
	// 			func f() {
	// 				_ = A
	// 			}
	// 		}
	//
	// g has no upvars
	// f has one upvar (A) with index -1 (declaration of A is a sibling of
	// declaration of f)
	//
	Index int16
}

func NewFunc(pos *Position, name *Identifier, typ *FuncType, body *Block) *Func {
	return &Func{expression{}, pos, name, typ, body, nil}
}

func (n *Func) String() string {
	if n.Ident == nil {
		return "func literal"
	}
	return "func declaration"
}

// Return node represents a return statement.
type Return struct {
	*Position
	Values []Expression // return values.
}

func NewReturn(pos *Position, values []Expression) *Return {
	return &Return{pos, values}
}

// For node represents a statement {% for ... %}.
type For struct {
	*Position            // position in the source.
	Init      Node       // initialization statement.
	Condition Expression // condition expression.
	Post      Node       // post iteration statement.
	Body      []Node     // nodes of the body.
}

func NewFor(pos *Position, init Node, condition Expression, post Node, body []Node) *For {
	if body == nil {
		body = []Node{}
	}
	return &For{pos, init, condition, post, body}
}

// ForRange node represents statements {% for ... range ... %} and
// {% for ... in ... %}.
type ForRange struct {
	*Position              // position in the source.
	Assignment *Assignment // assignment.
	Body       []Node      // nodes of the body.
}

func NewForRange(pos *Position, assignment *Assignment, body []Node) *ForRange {
	if body == nil {
		body = []Node{}
	}
	return &ForRange{pos, assignment, body}
}

// Break node represents a statement {% break %}.
type Break struct {
	*Position             // position in the source.
	Label     *Identifier // label.
}

func NewBreak(pos *Position, label *Identifier) *Break {
	return &Break{pos, label}
}

// Continue node represents a statement {% continue %}.
type Continue struct {
	*Position             // position in the source.
	Label     *Identifier // label.
}

func NewContinue(pos *Position, label *Identifier) *Continue {
	return &Continue{pos, label}
}

// If node represents a statement {% if ... %}.
type If struct {
	*Position            // position in the source.
	Init      Node       // init simple statement.
	Condition Expression // condition that once evaluated returns true or false.
	Then      *Block     // nodes to run if the expression is evaluated to true.
	Else      Node       // nodes to run if the expression is evaluated to false. Can be Block or If.
}

func NewIf(pos *Position, init Node, cond Expression, then *Block, els Node) *If {
	if then == nil {
		then = NewBlock(nil, []Node{})
	}
	return &If{pos, init, cond, then, els}
}

// Switch node represents a statement {% switch ... %}.
type Switch struct {
	*Position
	Init        Node
	Expr        Expression
	LeadingText *Text
	Cases       []*Case
}

// NewSwitch returns a new Switch node.
func NewSwitch(pos *Position, init Node, expr Expression, leadingText *Text, cases []*Case) *Switch {
	return &Switch{pos, init, expr, leadingText, cases}
}

// TypeSwitch node represents a statement {% switch ... %} on types.
type TypeSwitch struct {
	*Position
	Init        Node
	Assignment  *Assignment
	LeadingText *Text
	Cases       []*Case
}

// NewTypeSwitch returns a new TypeSwitch node.
func NewTypeSwitch(pos *Position, init Node, assignment *Assignment, leadingText *Text, cases []*Case) *TypeSwitch {
	return &TypeSwitch{pos, init, assignment, leadingText, cases}
}

// Case node represents a statement {% case ... %} or {% default %}.
type Case struct {
	*Position
	Expressions []Expression
	Body        []Node
}

// NewCase returns a new Case node.
func NewCase(pos *Position, expressions []Expression, body []Node) *Case {
	return &Case{pos, expressions, body}
}

// Fallthrough node represents a statement {% fallthrough %}.
type Fallthrough struct {
	*Position
}

// NewFallthrough returns a new Fallthrough node.
func NewFallthrough(pos *Position) *Fallthrough {
	return &Fallthrough{pos}
}

// Select node represents a statement {% select ... %}.
type Select struct {
	*Position
	LeadingText *Text
	Cases       []*SelectCase
}

// NewSelect returns a new Select node.
func NewSelect(pos *Position, leadingText *Text, cases []*SelectCase) *Select {
	return &Select{pos, leadingText, cases}
}

// NewSelectCase represents a statement {% case ... %} in a select.
type SelectCase struct {
	*Position
	Comm Node
	Body []Node
}

// NewSelectCase returns a new SelectCase node.
func NewSelectCase(pos *Position, comm Node, body []Node) *SelectCase {
	return &SelectCase{pos, comm, body}
}

// TypeDeclaration node represents a type declaration, that is an alias
// declaration or a type definition.
type TypeDeclaration struct {
	*Position                      // position in the source.
	Ident              *Identifier // identifier of the type.
	Type               Expression  // expression representing the type.
	IsAliasDeclaration bool        // reports whether it is an alias declaration or a type definition.
}

func (n *TypeDeclaration) String() string {
	if n.IsAliasDeclaration {
		return fmt.Sprintf("type %s = %s", n.Ident.Name, n.Type.String())
	}
	return fmt.Sprintf("type %s %s", n.Ident.Name, n.Type.String())
}

// NewTypeDeclaration returns a new TypeDeclaration node.
func NewTypeDeclaration(pos *Position, ident *Identifier, typ Expression, isAliasDeclaration bool) *TypeDeclaration {
	return &TypeDeclaration{pos, ident, typ, isAliasDeclaration}
}

// Macro node represents a statement {% macro ... %}.
type Macro struct {
	*Position             // position in the source.
	Ident     *Identifier // name.
	Type      *FuncType   // type.
	Body      []Node      // body.
	Upvars    []Upvar     // Upvars of macro.
	Context   Context     // context.
}

func NewMacro(pos *Position, name *Identifier, typ *FuncType, body []Node, ctx Context) *Macro {
	if body == nil {
		body = []Node{}
	}
	return &Macro{pos, name, typ, body, nil, ctx}
}

// ShowMacroOr specifies behavior when macro is not defined.
type ShowMacroOr int8

const (
	// ShowMacroOrIgnore ignores if not defined.
	ShowMacroOrIgnore ShowMacroOr = iota
	// ShowMacroOrTodo returns error if compiled with "fail on todo" option.
	ShowMacroOrTodo
	// ShowMacroOrError is the default behavior: returns an error.
	ShowMacroOrError
)

func (s ShowMacroOr) String() string {
	switch s {
	case ShowMacroOrIgnore:
		return "ShowMacroOrIgnore"
	case ShowMacroOrTodo:
		return "ShowMacroOrTodo"
	case ShowMacroOrError:
		return "ShowMacroOrError"
	default:
		panic("not defined")
	}
}

// ShowMacro node represents a statement {% show <macro> %}.
type ShowMacro struct {
	*Position               // position in the source.
	Macro      Expression   // macro.
	Args       []Expression // arguments.
	IsVariadic bool         // reports whether it is variadic.
	Or         ShowMacroOr  // when macro is not defined.
	Context    Context      // context.
}

func NewShowMacro(pos *Position, macro Expression, args []Expression, isVariadic bool, or ShowMacroOr, ctx Context) *ShowMacro {
	return &ShowMacro{Position: pos, Macro: macro, Args: args, IsVariadic: isVariadic, Or: or, Context: ctx}
}

// Include node represents a statement {% include <path> %}.
type ShowPartial struct {
	*Position         // position in the source.
	Path      string  // path of the source to include.
	Context   Context // context.
	Tree      *Tree   // expanded tree of <path>.
}

func NewShowPartial(pos *Position, path string, ctx Context) *ShowPartial {
	return &ShowPartial{Position: pos, Path: path, Context: ctx}
}

// Show node represents a statement {{ ... }}.
type Show struct {
	*Position            // position in the source.
	Expr      Expression // expression that once evaluated returns the value to show.
	Context   Context    // context.
}

func NewShow(pos *Position, expr Expression, ctx Context) *Show {
	return &Show{pos, expr, ctx}
}

func (n *Show) String() string {
	return fmt.Sprintf("{{ %v }}", n.Expr)
}

// Extends node represents a statement {% extends ... %}.
type Extends struct {
	*Position         // position in the source.
	Path      string  // path to extend.
	Context   Context // context.
	Tree      *Tree   // expanded tree of extends.
}

func NewExtends(pos *Position, path string, ctx Context) *Extends {
	return &Extends{Position: pos, Path: path, Context: ctx}
}

func (n *Extends) String() string {
	return fmt.Sprintf("{%% extends %v %%}", strconv.Quote(n.Path))
}

// Import node represents a statement {% import ... %}.
type Import struct {
	*Position             // position in the source.
	Ident     *Identifier // name (including "." and "_") or nil.
	Path      string      // path to import.
	Context   Context     // context.
	Tree      *Tree       // expanded tree of import.
}

func NewImport(pos *Position, ident *Identifier, path string, ctx Context) *Import {
	return &Import{Position: pos, Ident: ident, Path: path, Context: ctx}
}

func (n *Import) String() string {
	if n.Ident == nil {
		return fmt.Sprintf("{%% import %v %%}", strconv.Quote(n.Path))
	}

	return fmt.Sprintf("{%% import %v %v %%}", n.Ident, strconv.Quote(n.Path))
}

// Comment node represents a statement {# ... #}.
type Comment struct {
	*Position        // position in the source.
	Text      string // comment text.
}

func NewComment(pos *Position, text string) *Comment {
	return &Comment{pos, text}
}

type BasicLiteral struct {
	expression
	*Position             // position in the source.
	Type      LiteralType // type.
	Value     string      // value.
}

func NewBasicLiteral(pos *Position, typ LiteralType, value string) *BasicLiteral {
	return &BasicLiteral{expression{}, pos, typ, value}
}

func (n *BasicLiteral) String() string {
	return n.Value
}

// Identifier node represents an identifier expression.
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

// Operator represents an operator expression. It is implemented by
// the nodes UnaryOperator and BinaryOperator.
type Operator interface {
	Expression
	Operator() OperatorType
	Precedence() int
}

// UnaryOperator node represents an unary operator expression.
type UnaryOperator struct {
	*expression
	*Position              // position in the source.
	Op        OperatorType // operator.
	Expr      Expression   // expression.
}

func NewUnaryOperator(pos *Position, op OperatorType, expr Expression) *UnaryOperator {
	return &UnaryOperator{&expression{}, pos, op, expr}
}

func (n *UnaryOperator) String() string {
	s := n.Op.String()
	if n.Op == OperatorRelaxedNot {
		s += " "
	}
	if e, ok := n.Expr.(Operator); ok && (n.Op == OperatorReceive || e.Precedence() <= n.Precedence()) {
		s += "(" + n.Expr.String() + ")"
	} else {
		s += n.Expr.String()
	}
	return s
}

// Operator returns the operator type of the expression.
func (n *UnaryOperator) Operator() OperatorType {
	return n.Op
}

// Precedence returns a number that represents the precedence of the
// expression.
func (n *UnaryOperator) Precedence() int {
	return 6
}

// BinaryOperator node represents a binary operator expression.
type BinaryOperator struct {
	*expression
	*Position              // position in the source.
	Op        OperatorType // operator.
	Expr1     Expression   // first expression.
	Expr2     Expression   // second expression.
}

func NewBinaryOperator(pos *Position, op OperatorType, expr1, expr2 Expression) *BinaryOperator {
	return &BinaryOperator{&expression{}, pos, op, expr1, expr2}
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

// Operator returns the operator type of the expression.
func (n *BinaryOperator) Operator() OperatorType {
	return n.Op
}

// Precedence returns a number that represents the precedence of the
// expression.
func (n *BinaryOperator) Precedence() int {
	switch n.Op {
	case OperatorMultiplication, OperatorDivision, OperatorModulo,
		OperatorLeftShift, OperatorRightShift, OperatorBitAnd, OperatorAndNot:
		return 5
	case OperatorAddition, OperatorSubtraction, OperatorBitOr, OperatorXor:
		return 4
	case OperatorEqual, OperatorNotEqual, OperatorLess, OperatorLessEqual,
		OperatorGreater, OperatorGreaterEqual:
		return 3
	case OperatorAnd, OperatorRelaxedAnd:
		return 2
	case OperatorOr, OperatorRelaxedOr:
		return 1
	}
	panic("invalid operator type")
}

// StructType node represents a struct type.
type StructType struct {
	*expression
	*Position
	Fields []*Field
}

// NewStructType returns a new StructType node.
func NewStructType(pos *Position, fields []*Field) *StructType {
	return &StructType{&expression{}, pos, fields}
}

func (n *StructType) String() string {
	s := "struct { "
	for i, fd := range n.Fields {
		s += fd.String()
		if i != len(n.Fields)-1 {
			s += "; "
		}
	}
	s += " }"
	return s
}

// Field represents a field declaration in a struct type. A field
// declaration can be explicit (having an identifier list and a type) or
// implicit (having a type only).
type Field struct {
	Idents []*Identifier // identifiers. If nil is an embedded field.
	Type   Expression
	Tag    *string
}

// NewField returns a new NewField node.
func NewField(idents []*Identifier, typ Expression, tag *string) *Field {
	return &Field{idents, typ, tag}
}

func (n *Field) String() string {
	s := ""
	for i, ident := range n.Idents {
		s += ident.String()
		if i != len(n.Idents)-1 {
			s += ","
		}
		s += " "
	}
	s += n.Type.String()
	if n.Tag != nil {
		s += " `" + *n.Tag + "`"
	}
	return s
}

// SliceType node represents a slice type.
type SliceType struct {
	*expression
	*Position              // position in the source.
	ElementType Expression // element type.
}

func NewSliceType(pos *Position, elementType Expression) *SliceType {
	return &SliceType{&expression{}, pos, elementType}
}

func (n *SliceType) String() string {
	return "[]" + n.ElementType.String()
}

// ArrayType node represents an array type.
type ArrayType struct {
	*expression
	*Position              // position in the source.
	Len         Expression // length. It is nil for arrays specified with ... notation.
	ElementType Expression // element type.
}

func NewArrayType(pos *Position, len Expression, elementType Expression) *ArrayType {
	return &ArrayType{&expression{}, pos, len, elementType}
}

func (n *ArrayType) String() string {
	s := "["
	if n.Len == nil {
		s += "..."
	} else {
		s += n.Len.String()
	}
	s += "]" + n.ElementType.String()
	return s
}

// CompositeLiteral node represent a composite literal.
type CompositeLiteral struct {
	*expression
	*Position            // position in the source.
	Type      Expression // type of the composite literal. nil for composite literals without type.
	KeyValues []KeyValue // nil for empty composite literals.
}

func NewCompositeLiteral(pos *Position, typ Expression, keyValues []KeyValue) *CompositeLiteral {
	return &CompositeLiteral{&expression{}, pos, typ, keyValues}
}

func (n *CompositeLiteral) String() string {
	if expandedPrint {
		s := n.Type.String()
		s += "{"
		for i, kv := range n.KeyValues {
			if i > 0 {
				s += ", "
			}
			s += kv.String()
		}
		s += "}"
		return s
	}
	return n.Type.String() + " literal"
}

// KeyValue represents a key value pair in a slice, map or struct composite literal.
type KeyValue struct {
	Key   Expression // nil for not-indexed values.
	Value Expression
}

func (kv KeyValue) String() string {
	if kv.Key == nil {
		return kv.Value.String()
	}
	return kv.Key.String() + ": " + kv.Value.String()
}

// MapType node represents a map type.
type MapType struct {
	*expression
	*Position            // position in the source.
	KeyType   Expression // type of map keys.
	ValueType Expression // type of map values.
}

func NewMapType(pos *Position, keyType, valueType Expression) *MapType {
	return &MapType{&expression{}, pos, keyType, valueType}
}

func (n *MapType) String() string {
	return "map[" + n.KeyType.String() + "]" + n.ValueType.String()
}

// Interface node represents an interface type.
type Interface struct {
	*expression
	*Position // position in the source.
}

func NewInterface(pos *Position) *Interface {
	return &Interface{&expression{}, pos}
}

func (n *Interface) String() string {
	return "interface{}"
}

// Call node represents a function call expression.
type Call struct {
	*expression
	*Position               // position in the source.
	Func       Expression   // function.
	Args       []Expression // arguments.
	IsVariadic bool         // reports whether it is variadic.
}

func NewCall(pos *Position, fun Expression, args []Expression, isVariadic bool) *Call {
	return &Call{&expression{}, pos, fun, args, isVariadic}
}

func (n *Call) String() string {
	s := n.Func.String()
	switch fn := n.Func.(type) {
	case *UnaryOperator:
		if fn.Op == OperatorPointer || fn.Op == OperatorReceive {
			s = "(" + s + ")"
		}
	case *FuncType:
		if len(fn.Result) == 0 {
			s = "(" + s + ")"
		}
	case *ChanType:
		s = "(" + s + ")"
	}
	s += "("
	for i, arg := range n.Args {
		if i > 0 {
			s += ", "
		}
		s += arg.String()
	}
	s += ")"
	return s
}

// Defer node represents a defer statement.
type Defer struct {
	*Position            // position in the source.
	Call      Expression // function or method call (should be a Call node).
}

func NewDefer(pos *Position, call Expression) *Defer {
	return &Defer{pos, call}
}

func (n *Defer) String() string {
	return "defer " + n.Call.String()
}

// Go node represents a go statement.
type Go struct {
	*Position            // position in the source.
	Call      Expression // function or method call (should be a Call node).
}

func NewGo(pos *Position, call Expression) *Go {
	return &Go{pos, call}
}

func (n *Go) String() string {
	return "go " + n.Call.String()
}

// Goto node represents a goto statement.
type Goto struct {
	*Position             // position in the source.
	Label     *Identifier // label.
}

func NewGoto(pos *Position, label *Identifier) *Goto {
	return &Goto{pos, label}
}

func (n *Goto) String() string {
	return "goto " + n.Label.String()
}

// Label node represents a label statement.
type Label struct {
	*Position             // position in the source.
	Ident     *Identifier // identifier.
	Statement Node        // statement.
}

func NewLabel(pos *Position, ident *Identifier, statement Node) *Label {
	return &Label{pos, ident, statement}
}

// Var node represent a variable declaration by keyword "var".
type Var struct {
	*Position               // position in the source.
	Lhs       []*Identifier // left-hand side of assignment.
	Type      Expression    // nil for non-typed variable declarations.
	Rhs       []Expression  // nil for non-initialized variable declarations.
}

func NewVar(pos *Position, lhs []*Identifier, typ Expression, rhs []Expression) *Var {
	return &Var{pos, lhs, typ, rhs}
}

func (n *Var) String() string {
	s := "var "
	for i, ident := range n.Lhs {
		if i > 0 {
			s += " "
		}
		s += ident.Name
	}
	if n.Type != nil {
		s += " " + n.Type.String()
	}
	if len(n.Rhs) > 0 {
		s += " = "
		for i, value := range n.Rhs {
			if i > 0 {
				s += " "
			}
			s += value.String()
		}
	}
	return s
}

// Const node represent a const declaration.
type Const struct {
	*Position               // position in the source.
	Lhs       []*Identifier // left-hand side identifiers.
	Type      Expression    // nil for non-typed constant declarations.
	Rhs       []Expression  // nil for implicit-value constant declarations.
	Index     int           // index of the declaration in the constant declaration group or 0 if not in a group.
}

func NewConst(pos *Position, lhs []*Identifier, typ Expression, rhs []Expression, index int) *Const {
	return &Const{pos, lhs, typ, rhs, index}
}

// Index node represents an index expression.
type Index struct {
	*expression
	*Position            // position in the source.
	Expr      Expression // expression.
	Index     Expression // index.
}

func NewIndex(pos *Position, expr Expression, index Expression) *Index {
	return &Index{&expression{}, pos, expr, index}
}

func (n *Index) String() string {
	return n.Expr.String() + "[" + n.Index.String() + "]"
}

// Slicing node represents a slicing expression.
type Slicing struct {
	*expression
	*Position            // position in the source.
	Expr      Expression // expression.
	Low       Expression // low bound.
	High      Expression // high bound.
	Max       Expression // max bound.
	IsFull    bool       // reports whether is a full expression.
}

func NewSlicing(pos *Position, expr, low, high Expression, max Expression, isFull bool) *Slicing {
	return &Slicing{&expression{}, pos, expr, low, high, max, isFull}
}

func (n *Slicing) String() string {
	s := n.Expr.String() + "["
	if n.Low != nil {
		s += n.Low.String()
	}
	s += ":"
	if n.High != nil {
		s += n.High.String()
	}
	if n.Max != nil {
		s += ":"
		s += n.Max.String()
	}
	s += "]"
	return s
}

// ChanType node represents a chan type.
type ChanType struct {
	*expression
	*Position                 // position in the source.
	Direction   ChanDirection // direction.
	ElementType Expression    // type of chan elements.
}

func NewChanType(pos *Position, direction ChanDirection, elementType Expression) *ChanType {
	return &ChanType{&expression{}, pos, direction, elementType}
}

func (n *ChanType) String() string {
	var s string
	if n.Direction == ReceiveDirection {
		s = "<-"
	}
	s += "chan"
	if n.Direction == SendDirection {
		s += "<-"
	}
	return s + " " + n.ElementType.String()
}

// Selector node represents a selector expression.
type Selector struct {
	*expression
	*Position            // position in the source.
	Expr      Expression // expression.
	Ident     string     // identifier.
}

func NewSelector(pos *Position, expr Expression, ident string) *Selector {
	return &Selector{&expression{}, pos, expr, ident}
}

func (n *Selector) String() string {
	return n.Expr.String() + "." + n.Ident
}

// TypeAssertion node represents a type assertion expression.
type TypeAssertion struct {
	*expression
	*Position            // position in the source.
	Expr      Expression // expression.
	Type      Expression // type, is nil if it is a type switch assertion ".(type)".
}

func NewTypeAssertion(pos *Position, expr Expression, typ Expression) *TypeAssertion {
	return &TypeAssertion{&expression{}, pos, expr, typ}
}

func (n *TypeAssertion) String() string {
	if n.Type == nil {
		return n.Expr.String() + ".(type)"
	}
	return n.Expr.String() + ".(" + n.Type.String() + ")"
}

// Placeholder node represent a special placeholder node.
type Placeholder struct {
	*expression
	*Position // position in the source.
}

func NewPlaceholder() *Placeholder {
	return &Placeholder{&expression{}, nil}
}

func (n *Placeholder) String() string {
	return "[Placeholder]"
}

// Send node represents a send statement.
type Send struct {
	*Position            // position in the source.
	Channel   Expression // channel.
	Value     Expression // value to send on the channel.
}

func NewSend(pos *Position, channel Expression, value Expression) *Send {
	return &Send{pos, channel, value}
}

func (n *Send) String() string {
	return n.Channel.String() + " <- " + n.Value.String()
}
