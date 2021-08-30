// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ast declares the types used to define program and template trees.
//
// For example, the source in a template file named "articles.html":
//
//    {% for article in articles %}
//    <div>{{ article.Title }}</div>
//    {% end %}
//
// is represented with the tree:
//
//	ast.NewTree("articles.txt", []ast.Node{
//		ast.NewForIn(
//			&ast.Position{Line: 1, Column: 1, Start: 0, End: 69},
//			ast.NewIdentifier(&ast.Position{Line: 1, Column: 8, Start: 7, End: 13}, "article"),
//			ast.NewIdentifier(&ast.Position{Line: 1, Column: 19, Start: 18, End: 25}, "articles"),
//			[]ast.Node{
//				ast.NewText(&ast.Position{Line: 1, Column: 30, Start: 29, End: 34}, []byte("\n<div>"), ast.Cut{1, 0}),
//				ast.NewShow(
//					&ast.Position{Line: 2, Column: 6, Start: 35, End: 53},
//					[]ast.Expression{
//						ast.NewSelector(
//							&ast.Position{Line: 2, Column: 16, Start: 38, End: 50},
//							ast.NewIdentifier(&ast.Position{Line: 2, Column: 9, Start: 38, End: 44}, "article"),
//							"Title"),
//					},
//					ast.ContextHTML),
//				ast.NewText(&ast.Position{Line: 2, Column: 25, Start: 54, End: 60}, []byte("</div>\n"), ast.Cut{}),
//			},
//		),
//	}, ast.FormatHTML)
//
package ast

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// expandedPrint is set to true in tests to print completely a composite
// literal.
var expandedPrint = false

// OperatorType represents an operator type in a unary and binary expression.
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
	OperatorContains                           // contains
	OperatorNotContains                        // not contains
	OperatorReceive                            // <-
	OperatorAddress                            // &
	OperatorPointer                            // *
	OperatorExtendedAnd                        // and
	OperatorExtendedOr                         // or
	OperatorExtendedNot                        // not
)

// String returns the string representation of the operator type.
func (op OperatorType) String() string {
	// The last empty operators are compiler.internalOperatorZero and
	// compiler.internalOperatorNotZero.
	return []string{"==", "!=", "<", "<=", ">", ">=", "!", "&", "|", "&&", "||",
		"+", "-", "*", "/", "%", "^", "&^", "<<", ">>", "contains", "not contains",
		"<-", "&", "*", "and", "or", "not", "", ""}[op]
}

// AssignmentType represents a type of assignment.
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

// A Format represents a content format.
type Format int

const (
	FormatText Format = iota
	FormatHTML
	FormatCSS
	FormatJS
	FormatJSON
	FormatMarkdown
)

// String returns the name of the format.
func (format Format) String() string {
	switch format {
	case FormatText:
		return "text"
	case FormatHTML:
		return "HTML"
	case FormatCSS:
		return "CSS"
	case FormatJS:
		return "JavaScript"
	case FormatJSON:
		return "JSON"
	case FormatMarkdown:
		return "Markdown"
	}
	panic("invalid format")
}

// Context indicates the context in which a show statement is evaluated.
type Context int

const (
	ContextText Context = iota
	ContextHTML
	ContextCSS
	ContextJS
	ContextJSON
	ContextMarkdown
	ContextTag
	ContextQuotedAttr
	ContextUnquotedAttr
	ContextCSSString
	ContextJSString
	ContextJSONString
	ContextTabCodeBlock
	ContextSpacesCodeBlock
)

// String returns the name of the context.
func (ctx Context) String() string {
	switch ctx {
	case ContextText:
		return "text"
	case ContextHTML:
		return "HTML"
	case ContextCSS:
		return "CSS"
	case ContextJS:
		return "JavaScript"
	case ContextJSON:
		return "JSON"
	case ContextMarkdown:
		return "Markdown"
	case ContextTag:
		return "tag"
	case ContextQuotedAttr:
		return "quoted attribute"
	case ContextUnquotedAttr:
		return "unquoted attribute"
	case ContextCSSString:
		return "CSS string"
	case ContextJSString:
		return "JavaScript string"
	case ContextJSONString:
		return "JSON string"
	case ContextTabCodeBlock:
		return "tab code block"
	case ContextSpacesCodeBlock:
		return "spaces code block"
	}
	panic("invalid context")
}

