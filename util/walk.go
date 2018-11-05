package util

import (
	"fmt"

	"open2b/template/ast"
)

// Il metodo Visit di Visitor viene invocato per ogni nodo incontrato da Walk.
type Visitor interface {
	Visit(node ast.Node) (w Visitor)
}

// Walk visita in profondità un albero.
// Inizialmente chiama v.Visit(node), dove node non deve essere nil.
// Se il valore w restituito da v.Visit(node) è diverso da nil, Walk viene
// chiamata ricorsivamente utilizzando w come Visitor su tutti i figli diversi da nil dell'albero.
// Chiama infine w.Visit(nil).
func Walk(v Visitor, node ast.Node) {

	if v == nil {
		panic("v can't be nil")
	}

	if node == nil {
		panic("node can't be nil")
	}

	v = v.Visit(node)

	if v == nil {
		return
	}

	switch n := node.(type) {

	// Se il figlio è un tipo concreto, lo lascia in gestione alla Visit,
	// altrimenti lo gestisce qui nella Walk.

	case *ast.Tree:
		for _, child := range n.Nodes {
			Walk(v, child)
		}

	case *ast.Var:
		Walk(v, n.Expr)

	case *ast.Assignment:
		Walk(v, n.Expr)

	case *ast.For:
		Walk(v, n.Expr1)

		if n.Expr2 != nil {
			Walk(v, n.Expr2)
		}

		for _, n := range n.Nodes {
			Walk(v, n)
		}

	case *ast.If:
		Walk(v, n.Expr)
		for _, child := range n.Then {
			Walk(v, child)
		}

		for _, child := range n.Else {
			Walk(v, child)
		}

	case *ast.Region:
		for _, child := range n.Body {
			Walk(v, child)
		}

	case *ast.Value:
		Walk(v, n.Expr)

	case *ast.Parentesis:
		Walk(v, n.Expr)

	case *ast.UnaryOperator:
		Walk(v, n.Expr)

	case *ast.BinaryOperator:
		Walk(v, n.Expr1)
		Walk(v, n.Expr2)

	case *ast.Call:
		for _, arg := range n.Args {
			Walk(v, arg)
		}

	case *ast.Index:
		Walk(v, n.Expr)
		Walk(v, n.Index)

	case *ast.Slice:
		Walk(v, n.Expr)
		if n.Low != nil {
			Walk(v, n.Low)
		}
		if n.High != nil {
			Walk(v, n.High)
		}

	case *ast.Selector:
		Walk(v, n.Expr)

	case *ast.Extend:
	case *ast.Import:
	case *ast.ShowPath:
		// Niente da fare, la visita dell'albero
		// espanso viene effettuata dalla funzione Visit se necessario.

	case *ast.Int:
	case *ast.Number:
	case *ast.Identifier:
	case *ast.String:
	case *ast.ShowRegion:
	case *ast.Comment:
	case *ast.Break:
	case *ast.Continue:
	case *ast.Text:
		// Niente da fare

	default:
		panic(fmt.Sprintf("No cases were defined for type %T on function Walk", n))
	}

	v.Visit(nil)

}

// Visit implementa l'interfaccia Visitor per la funzione f.
func (f inspector) Visit(node ast.Node) Visitor {
	if f(node) {
		return f
	}
	return nil
}

type inspector func(ast.Node) bool

// Inspect visita l'albero chiamando la funzione f
// su ogni suo nodo. Per ulteriori informazioni si veda la documentazione
// della funzione Walk.
func Inspect(node ast.Node, f func(ast.Node) bool) {
	Walk(inspector(f), node)
}
