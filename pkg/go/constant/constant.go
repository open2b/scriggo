// Go version: go1.11.5

package constant

import original "go/constant"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"BinaryOp": original.BinaryOp,
	"BitLen": original.BitLen,
	"Bool": scrigo.Constant(original.Bool, nil),
	"BoolVal": original.BoolVal,
	"Bytes": original.Bytes,
	"Compare": original.Compare,
	"Complex": scrigo.Constant(original.Complex, nil),
	"Denom": original.Denom,
	"Float": scrigo.Constant(original.Float, nil),
	"Float32Val": original.Float32Val,
	"Float64Val": original.Float64Val,
	"Imag": original.Imag,
	"Int": scrigo.Constant(original.Int, nil),
	"Int64Val": original.Int64Val,
	"Kind": reflect.TypeOf(original.Kind(int(0))),
	"MakeBool": original.MakeBool,
	"MakeFloat64": original.MakeFloat64,
	"MakeFromBytes": original.MakeFromBytes,
	"MakeFromLiteral": original.MakeFromLiteral,
	"MakeImag": original.MakeImag,
	"MakeInt64": original.MakeInt64,
	"MakeString": original.MakeString,
	"MakeUint64": original.MakeUint64,
	"MakeUnknown": original.MakeUnknown,
	"Num": original.Num,
	"Real": original.Real,
	"Shift": original.Shift,
	"Sign": original.Sign,
	"String": scrigo.Constant(original.String, nil),
	"StringVal": original.StringVal,
	"ToComplex": original.ToComplex,
	"ToFloat": original.ToFloat,
	"ToInt": original.ToInt,
	"Uint64Val": original.Uint64Val,
	"UnaryOp": original.UnaryOp,
	"Unknown": scrigo.Constant(original.Unknown, nil),
	"Value": reflect.TypeOf((*original.Value)(nil)).Elem(),
}