// LiteralType represents the type of a literal.
type LiteralType int

const (
	StringLiteral LiteralType = iota
	RuneLiteral
	IntLiteral
	FloatLiteral
	ImaginaryLiteral
)

// ChanDirection represents the direction of a channel type.
type ChanDirection int

const (
	NoDirection ChanDirection = iota
	ReceiveDirection
	SendDirection
)

var directionString = [3]string{"no direction", "receive", "send"}

// String returns the name of the direction dir or "no direction" if there is
// no direction.
func (dir ChanDirection) String() string { return directionString[dir] }

// Node is a node of the tree.
type Node interface {
	Pos() *Position // position in the original source
}

// Position is a position of a node in the source.
type Position struct {
	Line   int // line starting from 1
	Column int // column in characters starting from 1
	Start  int // index of the first byte
	End    int // index of the last byte
}

// Pos returns the position p.
func (p *Position) Pos() *Position {
	return p
}

// String returns the line and column separated by a colon, for example "37:18".
func (p Position) String() string {
	return strconv.Itoa(p.Line) + ":" + strconv.Itoa(p.Column)
}

// WithEnd returns a copy of the position but with the given end index.
func (p *Position) WithEnd(end int) *Position {
	pp := *p
	pp.End = end
	return &pp
}

// Operator represents an operator expression. It is implemented by the
// UnaryOperator and BinaryOperator nodes.
type Operator interface {
	Expression
	Operator() OperatorType
	Precedence() int
}

// Upvar represents a variable defined outside function body. Even package level
// variables (native or not) are considered upvars.
type Upvar struct {

	// NativePkg is the name of the native package which holds a native Upvar.
	// If Upvar is not a native Upvar then NativeName is an empty string.
	NativePkg string

	// NativeName is the name of the native declaration of a native Upvar. If
	// Upvar is not a native Upvar then NativeName is an empty string.
	NativeName string

	// NativeValue is the value of the native variable Upvar. If Upvar is not
	// native then Upvar is nil.
	NativeValue *reflect.Value

	// NativeValueType, in case of Upvar refers to a native variable, contains
	// the type of such variable. If not then NativeValueType is nil.
	//
	// NativeValueType is necessary because the type cannot be stored into
	// the NativeValue, as if the upvar is not initialized in the compiler
	// then NativeValue contains an invalid reflect.Value.
	NativeValueType reflect.Type

	// Declaration is the ast node where Upvar is defined. If Upvar is a
	// native var then Declaration is nil.
	Declaration Node
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

// Cut indicates, in a Text node, how many bytes should be cut from the left
// and the right of the text before rendering the Text node.
type Cut struct {
	Left  int
	Right int
}

// ArrayType node represents an array type.
type ArrayType struct {
	*expression
	*Position              // position in the source.
	Len         Expression // length. It is nil for arrays specified with ... notation.
	ElementType Expression // element type.
}

// NewArrayType returns a new ArrayType node.
func NewArrayType(pos *Position, len Expression, elementType Expression) *ArrayType {
	return &ArrayType{&expression{}, pos, len, elementType}
}

// String returns the string representation of n.
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

// Assignment node represents an assignment statement.
type Assignment struct {
	*Position                // position in the source.
	Lhs       []Expression   // left-hand variables.
	Type      AssignmentType // type.
	Rhs       []Expression   // assigned values (nil for increment and decrement).
}

// NewAssignment returns a new Assignment node.
func NewAssignment(pos *Position, lhs []Expression, typ AssignmentType, rhs []Expression) *Assignment {
	return &Assignment{pos, lhs, typ, rhs}
}

// String returns the string representation of n.
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

// BasicLiteral represents integer, floating-point, imaginary, rune and
// string literals.
type BasicLiteral struct {
	expression
	*Position             // position in the source.
	Type      LiteralType // type.
	Value     string      // value.
}

// NewBasicLiteral returns a new BasicLiteral node.
func NewBasicLiteral(pos *Position, typ LiteralType, value string) *BasicLiteral {
	return &BasicLiteral{expression{}, pos, typ, value}
}

// String returns the string representation of n.
func (n *BasicLiteral) String() string {
	return n.Value
}

// BinaryOperator node represents a binary operator expression.
type BinaryOperator struct {
	*expression
	*Position              // position in the source.
	Op        OperatorType // operator.
	Expr1     Expression   // first expression.
	Expr2     Expression   // second expression.
}

// NewBinaryOperator returns a new binary operator.
func NewBinaryOperator(pos *Position, op OperatorType, expr1, expr2 Expression) *BinaryOperator {
	return &BinaryOperator{&expression{}, pos, op, expr1, expr2}
}

// String returns the string representation of n.
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
		OperatorGreater, OperatorGreaterEqual, OperatorContains, OperatorNotContains:
		return 3
	case OperatorAnd, OperatorExtendedAnd:
		return 2
	case OperatorOr, OperatorExtendedOr:
		return 1
	}
	panic("invalid operator type")
}

