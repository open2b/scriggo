package scrigo

import (
	"fmt"

	"scrigo/compiler/ast"

	"github.com/cockroachdb/apd"
)

var oneAPDDecimal = apd.New(1, 0)
var decimalContext = apd.BaseContext.WithPrecision(decPrecision)

var numberConversionContext *apd.Context

func init() {
	numberConversionContext = decimalContext.WithPrecision(decPrecision)
	numberConversionContext.Rounding = apd.RoundDown
}

const decPrecision = 32

type decimal struct {
	n *apd.Decimal
}

func (d decimal) Name() string {
	return "decimal"
}

func (d decimal) New() CustomNumber {
	return decimal{n: &apd.Decimal{}}
}

func (d decimal) Cmp(y CustomNumber) int {
	if y == nil {
		return d.n.Sign()
	}
	return d.n.Cmp(y.(decimal).n)
}

func (d decimal) BinaryOp(x CustomNumber, op ast.OperatorType, y CustomNumber) CustomNumber {
	dc := decimalContext
	xn := x.(decimal).n
	yn := y.(decimal).n
	switch op {
	case ast.OperatorAddition:
		_, _ = dc.Add(d.n, xn, yn)
	case ast.OperatorSubtraction:
		_, _ = dc.Sub(d.n, xn, yn)
	case ast.OperatorMultiplication:
		_, _ = dc.Mul(d.n, xn, yn)
	case ast.OperatorDivision:
		_, _ = dc.Quo(d.n, xn, yn)
	case ast.OperatorModulo:
		_, _ = dc.Rem(d.n, xn, yn)
	default:
		panic("unknown operator")
	}
	return d
}

func (d decimal) Inc() {
	_, _ = decimalContext.Add(d.n, d.n, oneAPDDecimal)
}

func (d decimal) Dec() {
	_, _ = decimalContext.Sub(d.n, d.n, oneAPDDecimal)
}

func (d decimal) Neg(x CustomNumber) CustomNumber {
	d.n.Neg(x.(decimal).n)
	return d
}

func (d decimal) Convert(x interface{}) (CustomNumber, error) {
	switch n := x.(type) {
	case int:
		d.n.SetInt64(int64(n))
	case int64:
		d.n.SetInt64(n)
	case int32:
		d.n.SetInt64(int64(n))
	case int16:
		d.n.SetInt64(int64(n))
	case int8:
		d.n.SetInt64(int64(n))
	case uint:
		d.n.SetInt64(int64(n))
	case uint64:
		d.n.SetInt64(int64(n))
	case uint32:
		d.n.SetInt64(int64(n))
	case uint16:
		d.n.SetInt64(int64(n))
	case uint8:
		d.n.SetInt64(int64(n))
	case float64:
		_, _ = d.n.SetFloat64(n)
	case float32:
		_, _ = d.n.SetFloat64(float64(n))
	case ConstantNumber:
		panic("TODO") // TODO
	default:
		return nil, fmt.Errorf("cannot convert to decimal")
	}
	return d, nil
}

func (d decimal) IsInf() bool {
	return d.n.Form == apd.Infinite
}

func (d decimal) IsNaN() bool {
	return d.n.Form == apd.NaN || d.n.Form == apd.NaNSignaling
}

func (d decimal) Format(s fmt.State, format rune) {
	d.n.Format(s, format)
}

func (d decimal) String() string {
	return d.n.String()
}
