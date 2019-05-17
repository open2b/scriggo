package script

import (
	"io"
	"io/ioutil"
	"reflect"

	"scrigo/internal/compiler"
	"scrigo/internal/compiler/ast"
	"scrigo/vm"
)

type Script struct {
	fn *vm.ScrigoFunction
}

func Compile(src io.Reader, main *compiler.GoPackage) (*Script, error) {

	// Parsing.
	buf, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}
	tree, err := compiler.ParseSource(buf, ast.ContextNone)
	if err != nil {
		return nil, err
	}

	// Type checking.
	pkgInfo, err := typecheck(tree, main)
	if err != nil {
		return nil, err
	}
	tci := map[string]*compiler.PackageInfo{"main": pkgInfo}

	// Emitting.
	// TODO(Gianluca): pass "main" to emitter.
	// main contains user defined variables.
	emitter := compiler.NewEmitter(tree, nil, tci["main"].TypeInfo, tci["main"].IndirectVars)
	fn := compiler.NewScrigoFunction("main", "main", reflect.FuncOf(nil, nil, false))
	emitter.CurrentFunction = fn
	emitter.FB = compiler.NewBuilder(emitter.CurrentFunction)
	emitter.FB.EnterScope()
	compiler.AddExplicitReturn(tree)
	emitter.EmitNodes(tree.Nodes)
	emitter.FB.ExitScope()

	return &Script{fn: emitter.CurrentFunction}, nil
}

func Execute(script *Script, globals map[string]reflect.Value) error {
	// TODO (Gianluca): "overwrite" main with globals before calling runwithglobals (do not overwrite)
	pvm := vm.New()
	_, err := pvm.RunWithGlobals(script.fn, globals)
	return err
}

func typecheck(tree *ast.Tree, main *compiler.GoPackage) (_ *compiler.PackageInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			if rerr, ok := r.(*compiler.Error); ok {
				err = rerr
			} else {
				panic(r)
			}
		}
	}()
	tc := compiler.NewTypechecker(tree.Path, true)
	tc.Universe = compiler.Universe
	if main != nil {
		tc.Scopes = append(tc.Scopes, main.ToTypeCheckerScope())
	}
	tc.CheckNodesInNewScope(tree.Nodes)
	pkgInfo := &compiler.PackageInfo{}
	pkgInfo.IndirectVars = tc.IndirectVars
	pkgInfo.TypeInfo = tc.TypeInfo
	return pkgInfo, err
}