// Block node represents a block with his own scope.
type Block struct {
	*Position
	Nodes []Node
}

// NewBlock returns a new block statement.
func NewBlock(pos *Position, nodes []Node) *Block {
	return &Block{pos, nodes}
}

// Break node represents a "break" statement.
type Break struct {
	*Position             // position in the source.
	Label     *Identifier // label.
}

// NewBreak returns a new Break node.
func NewBreak(pos *Position, label *Identifier) *Break {
	return &Break{pos, label}
}

// Call node represents a function call expression.
type Call struct {
	*expression
	*Position               // position in the source.
	Func       Expression   // function.
	Args       []Expression // arguments.
	IsVariadic bool         // reports whether it is variadic.

	IR struct {
		// AppendArg1, in transformed calls to the builtin function 'append',
		// is the argument with index 1.
		AppendArg1 *Call
	}
}

func NewCall(pos *Position, fun Expression, args []Expression, isVariadic bool) *Call {
	return &Call{expression: &expression{}, Position: pos, Func: fun, Args: args, IsVariadic: isVariadic}
}

// String returns the string representation of n.
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
	if n.IsVariadic {
		s += "..."
	}
	s += ")"
	return s
}

// Case node represents "case" and "default" statements.
type Case struct {
	*Position
	Expressions []Expression
	Body        []Node
}

// NewCase returns a new Case node.
func NewCase(pos *Position, expressions []Expression, body []Node) *Case {
	return &Case{pos, expressions, body}
}

// ChanType node represents a chan type.
type ChanType struct {
	*expression
	*Position                 // position in the source.
	Direction   ChanDirection // direction.
	ElementType Expression    // type of chan elements.
}

// NewChanType returns a new ChanType node.
func NewChanType(pos *Position, direction ChanDirection, elementType Expression) *ChanType {
	return &ChanType{&expression{}, pos, direction, elementType}
}

// String returns the string representation of n.
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

// Comment node represents a comment statement in the form {# ... #}.
type Comment struct {
	*Position        // position in the source.
	Text      string // comment text.
}

// NewComment returns a new Comment node.
func NewComment(pos *Position, text string) *Comment {
	return &Comment{pos, text}
}

// CompositeLiteral node represents a composite literal.
type CompositeLiteral struct {
	*expression
	*Position            // position in the source.
	Type      Expression // type of the composite literal. nil for composite literals without type.
	KeyValues []KeyValue // nil for empty composite literals.
}

// NewCompositeLiteral returns a new CompositeLiteral node.
func NewCompositeLiteral(pos *Position, typ Expression, keyValues []KeyValue) *CompositeLiteral {
	return &CompositeLiteral{&expression{}, pos, typ, keyValues}
}

