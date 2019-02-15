// Go version: go1.11.5

package strconv

import "scrigo"
import "reflect"
import original "strconv"

var Package = scrigo.Package{
	"AppendBool": original.AppendBool,
	"AppendFloat": original.AppendFloat,
	"AppendInt": original.AppendInt,
	"AppendQuote": original.AppendQuote,
	"AppendQuoteRune": original.AppendQuoteRune,
	"AppendQuoteRuneToASCII": original.AppendQuoteRuneToASCII,
	"AppendQuoteRuneToGraphic": original.AppendQuoteRuneToGraphic,
	"AppendQuoteToASCII": original.AppendQuoteToASCII,
	"AppendQuoteToGraphic": original.AppendQuoteToGraphic,
	"AppendUint": original.AppendUint,
	"Atoi": original.Atoi,
	"CanBackquote": original.CanBackquote,
	"ErrRange": &original.ErrRange,
	"ErrSyntax": &original.ErrSyntax,
	"FormatBool": original.FormatBool,
	"FormatFloat": original.FormatFloat,
	"FormatInt": original.FormatInt,
	"FormatUint": original.FormatUint,
	"IsGraphic": original.IsGraphic,
	"IsPrint": original.IsPrint,
	"Itoa": original.Itoa,
	"NumError": reflect.TypeOf(original.NumError{}),
	"ParseBool": original.ParseBool,
	"ParseFloat": original.ParseFloat,
	"ParseInt": original.ParseInt,
	"ParseUint": original.ParseUint,
	"Quote": original.Quote,
	"QuoteRune": original.QuoteRune,
	"QuoteRuneToASCII": original.QuoteRuneToASCII,
	"QuoteRuneToGraphic": original.QuoteRuneToGraphic,
	"QuoteToASCII": original.QuoteToASCII,
	"QuoteToGraphic": original.QuoteToGraphic,
	"Unquote": original.Unquote,
	"UnquoteChar": original.UnquoteChar,
}
