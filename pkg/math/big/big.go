// Go version: go1.11.5

package big

import original "math/big"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Above": scrigo.Constant(original.Above, nil),
	"Accuracy": reflect.TypeOf(original.Accuracy(int8(0))),
	"AwayFromZero": scrigo.Constant(original.AwayFromZero, nil),
	"Below": scrigo.Constant(original.Below, nil),
	"ErrNaN": reflect.TypeOf(original.ErrNaN{}),
	"Exact": scrigo.Constant(original.Exact, nil),
	"Float": reflect.TypeOf(original.Float{}),
	"Int": reflect.TypeOf(original.Int{}),
	"Jacobi": original.Jacobi,
	"MaxBase": scrigo.Constant(original.MaxBase, nil),
	"MaxExp": scrigo.Constant(original.MaxExp, nil),
	"MaxPrec": scrigo.Constant(original.MaxPrec, nil),
	"MinExp": scrigo.Constant(original.MinExp, nil),
	"NewFloat": original.NewFloat,
	"NewInt": original.NewInt,
	"NewRat": original.NewRat,
	"ParseFloat": original.ParseFloat,
	"Rat": reflect.TypeOf(original.Rat{}),
	"RoundingMode": reflect.TypeOf(original.RoundingMode(byte(0))),
	"ToNearestAway": scrigo.Constant(original.ToNearestAway, nil),
	"ToNearestEven": scrigo.Constant(original.ToNearestEven, nil),
	"ToNegativeInf": scrigo.Constant(original.ToNegativeInf, nil),
	"ToPositiveInf": scrigo.Constant(original.ToPositiveInf, nil),
	"ToZero": scrigo.Constant(original.ToZero, nil),
	"Word": reflect.TypeOf(original.Word(uint(0))),
}