// String returns the string representation of n.
func (n *CompositeLiteral) String() string {
	s := n.Type.String()
	if expandedPrint {
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
	if len(n.KeyValues) > 0 {
		return s + "{...}"
	}
	return s + "{}"
}

// Const node represents a "const" declaration.
type Const struct {
	*Position               // position in the source.
	Lhs       []*Identifier // left-hand side identifiers.
	Type      Expression    // nil for non-typed constant declarations.
	Rhs       []Expression  // nil for implicit-value constant declarations.
	Index     int           // index of the declaration in the constant declaration group or 0 if not in a group.
}

// NewConst returns a new Const node.
func NewConst(pos *Position, lhs []*Identifier, typ Expression, rhs []Expression, index int) *Const {
	return &Const{pos, lhs, typ, rhs, index}
}

// Continue node represents a "continue" statement.
type Continue struct {
	*Position             // position in the source.
	Label     *Identifier // label.
}

// NewContinue returns a new Continue node.
func NewContinue(pos *Position, label *Identifier) *Continue {
	return &Continue{pos, label}
}

// Default node represents a default expression.
type Default struct {
	expression
	*Position            // position in the source.
	Expr1     Expression // left hand expression.
	Expr2     Expression // right hand expression.
}

// NewDefault returns a new Defualt node.
func NewDefault(pos *Position, expr1, expr2 Expression) *Default {
	return &Default{Position: pos, Expr1: expr1, Expr2: expr2}
}

// String returns the string representation of n.
func (n *Default) String() string {
	return n.Expr1.String() + " default " + n.Expr2.String()
}

// Defer node represents a "defer" statement.
type Defer struct {
	*Position            // position in the source.
	Call      Expression // function or method call (should be a Call node).
}

// NewDefer returns a new Defer node.
func NewDefer(pos *Position, call Expression) *Defer {
	return &Defer{pos, call}
}

// String returns the string representation of n.
func (n *Defer) String() string {
	return "defer " + n.Call.String()
}

// DollarIdentifier node represents a dollar identifier in the form  $id.
type DollarIdentifier struct {
	*expression
	*Position             // position in the source.
	Ident     *Identifier // identifier.

	IR struct {
		Ident Expression
	}
}

// NewDollarIdentifier returns a new DollarIdentifier node.
func NewDollarIdentifier(pos *Position, ident *Identifier) *DollarIdentifier {
	return &DollarIdentifier{&expression{}, pos, ident, struct{ Ident Expression }{}}
}

// String returns the string representation of n.
func (n *DollarIdentifier) String() string {
	return "$" + n.Ident.String()
}

// Extends node represents an "extends" statement.
type Extends struct {
	*Position        // position in the source.
	Path      string // path to extend.
	Format    Format // format.
	Tree      *Tree  // expanded tree of extends.
}

// NewExtends returns a new Extends node.
func NewExtends(pos *Position, path string, format Format) *Extends {
	return &Extends{Position: pos, Path: path, Format: format}
}

// String returns the string representation of n.
func (n *Extends) String() string {
	return fmt.Sprintf("extends %v", strconv.Quote(n.Path))
}

// Fallthrough node represents a "fallthrough" statement.
type Fallthrough struct {
	*Position
}

// NewFallthrough returns a new Fallthrough node.
func NewFallthrough(pos *Position) *Fallthrough {
	return &Fallthrough{pos}
}

// Field represents a field declaration in a struct type. A field
// declaration can be explicit (having an identifier list and a type) or
// implicit (having a type only).
type Field struct {
	Idents []*Identifier // identifiers. If nil is an embedded field.
	Type   Expression
	Tag    string
}

// NewField returns a new Field node.
func NewField(idents []*Identifier, typ Expression, tag string) *Field {
	return &Field{idents, typ, tag}
}

// String returns the string representation of n.
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
	if n.Tag != "" {
		s += " `" + n.Tag + "`"
	}
	return s
}

// For node represents a "for" statement.
type For struct {
	*Position            // position in the source.
	Init      Node       // initialization statement.
	Condition Expression // condition expression.
	Post      Node       // post iteration statement.
	Body      []Node     // nodes of the body.
}

// NewFor returns a new For node.
func NewFor(pos *Position, init Node, condition Expression, post Node, body []Node) *For {
	if body == nil {
		body = []Node{}
	}
	return &For{pos, init, condition, post, body}
}

// ForIn node represents a "for in" statement.
type ForIn struct {
	*Position             // position in the source.
	Ident     *Identifier // identifier.
	Expr      Expression  // range expression.
	Body      []Node      // nodes of the body.
}

// NewForIn represents a new ForIn node.
func NewForIn(pos *Position, ident *Identifier, expr Expression, body []Node) *ForIn {
	if body == nil {
		body = []Node{}
	}
	return &ForIn{pos, ident, expr, body}
}

// ForRange node represents the "for range" statement.
type ForRange struct {
	*Position              // position in the source.
	Assignment *Assignment // assignment.
	Body       []Node      // nodes of the body.
}

// NewForRange returns a new ForRange node.
func NewForRange(pos *Position, assignment *Assignment, body []Node) *ForRange {
	if body == nil {
		body = []Node{}
	}
	return &ForRange{pos, assignment, body}
}

