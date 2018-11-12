package util

import (
	"errors"
	"fmt"
	"io"
	"open2b"
	"strconv"

	"open2b/template/ast"
)

type dumper struct {
	output             io.Writer
	indentLevel        int
	externalReferences []*ast.Tree
}

type errVisitor struct {
	err error
}

func (e errVisitor) Error() string {
	return e.err.Error()
}

// Visit elabora un nodo di un albero, scrivendo su un Writer
// la rappresentazione del nodo stesso correttamente indentata.
// Il metodo Visit viene chiamato dalla funzione Walk.
func (d *dumper) Visit(node ast.Node) Visitor {

	// Gestione della chiamata v.Visit(nil) effettuata da Walk.
	if node == nil {
		d.indentLevel-- // Risale l'albero.
		return nil
	}

	d.indentLevel++

	// Nel caso in cui node contenga un riferimento
	// ad un altro albero (ovvero node è di tipo Import, Extend, ShowPath), aggiunge il riferimento
	// alla lista externalReferences per visitare i nodi con una chiamata ricorsiva della Dump.
	var tree *ast.Tree
	switch n := node.(type) {
	case *ast.Import:
		tree = n.Tree
	case *ast.Extend:
		tree = n.Tree
	case *ast.ShowPath:
		tree = n.Tree
	default:
		// Nessun riferimento da aggiungere, prosegue senza fare niente.
	}
	if tree != nil {
		d.externalReferences = append(d.externalReferences, tree)
	}

	// Se il nodo è di tipo Tree, lo scrive e ritorna senza fare altro.
	if n, ok := node.(*ast.Tree); ok {
		_, err := fmt.Fprintf(d.output, "\nTree: %v:%v\n", strconv.Quote(n.Path), n.Position)
		if err != nil {
			panic(errVisitor{err})
		}
		return d
	}

	// Cerca la rappresentazione come stringa di node.
	// Se il caso non è qui definito, viene utilizzata la conversione
	// a stringa di default del nodo.
	var text string
	switch n := node.(type) {
	case *ast.Text:
		text = n.Text
		if len(text) > 30 {
			text = open2b.Truncate(text, 30) + "..."
		}
		text = strconv.Quote(text)
	case *ast.If:
		text = n.Expr.String()
	default:
		text = fmt.Sprintf("%v", node)
	}

	// Inserisce il giusto livello di indentazione.
	for i := 0; i < d.indentLevel; i++ {
		_, err := fmt.Fprint(d.output, "│    ")
		if err != nil {
			panic(errVisitor{err})
		}
	}

	// Determina il tipo rimuovendo il prefisso "*ast."
	typeStr := fmt.Sprintf("%T", node)[5:]

	posStr := node.Pos().String()

	_, err := fmt.Fprintf(d.output, "%v (%v) %v\n", typeStr, posStr, text)
	if err != nil {
		panic(errVisitor{err})
	}

	return d
}

// Dump scrive su w il dump dell'albero.
// Nel caso in cui l'albero sia nil, la funzione interrompe l'esecuzione
// restituendo un errore diverso da nil.
// Nel caso in cui l'albero non sia esteso, la funzione conclude la sua
// esecuzione dopo aver scritto l'albero di base restituendo un errore non nil.
func Dump(w io.Writer, node ast.Node) (err error) {

	defer func() {
		if r := recover(); r != nil {
			if t, ok := r.(errVisitor); ok {
				err = t.err
			} else {
				panic(r)
			}
		}
	}()

	if node == nil {
		return errors.New("can't dump a nil tree")
	}

	d := dumper{w, -1, []*ast.Tree{}}
	Walk(&d, node)

	// Scrive gli alberi che sono stati dichiarati come riferimenti esterni.
	for _, tree := range d.externalReferences {
		Dump(w, tree)
	}

	return nil
}
