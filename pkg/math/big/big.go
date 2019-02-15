// Go version: go1.11.5

package big

import original "math/big"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Accuracy": reflect.TypeOf(original.Accuracy(int8(0))),
	"ErrNaN": reflect.TypeOf(original.ErrNaN{}),
	"Float": reflect.TypeOf(original.Float{}),
	"Int": reflect.TypeOf(original.Int{}),
	"Jacobi": original.Jacobi,
	"NewFloat": original.NewFloat,
	"NewInt": original.NewInt,
	"NewRat": original.NewRat,
	"ParseFloat": original.ParseFloat,
	"Rat": reflect.TypeOf(original.Rat{}),
	"RoundingMode": reflect.TypeOf(original.RoundingMode(byte(0))),
	"Word": reflect.TypeOf(original.Word(uint(0))),
}