// Func node represents a function declaration or literal.
type Func struct {
	expression
	*Position
	Ident   *Identifier // name, nil for function literals.
	Type    *FuncType   // type.
	Body    *Block      // body.
	Endless bool        // reports whether it is endless.
	Upvars  []Upvar     // Upvars of func.
	Format  Format      // macro format.
}

// NewFunc returns a new Func node.
func NewFunc(pos *Position, name *Identifier, typ *FuncType, body *Block, endless bool, format Format) *Func {
	return &Func{expression{}, pos, name, typ, body, endless, nil, format}
}

// String returns the string representation of n.
func (n *Func) String() string {
	if n.Type.Macro {
		return "macro declaration"
	}
	if n.Ident == nil {
		return "func literal"
	}
	return "func declaration"
}

// FuncType node represents a function type.
type FuncType struct {
	expression
	*Position               // position in the source.
	Macro      bool         // indicates whether it is declared as macro.
	Parameters []*Parameter // parameters.
	Result     []*Parameter // result.
	IsVariadic bool         // reports whether it is variadic.
	Reflect    reflect.Type // reflect type.
}

// NewFuncType returns a new FuncType node.
func NewFuncType(pos *Position, macro bool, parameters []*Parameter, result []*Parameter, isVariadic bool) *FuncType {
	return &FuncType{expression{}, pos, macro, parameters, result, isVariadic, nil}
}

