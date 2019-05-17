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
	"math/big"
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
	OperatorLessOrEqual                        // <=
	OperatorGreater                            // >
	OperatorGreaterOrEqual                     // >=
	OperatorNot                                // !
	OperatorAnd                                // &
	OperatorOr                                 // |
	OperatorAndAnd                             // &&
	OperatorOrOr                               // ||
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
)

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
	AssignmentIncrement                            // --
	AssignmentDecrement                            // --
)

func (op OperatorType) String() string {
	return []string{"==", "!=", "<", "<=", ">", ">=", "!", "&", "|", "&&", "||",
		"+", "-", "*", "/", "%", "^", "&^", "<<", ">>", "<-"}[op]
}

// Context indicates the context in which a value statement must be valuated.
type Context int

const (
	ContextNone Context = iota
	ContextText
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
	case ContextNone:
		return "None"
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
	isexpr()
	Node
	String() string
}

type expression struct{}

func (e expression) isexpr() {}

// Tree node represents a tree.
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

func (t Text) String() string {
	return string(t.Text)
}

// URL node represents an URL in an attribute value. Show nodes that are
// children of an URL node are rendered accordingly.
type URL struct {
	*Position        // position in the source.
	Tag       string // tag (in lowercase).
	Attribute string // attribute (in lowercase).
	Value     []Node // value nodes.
}

func NewURL(pos *Position, tag, attribute string, value []Node) *URL {
	return &URL{pos, tag, attribute, value}
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
	Variables []Expression   // left hand variables.
	Type      AssignmentType // type.
	Values    []Expression   // assigned values (nil for increment and decrement).
}

func NewAssignment(pos *Position, variables []Expression, typ AssignmentType, values []Expression) *Assignment {
	return &Assignment{pos, variables, typ, values}
}

