// Go version: go1.11.5

package scanner

import "reflect"
import original "text/scanner"
import "scrigo"

var Package = scrigo.Package{
	"Char": scrigo.Constant(original.Char, nil),
	"Comment": scrigo.Constant(original.Comment, nil),
	"EOF": scrigo.Constant(original.EOF, nil),
	"Float": scrigo.Constant(original.Float, nil),
	"GoTokens": scrigo.Constant(original.GoTokens, nil),
	"GoWhitespace": scrigo.Constant(original.GoWhitespace, nil),
	"Ident": scrigo.Constant(original.Ident, nil),
	"Int": scrigo.Constant(original.Int, nil),
	"Position": reflect.TypeOf(original.Position{}),
	"RawString": scrigo.Constant(original.RawString, nil),
	"ScanChars": scrigo.Constant(original.ScanChars, nil),
	"ScanComments": scrigo.Constant(original.ScanComments, nil),
	"ScanFloats": scrigo.Constant(original.ScanFloats, nil),
	"ScanIdents": scrigo.Constant(original.ScanIdents, nil),
	"ScanInts": scrigo.Constant(original.ScanInts, nil),
	"ScanRawStrings": scrigo.Constant(original.ScanRawStrings, nil),
	"ScanStrings": scrigo.Constant(original.ScanStrings, nil),
	"Scanner": reflect.TypeOf(original.Scanner{}),
	"SkipComments": scrigo.Constant(original.SkipComments, nil),
	"String": scrigo.Constant(original.String, nil),
	"TokenString": original.TokenString,
}