// String returns the string representation of n.
func (n *FuncType) String() string {
	s := "func("
	if n.Macro {
		s = "macro("
	}
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

// Go node represents a "go" statement.
type Go struct {
	*Position            // position in the source.
	Call      Expression // function or method call (should be a Call node).
}

// NewGo returns a new Go node.
func NewGo(pos *Position, call Expression) *Go {
	return &Go{pos, call}
}

// String returns the string representation of n.
func (n *Go) String() string {
	return "go " + n.Call.String()
}

// Goto node represents a "goto" statement.
type Goto struct {
	*Position             // position in the source.
	Label     *Identifier // label.
}

// NewGoto returns a new Goto node.
func NewGoto(pos *Position, label *Identifier) *Goto {
	return &Goto{pos, label}
}

// String returns the string representation of n.
func (n *Goto) String() string {
	return "goto " + n.Label.String()
}

// Identifier node represents an identifier expression.
type Identifier struct {
	expression
	*Position        // position in the source.
	Name      string // name.
}

// NewIdentifier returns a new Identifier node.
func NewIdentifier(pos *Position, name string) *Identifier {
	return &Identifier{expression{}, pos, name}
}

// String returns the string representation of n.
func (n *Identifier) String() string {
	if strings.HasPrefix(n.Name, "$itea") {
		return "itea"
	}
	return n.Name
}

// If node represents an "if" statement.
type If struct {
	*Position            // position in the source.
	Init      Node       // init simple statement.
	Condition Expression // condition that once evaluated returns true or false.
	Then      *Block     // nodes to run if the expression is evaluated to true.
	Else      Node       // nodes to run if the expression is evaluated to false. Can be Block or If.
}

// NewIf returns a new If node.
func NewIf(pos *Position, init Node, cond Expression, then *Block, els Node) *If {
	if then == nil {
		then = NewBlock(nil, []Node{})
	}
	return &If{pos, init, cond, then, els}
}

// Import node represents a "import" statement.
type Import struct {
	*Position               // position in the source.
	Ident     *Identifier   // name (including "." and "_") or nil.
	Path      string        // path to import.
	For       []*Identifier // exported identifiers that are enable for access.
	Tree      *Tree         // expanded tree of import.
}

// NewImport returns a new Import node.
func NewImport(pos *Position, ident *Identifier, path string, forIdents []*Identifier) *Import {
	return &Import{Position: pos, Ident: ident, Path: path, For: forIdents}
}

// String returns the string representation of n.
func (n *Import) String() string {
	var s strings.Builder
	s.WriteString("import ")
	if n.Ident != nil {
		s.WriteString(n.Ident.Name)
		s.WriteString(" ")
	}
	s.WriteString(strconv.Quote(n.Path))
	if n.For != nil {
		s.WriteString("for ")
		for i, ident := range n.For {
			if i > 0 {
				s.WriteString(", ")
			}
			s.WriteString(ident.Name)
		}
	}
	return s.String()
}

// Index node represents an index expression.
type Index struct {
	*expression
	*Position            // position in the source.
	Expr      Expression // expression.
	Index     Expression // index.
}

// NewIndex returns a new Index node.
func NewIndex(pos *Position, expr Expression, index Expression) *Index {
	return &Index{&expression{}, pos, expr, index}
}

// String returns the string representation of n.
func (n *Index) String() string {
	return n.Expr.String() + "[" + n.Index.String() + "]"
}

// Interface node represents an interface type.
type Interface struct {
	*expression
	*Position // position in the source.
}

// NewInterface returns a new Interface node.
func NewInterface(pos *Position) *Interface {
	return &Interface{&expression{}, pos}
}

// String returns the string representation of n.
func (n *Interface) String() string {
	return "interface{}"
}

// KeyValue represents a key value pair in a slice, map or struct composite literal.
type KeyValue struct {
	Key   Expression // nil for not-indexed values.
	Value Expression
}

// String returns the string representation of kv.
func (kv KeyValue) String() string {
	if kv.Key == nil {
		return kv.Value.String()
	}
	return kv.Key.String() + ": " + kv.Value.String()
}

// Label node represents a label statement.
type Label struct {
	*Position             // position in the source.
	Ident     *Identifier // identifier.
	Statement Node        // statement.
}

// NewLabel returns a new Label node.
func NewLabel(pos *Position, ident *Identifier, statement Node) *Label {
	return &Label{pos, ident, statement}
}

// MapType node represents a map type.
type MapType struct {
	*expression
	*Position            // position in the source.
	KeyType   Expression // type of map keys.
	ValueType Expression // type of map values.
}

// NewMapType returns a new MapType node.
func NewMapType(pos *Position, keyType, valueType Expression) *MapType {
	return &MapType{&expression{}, pos, keyType, valueType}
}

// String returns the string representation of n.
func (n *MapType) String() string {
	return "map[" + n.KeyType.String() + "]" + n.ValueType.String()
}

// Package node represents a package.
type Package struct {
	*Position
	Name         string // name.
	Declarations []Node

	IR struct {
		// IteaNameToVarIdents maps the name of the transformed 'itea'
		// identifier to the identifiers on the left side of a 'var'
		// declarations with an 'using' statement at package level.
		//
		// For example a package containing this declaration:
		//
		//    {% var V1, V2 = $itea2, len($itea2) using %} ... {% end using %}
		//
		//  will have a mapping in the form:
		//
		//    "$itea2" => [V1, V2]
		//
		IteaNameToVarIdents map[string][]*Identifier
	}
}

// NewPackage returns a new Package node.
func NewPackage(pos *Position, name string, nodes []Node) *Package {
	return &Package{Position: pos, Name: name, Declarations: nodes}
}

// Parameter node represents a parameter in a function type, literal or
// declaration.
type Parameter struct {
	Ident *Identifier // name, can be nil.
	Type  Expression  // type.
}

// NewParameter returns a new Parameter node.
func NewParameter(ident *Identifier, typ Expression) *Parameter {
	return &Parameter{ident, typ}
}

// String returns the string representation of n.
func (n *Parameter) String() string {
	if n.Ident == nil {
		return n.Type.String()
	}
	if n.Type == nil {
		return n.Ident.Name
	}
	return n.Ident.Name + " " + n.Type.String()
}

// Placeholder node represents a special placeholder node.
type Placeholder struct {
	*expression
	*Position // position in the source.
}

// NewPlaceholder returns a new Placeholder node.
func NewPlaceholder() *Placeholder {
	return &Placeholder{&expression{}, nil}
}

// String returns the string representation of n.
func (n *Placeholder) String() string {
	return "[Placeholder]"
}

// Raw node represents a "raw" statement.
type Raw struct {
	*Position        // position in the source.
	Marker    string // marker.
	Tag       string // tag.
	Text      *Text  // text.
}

// NewRaw returns a new Raw node.
func NewRaw(pos *Position, marker, tag string, text *Text) *Raw {
	return &Raw{pos, marker, tag, text}
}

// Render node represents a "render <path>" expression.
type Render struct {
	expression
	*Position        // position in the source.
	Path      string // path of the file to render.
	Tree      *Tree  // expanded tree of <path>.

	// IR holds the internal representation. The type checker transforms the
	// 'render' expression into a macro call, where the macro body is the
	// rendered file.
	IR struct {
		// Import is the 'import' statement that imports the dummy file
		// declaring the dummy macro.
		Import *Import
		// Call is the call to the dummy macro.
		Call *Call
	}
}

// NewRender returns a new Render node.
func NewRender(pos *Position, path string) *Render {
	return &Render{Position: pos, Path: path}
}

// String returns the string representation of n.
func (n *Render) String() string {
	return "render " + strconv.Quote(n.Path)
}

// Return node represents a "return" statement.
type Return struct {
	*Position
	Values []Expression // return values.
}

// NewReturn returns a new Return node.
func NewReturn(pos *Position, values []Expression) *Return {
	return &Return{pos, values}
}

// Select node represents a "select" statement.
type Select struct {
	*Position
	LeadingText *Text
	Cases       []*SelectCase
}

// NewSelect returns a new Select node.
func NewSelect(pos *Position, leadingText *Text, cases []*SelectCase) *Select {
	return &Select{pos, leadingText, cases}
}

// SelectCase represents a "case" statement in a select.
type SelectCase struct {
	*Position
	Comm Node
	Body []Node
}

// NewSelectCase returns a new SelectCase node.
func NewSelectCase(pos *Position, comm Node, body []Node) *SelectCase {
	return &SelectCase{pos, comm, body}
}

// Selector node represents a selector expression.
type Selector struct {
	*expression
	*Position            // position in the source.
	Expr      Expression // expression.
	Ident     string     // identifier.
}

// NewSelector returns a new Selector node.
func NewSelector(pos *Position, expr Expression, ident string) *Selector {
	return &Selector{&expression{}, pos, expr, ident}
}

// String returns the string representation of n.
func (n *Selector) String() string {
	return n.Expr.String() + "." + n.Ident
}

// Send node represents a "send" statement.
type Send struct {
	*Position            // position in the source.
	Channel   Expression // channel.
	Value     Expression // value to send on the channel.
}

// NewSend returns a new Send node.
func NewSend(pos *Position, channel Expression, value Expression) *Send {
	return &Send{pos, channel, value}
}

// String returns the string representation of n.
func (n *Send) String() string {
	return n.Channel.String() + " <- " + n.Value.String()
}

// Show node represents the "show <expr>" statement and its short syntax
// "{{ ... }}".
type Show struct {
	*Position                // position in the source.
	Expressions []Expression // expressions that once evaluated return the values to show.
	Context     Context      // context.
}

// NewShow returns a new Show node.
func NewShow(pos *Position, expressions []Expression, ctx Context) *Show {
	return &Show{pos, expressions, ctx}
}

// String returns the string representation of n.
func (n *Show) String() string {
	var b strings.Builder
	b.WriteString("show ")
	for i, expr := range n.Expressions {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(expr.String())
	}
	return b.String()
}

// SliceType node represents a slice type.
type SliceType struct {
	*expression
	*Position              // position in the source.
	ElementType Expression // element type.
}

// NewSliceType returns a new SliceType node.
func NewSliceType(pos *Position, elementType Expression) *SliceType {
	return &SliceType{&expression{}, pos, elementType}
}

// String returns the string representation of n.
func (n *SliceType) String() string {
	return "[]" + n.ElementType.String()
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

// NewSlicing returns a new Slicing node.
func NewSlicing(pos *Position, expr, low, high Expression, max Expression, isFull bool) *Slicing {
	return &Slicing{&expression{}, pos, expr, low, high, max, isFull}
}

// String returns the string representation of n.
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

// Statements node represents a "{%% ... %%} statement.
type Statements struct {
	*Position
	Nodes []Node // nodes.
}

// NewStatements returns a new Statements node.
func NewStatements(pos *Position, nodes []Node) *Statements {
	if nodes == nil {
		nodes = []Node{}
	}
	return &Statements{pos, nodes}
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

// String returns the string representation of n.
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

// Switch node represents a "switch" statement.
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

// Text node represents a text in a template source.
type Text struct {
	*Position        // position in the source.
	Text      []byte // text.
	Cut       Cut    // cut.
}

// NewText returns a new Text node.
func NewText(pos *Position, text []byte, cut Cut) *Text {
	return &Text{pos, text, cut}
}

// String returns the string representation of n.
func (n *Text) String() string {
	return string(n.Text)
}

// Tree node represents a tree.
type Tree struct {
	*Position
	Path   string // path of the tree.
	Nodes  []Node // nodes of the first level of the tree.
	Format Format // content format.
}

// NewTree returns a new Tree node.
func NewTree(path string, nodes []Node, format Format) *Tree {
	if nodes == nil {
		nodes = []Node{}
	}
	tree := &Tree{
		Position: &Position{1, 1, 0, 0},
		Path:     path,
		Nodes:    nodes,
		Format:   format,
	}
	return tree
}

// TypeAssertion node represents a type assertion expression.
type TypeAssertion struct {
	*expression
	*Position            // position in the source.
	Expr      Expression // expression.
	Type      Expression // type, is nil if it is a type switch assertion ".(type)".
}

// NewTypeAssertion returns a new TypeAssertion node.
func NewTypeAssertion(pos *Position, expr Expression, typ Expression) *TypeAssertion {
	return &TypeAssertion{&expression{}, pos, expr, typ}
}

// String returns the string representation of n.
func (n *TypeAssertion) String() string {
	if n.Type == nil {
		return n.Expr.String() + ".(type)"
	}
	return n.Expr.String() + ".(" + n.Type.String() + ")"
}

// TypeDeclaration node represents a type declaration, that is an alias
// declaration or a type definition.
type TypeDeclaration struct {
	*Position                      // position in the source.
	Ident              *Identifier // identifier of the type.
	Type               Expression  // expression representing the type.
	IsAliasDeclaration bool        // reports whether it is an alias declaration or a type definition.
}

// NewTypeDeclaration returns a new TypeDeclaration node.
func NewTypeDeclaration(pos *Position, ident *Identifier, typ Expression, isAliasDeclaration bool) *TypeDeclaration {
	return &TypeDeclaration{pos, ident, typ, isAliasDeclaration}
}

// String returns the string representation of n.
func (n *TypeDeclaration) String() string {
	if n.IsAliasDeclaration {
		return fmt.Sprintf("type %s = %s", n.Ident.Name, n.Type.String())
	}
	return fmt.Sprintf("type %s %s", n.Ident.Name, n.Type.String())
}

// TypeSwitch node represents a "switch" statement on types.
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

// UnaryOperator node represents an unary operator expression.
type UnaryOperator struct {
	*expression
	*Position              // position in the source.
	Op        OperatorType // operator.
	Expr      Expression   // expression.
}

// NewUnaryOperator returns a new UnaryOperator node.
func NewUnaryOperator(pos *Position, op OperatorType, expr Expression) *UnaryOperator {
	return &UnaryOperator{&expression{}, pos, op, expr}
}

// String returns the string representation of n.
func (n *UnaryOperator) String() string {
	s := n.Op.String()
	if n.Op == OperatorExtendedNot {
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

// URL node represents a URL in an attribute value or Markdown. "show" nodes
// that are children of a URL node are rendered accordingly.
type URL struct {
	*Position        // position in the source.
	Tag       string // tag (in lowercase).
	Attribute string // attribute (in lowercase).
	Value     []Node // value nodes.
}

// NewURL returns a new URL node.
func NewURL(pos *Position, tag, attribute string, value []Node) *URL {
	return &URL{pos, tag, attribute, value}
}

// Using node represents a "using" statement.
type Using struct {
	*Position            // position in the source.
	Statement Node       // statement preceding the using statement.
	Type      Expression // type, can be a format identifier or a macro.
	Body      *Block     // body.
	Format    Format     // using content format.
}

// NewUsing returns a new Using node.
func NewUsing(pos *Position, stmt Node, typ Expression, body *Block, format Format) *Using {
	return &Using{pos, stmt, typ, body, format}
}

// Var node represents a "var" declaration.
type Var struct {
	*Position               // position in the source.
	Lhs       []*Identifier // left-hand side of assignment.
	Type      Expression    // nil for non-typed variable declarations.
	Rhs       []Expression  // nil for non-initialized variable declarations.
}

// NewVar returns a new Var node.
func NewVar(pos *Position, lhs []*Identifier, typ Expression, rhs []Expression) *Var {
	return &Var{pos, lhs, typ, rhs}
}

// String returns the string representation of n.
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
