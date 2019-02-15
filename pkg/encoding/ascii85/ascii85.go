// Go version: go1.11.5

package ascii85

import original "encoding/ascii85"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"CorruptInputError": reflect.TypeOf(original.CorruptInputError(int64(0))),
	"Decode": original.Decode,
	"Encode": original.Encode,
	"MaxEncodedLen": original.MaxEncodedLen,
	"NewDecoder": original.NewDecoder,
	"NewEncoder": original.NewEncoder,
}
