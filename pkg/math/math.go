// Go version: go1.11.5

package math

import original "math"
import "scrigo"

var Package = scrigo.Package{
	"Abs": original.Abs,
	"Acos": original.Acos,
	"Acosh": original.Acosh,
	"Asin": original.Asin,
	"Asinh": original.Asinh,
	"Atan": original.Atan,
	"Atan2": original.Atan2,
	"Atanh": original.Atanh,
	"Cbrt": original.Cbrt,
	"Ceil": original.Ceil,
	"Copysign": original.Copysign,
	"Cos": original.Cos,
	"Cosh": original.Cosh,
	"Dim": original.Dim,
	"Erf": original.Erf,
	"Erfc": original.Erfc,
	"Erfcinv": original.Erfcinv,
	"Erfinv": original.Erfinv,
	"Exp": original.Exp,
	"Exp2": original.Exp2,
	"Expm1": original.Expm1,
	"Float32bits": original.Float32bits,
	"Float32frombits": original.Float32frombits,
	"Float64bits": original.Float64bits,
	"Float64frombits": original.Float64frombits,
	"Floor": original.Floor,
	"Frexp": original.Frexp,
	"Gamma": original.Gamma,
	"Hypot": original.Hypot,
	"Ilogb": original.Ilogb,
	"Inf": original.Inf,
	"IsInf": original.IsInf,
	"IsNaN": original.IsNaN,
	"J0": original.J0,
	"J1": original.J1,
	"Jn": original.Jn,
	"Ldexp": original.Ldexp,
	"Lgamma": original.Lgamma,
	"Log": original.Log,
	"Log10": original.Log10,
	"Log1p": original.Log1p,
	"Log2": original.Log2,
	"Logb": original.Logb,
	"Max": original.Max,
	"MaxUint64": scrigo.Constant(uint64(original.MaxUint64), nil),
	"Min": original.Min,
	"Mod": original.Mod,
	"Modf": original.Modf,
	"NaN": original.NaN,
	"Nextafter": original.Nextafter,
	"Nextafter32": original.Nextafter32,
	"Pow": original.Pow,
	"Pow10": original.Pow10,
	"Remainder": original.Remainder,
	"Round": original.Round,
	"RoundToEven": original.RoundToEven,
	"Signbit": original.Signbit,
	"Sin": original.Sin,
	"Sincos": original.Sincos,
	"Sinh": original.Sinh,
	"Sqrt": original.Sqrt,
	"Tan": original.Tan,
	"Tanh": original.Tanh,
	"Trunc": original.Trunc,
	"Y0": original.Y0,
	"Y1": original.Y1,
	"Yn": original.Yn,
}
