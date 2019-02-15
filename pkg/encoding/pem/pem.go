// Go version: go1.11.5

package pem

import original "encoding/pem"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Block": reflect.TypeOf(original.Block{}),
	"Decode": original.Decode,
	"Encode": original.Encode,
	"EncodeToMemory": original.EncodeToMemory,
}
