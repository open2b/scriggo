// Go version: go1.11.5

package base32

import original "encoding/base32"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"CorruptInputError": reflect.TypeOf(original.CorruptInputError(int64(0))),
	"Encoding": reflect.TypeOf(original.Encoding{}),
	"HexEncoding": &original.HexEncoding,
	"NewDecoder": original.NewDecoder,
	"NewEncoder": original.NewEncoder,
	"NewEncoding": original.NewEncoding,
	"NoPadding": scrigo.Constant(original.NoPadding, nil),
	"StdEncoding": &original.StdEncoding,
	"StdPadding": scrigo.Constant(original.StdPadding, nil),
}
