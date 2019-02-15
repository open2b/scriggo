// Go version: go1.11.5

package base64

import original "encoding/base64"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"CorruptInputError": reflect.TypeOf(original.CorruptInputError(int64(0))),
	"Encoding": reflect.TypeOf(original.Encoding{}),
	"NewDecoder": original.NewDecoder,
	"NewEncoder": original.NewEncoder,
	"NewEncoding": original.NewEncoding,
	"RawStdEncoding": &original.RawStdEncoding,
	"RawURLEncoding": &original.RawURLEncoding,
	"StdEncoding": &original.StdEncoding,
	"URLEncoding": &original.URLEncoding,
}
