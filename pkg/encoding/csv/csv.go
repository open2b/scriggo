// Go version: go1.11.5

package csv

import "reflect"
import original "encoding/csv"
import "scrigo"

var Package = scrigo.Package{
	"ErrBareQuote": &original.ErrBareQuote,
	"ErrFieldCount": &original.ErrFieldCount,
	"ErrQuote": &original.ErrQuote,
	"ErrTrailingComma": &original.ErrTrailingComma,
	"NewReader": original.NewReader,
	"NewWriter": original.NewWriter,
	"ParseError": reflect.TypeOf(original.ParseError{}),
	"Reader": reflect.TypeOf(original.Reader{}),
	"Writer": reflect.TypeOf(original.Writer{}),
}