func (a *Assignment) String() string {
	var s string
	for i, v := range a.Variables {
		if i > 0 {
			s += ", "
		}
		s += v.String()
	}
	switch a.Type {
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
	if a.Values != nil {
		for i, value := range a.Values {
			if i > 0 {
				s += ", "
			}
			s += value.String()
		}
	}
	return s
}

// Field node represents a field in a function type, literal or declaration.
type Field struct {
	Ident *Identifier // name, can be nil.
	Type  Expression  // type.
}

func NewField(ident *Identifier, typ Expression) *Field {
	return &Field{ident, typ}
}

func (n *Field) String() string {
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
	Parameters []*Field     // parameters.
	Result     []*Field     // result.
	IsVariadic bool         // indicates if it is variadic.
	Reflect    reflect.Type // reflect type.
}

func NewFuncType(pos *Position, parameters []*Field, result []*Field, isVariadic bool) *FuncType {
	return &FuncType{expression{}, pos, parameters, result, isVariadic, nil}
}

func (n *FuncType) String() string {
	s := "func("
	for i, field := range n.Parameters {
		if i > 0 {
			s += ", "
		}
		s += field.String()
	}
	s += ")"
	if len(n.Result) > 0 {
		if n.Result[0].Ident == nil {
			s += " " + n.Result[0].Type.String()
		} else {
			s += " ("
			for i, field := range n.Result {
				if i > 0 {
					s += ", "
				}
				s += field.String()
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
	Ident    *Identifier // name, nil for function literals.
	Type     *FuncType   // type.
	Body     *Block      // body.
	Upvalues []string    // upvalues. // TODO (Gianluca): deprecated.
	Upvars   []Upvar     // Upvars of func.
}

// Upvar represents a variable defined outside function body.
type Upvar struct {
	Declaration Node
	Index       int16
}

func NewFunc(pos *Position, name *Identifier, typ *FuncType, body *Block) *Func {
	return &Func{expression{}, pos, name, typ, body, nil, nil}
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
	*Position             // position in the source.
	Init      *Assignment // initialization statement.
	Condition Expression  // condition expression.
	Post      *Assignment // post iteration statement.
	Body      []Node      // nodes of the body.
}

func NewFor(pos *Position, init *Assignment, condition Expression, post *Assignment, body []Node) *For {
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
	*Position              // position in the source.
	Assignment *Assignment // assignment.
	Condition  Expression  // condition that once evaluated returns true or false.
	Then       *Block      // nodes to run if the expression is evaluated to true.
	Else       Node        // nodes to run if the expression is evaluated to false. Can be Block or If.
}

func NewIf(pos *Position, assignment *Assignment, cond Expression, then *Block, els Node) *If {
	if then == nil {
		then = NewBlock(nil, []Node{})
	}
	return &If{pos, assignment, cond, then, els}
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
	Fallthrough bool
}

// NewCase returns a new Case node.
func NewCase(pos *Position, expressions []Expression, body []Node, fallThrough bool) *Case {
	return &Case{pos, expressions, body, fallThrough}
}

// TypeDeclaration node represents a type declaration, that is an alias
// declaration or a type definition.
type TypeDeclaration struct {
	*Position                      // position in the source.
	Identifier         *Identifier // identifier of the type.
	Type               Expression  // expression representing the type.
	IsAliasDeclaration bool        // indicates if it is an alias declaration or a type definition.
}

func (td *TypeDeclaration) String() string {
	if td.IsAliasDeclaration {
		return fmt.Sprintf("type %s = %s", td.Identifier.Name, td.Type.String())
	}
	return fmt.Sprintf("type %s %s", td.Identifier.Name, td.Type.String())
}

// NewTypeDeclaration returns a new TypeDeclaration node.
func NewTypeDeclaration(pos *Position, identifier *Identifier, typ Expression, isAliasDeclaration bool) *TypeDeclaration {
	return &TypeDeclaration{pos, identifier, typ, isAliasDeclaration}
}

// Macro node represents a statement {% macro ... %}.
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

// ShowMacro node represents a statement {% show <macro> %}.
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

// Include node represents a statement {% include <path> %}.
type Include struct {
	*Position         // position in the source.
	Path      string  // path of the source to include.
	Context   Context // context.
	Tree      *Tree   // expanded tree of <path>.
}

func NewInclude(pos *Position, path string, ctx Context) *Include {
	return &Include{Position: pos, Path: path, Context: ctx}
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

func (v Show) String() string {
	return fmt.Sprintf("{{ %v }}", v.Expr)
}

// Extends node represents a statement {% extends ... %}.
type Extends struct {
	*Position         // position in the source.
	Path      string  // path to the file to extend.
	Context   Context // context.
	Tree      *Tree   // expanded tree of extends.
}

func NewExtends(pos *Position, path string, ctx Context) *Extends {
	return &Extends{Position: pos, Path: path, Context: ctx}
}

func (e Extends) String() string {
	return fmt.Sprintf("{%% extends %v %%}", strconv.Quote(e.Path))
}

// Import node represents a statement {% import ... %}.
type Import struct {
	*Position             // position in the source.
	Ident     *Identifier // name (including "." and "_") or nil.
	Path      string      // path of the imported file.
	Context   Context     // context.
	Tree      *Tree       // expanded tree of import.
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

// Comment node represents a statement {# ... #}.
type Comment struct {
	*Position        // position in the source.
	Text      string // comment text.
}

func NewComment(pos *Position, text string) *Comment {
	return &Comment{pos, text}
}

// Parenthesis node represents a parenthesized expression.
type Parenthesis struct {
	expression
	*Position            // position in the source.
	Expr      Expression // expression.
}

func NewParenthesis(pos *Position, expr Expression) *Parenthesis {
	return &Parenthesis{expression{}, pos, expr}
}

func (n *Parenthesis) String() string {
	return "(" + n.Expr.String() + ")"
}

// Rune node represents a rune constant.
type Rune struct {
	expression
	*Position      // position in the source.
	Value     rune // value.
}

func NewRune(pos *Position, value rune) *Rune {
	return &Rune{expression{}, pos, value}
}

func (n *Rune) Rune() string {
	return strconv.QuoteRuneToASCII(n.Value)
}

// Int node represents an integer constant.
type Int struct {
	expression
	*Position         // position in the source.
	Value     big.Int // value.
}

func NewInt(pos *Position, value *big.Int) *Int {
	return &Int{expression{}, pos, *value}
}

func (n *Int) String() string {
	return n.Value.String()
}

// Float node represents a float constant.
type Float struct {
	expression
	*Position           // position in the source.
	Value     big.Float // value.
}

func NewFloat(pos *Position, value *big.Float) *Float {
	return &Float{expression{}, pos, *value}
}

func (n *Float) String() string {
	return n.Value.String()
}

// String node represents a string expression, a sequence of UTF8 encoded
// characters.
type String struct {
	expression
	*Position        // position in the source.
	Text      string // text.
}

func NewString(pos *Position, text string) *String {
	return &String{expression{}, pos, text}
}

func (n *String) String() string {
	return strconv.Quote(n.Text)
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

// Operator returns the operator type of the expression.
func (n *BinaryOperator) Operator() OperatorType {
	return n.Op
}

// Precedence returns a number that represents the precedence of the
// expression.
func (n *BinaryOperator) Precedence() int {
	switch n.Op {
	case OperatorMultiplication, OperatorDivision, OperatorModulo,
		OperatorLeftShift, OperatorRightShift, OperatorAnd, OperatorAndNot:
		return 5
	case OperatorAddition, OperatorSubtraction, OperatorOr, OperatorXor:
		return 4
	case OperatorEqual, OperatorNotEqual, OperatorLess, OperatorLessOrEqual,
		OperatorGreater, OperatorGreaterOrEqual:
		return 3
	case OperatorAndAnd:
		return 2
	case OperatorOrOr:
		return 1
	}
	panic("invalid operator type")
}

// StructType node represents a struct type.
type StructType struct {
	expression
	*Position
	FieldDecl []*FieldDecl
}

// NewStructType returns a new StructType node.
func NewStructType(pos *Position, fieldDecl []*FieldDecl) *StructType {
	return &StructType{expression{}, pos, fieldDecl}
}

func (st *StructType) String() string {
	s := "struct { "
	for i, fd := range st.FieldDecl {
		s += fd.String()
		if i != len(st.FieldDecl)-1 {
			s += "; "
		}
	}
	s += " }"
	return s
}

// FieldDecl represents a field declaration in a struct type. A field
// declaration can be explicit (having an identifier list and a type) or
// implicit (having a type only).
type FieldDecl struct {
	IdentifierList []*Identifier // if nil is an embedded field.
	Type           Expression
	Tag            *string
}

// NewFieldDecl returns a new NewFieldDecl node.
func NewFieldDecl(identifierList []*Identifier, typ Expression, tag *string) *FieldDecl {
	return &FieldDecl{identifierList, typ, tag}
}

func (fd *FieldDecl) String() string {
	s := ""
	for i, ident := range fd.IdentifierList {
		s += ident.String()
		if i != len(fd.IdentifierList)-1 {
			s += ","
		}
		s += " "
	}
	s += fd.Type.String()
	if fd.Tag != nil {
		s += " `" + *fd.Tag + "`"
	}
	return s
}

// SliceType node represents a slice type.
type SliceType struct {
	expression
	*Position              // position in the source.
	ElementType Expression // element type.
}

func NewSliceType(pos *Position, elementType Expression) *SliceType {
	return &SliceType{expression{}, pos, elementType}
}

func (s *SliceType) String() string {
	return "[]" + s.ElementType.String()
}

// ArrayType node represents an array type.
type ArrayType struct {
	expression
	*Position              // position in the source.
	Len         Expression // length. It is nil for arrays specified with ... notation.
	ElementType Expression // element type.
}

func NewArrayType(pos *Position, len Expression, elementType Expression) *ArrayType {
	return &ArrayType{expression{}, pos, len, elementType}
}

func (a *ArrayType) String() string {
	s := "["
	if a.Len == nil {
		s += "..."
	} else {
		s += a.Len.String()
	}
	s += "]" + a.ElementType.String()
	return s
}

// CompositeLiteral node represent a composite literal.
type CompositeLiteral struct {
	expression
	*Position            // position in the source.
	Type      Expression // type of the composite literal. nil for composite literals without type.
	KeyValues []KeyValue // nil for empty composite literals.
}

func NewCompositeLiteral(pos *Position, typ Expression, keyValues []KeyValue) *CompositeLiteral {
	return &CompositeLiteral{expression{}, pos, typ, keyValues}
}

func (t *CompositeLiteral) String() string {
	if expandedPrint {
		s := t.Type.String()
		s += "{"
		for i, kv := range t.KeyValues {
			if i > 0 {
				s += ", "
			}
			s += kv.String()
		}
		s += "}"
		return s
	}
	return t.Type.String() + " literal"
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
	expression
	*Position            // position in the source.
	KeyType   Expression // type of map keys.
	ValueType Expression // type of map values.
}

func NewMapType(pos *Position, keyType, valueType Expression) *MapType {
	return &MapType{expression{}, pos, keyType, valueType}
}

func (m *MapType) String() string {
	return "map[" + m.KeyType.String() + "]" + m.ValueType.String()
}

// Call node represents a function call expression.
type Call struct {
	expression
	*Position               // position in the source.
	Func       Expression   // function.
	Args       []Expression // arguments.
	IsVariadic bool         // indicates if it is variadic.
}

func NewCall(pos *Position, fun Expression, args []Expression, isVariadic bool) *Call {
	return &Call{expression{}, pos, fun, args, isVariadic}
}

func (n *Call) String() string {
	s := n.Func.String() + "("
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
	*Position       // position in the source.
	Call      *Call // function or method call.
}

func NewDefer(pos *Position, call *Call) *Defer {
	return &Defer{pos, call}
}

func (n *Defer) String() string {
	return "defer " + n.Call.String()
}

// Go node represents a go statement.
type Go struct {
	*Position       // position in the source.
	Call      *Call // function or method call.
}

func NewGo(pos *Position, call *Call) *Go {
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
	Name      *Identifier // name.
	Statement Node        // statement.
}

func NewLabel(pos *Position, name *Identifier, statement Node) *Label {
	return &Label{pos, name, statement}
}

// Var node represent a variable declaration by keyword "var".
type Var struct {
	*Position                 // position in the source.
	Identifiers []*Identifier // variables.
	Type        Expression    // nil for non-typed variable declarations.
	Values      []Expression  // nil for non-initialized variable declarations.
}

func NewVar(pos *Position, variables []*Identifier, typ Expression, values []Expression) *Var {
	return &Var{pos, variables, typ, values}
}

func (n *Var) String() string {
	s := "var "
	for i, ident := range n.Identifiers {
		if i > 0 {
			s += " "
		}
		s += ident.Name
	}
	if n.Type != nil {
		s += " " + n.Type.String()
	}
	s += " = "
	for i, value := range n.Values {
		if i > 0 {
			s += " "
		}
		s += value.String()
	}
	return s
}

// Const node represent a const declaration.
type Const struct {
	*Position                 // position in the source.
	Identifiers []*Identifier // identifiers.
	Type        Expression    // nil for non-typed constant declarations.
	Values      []Expression  // nil for implicit-value constant declarations.
}

func NewConst(pos *Position, identifiers []*Identifier, typ Expression, values []Expression) *Const {
	return &Const{pos, identifiers, typ, values}
}

// Index node represents an index expression.
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

// Slicing node represents a slicing expression.
type Slicing struct {
	expression
	*Position            // position in the source.
	Expr      Expression // expression.
	Low       Expression // low bound.
	High      Expression // high bound.
}

func NewSlicing(pos *Position, expr, low, high Expression) *Slicing {
	return &Slicing{expression{}, pos, expr, low, high}
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
	s += "]"
	return s
}

// ChanType node represents a chan type.
type ChanType struct {
	expression
	*Position                 // position in the source.
	Direction   ChanDirection // direction.
	ElementType Expression    // type of chan elements.
}

func NewChanType(pos *Position, direction ChanDirection, elementType Expression) *ChanType {
	return &ChanType{expression{}, pos, direction, elementType}
}

func (n *ChanType) String() string {
	var s string
	if n.Direction == ReceiveDirection {
		s = "<-"
	}
	s += "chan"
	if n.Direction == SendDirection {
		s = "<-"
	}
	return s + " " + n.ElementType.String()
}

// Selector node represents a selector expression.
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

// TypeAssertion node represents a type assertion expression.
type TypeAssertion struct {
	expression
	*Position            // position in the source.
	Expr      Expression // expression.
	Type      Expression // type, is nil if it is a type switch assertion ".(type)".
}

func NewTypeAssertion(pos *Position, expr Expression, typ Expression) *TypeAssertion {
	return &TypeAssertion{expression{}, pos, expr, typ}
}

func (n *TypeAssertion) String() string {
	if n.Type == nil {
		return n.Expr.String() + ".(type)"
	}
	return n.Expr.String() + ".(" + n.Type.String() + ")"
}

// Value node represent a special node with an associated value.
type Value struct {
	expression
	*Position             // position in the source.
	Val       interface{} // associated value.
}

func NewValue(val interface{}) *Value {
	return &Value{expression{}, nil, val}
}

func (n *Value) String() string {
	return fmt.Sprintf("%v", n.Val)
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
