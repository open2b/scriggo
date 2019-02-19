// Go version: go1.11.5

package parser

import original "go/parser"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"AllErrors": scrigo.Constant(original.AllErrors, nil),
	"DeclarationErrors": scrigo.Constant(original.DeclarationErrors, nil),
	"ImportsOnly": scrigo.Constant(original.ImportsOnly, nil),
	"Mode": reflect.TypeOf(original.Mode(uint(0))),
	"PackageClauseOnly": scrigo.Constant(original.PackageClauseOnly, nil),
	"ParseComments": scrigo.Constant(original.ParseComments, nil),
	"ParseDir": original.ParseDir,
	"ParseExpr": original.ParseExpr,
	"ParseExprFrom": original.ParseExprFrom,
	"ParseFile": original.ParseFile,
	"SpuriousErrors": scrigo.Constant(original.SpuriousErrors, nil),
	"Trace": scrigo.Constant(original.Trace, nil),
}
