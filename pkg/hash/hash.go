// Go version: go1.11.5

package hash

import original "hash"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Hash": reflect.TypeOf((*original.Hash)(nil)).Elem(),
	"Hash32": reflect.TypeOf((*original.Hash32)(nil)).Elem(),
	"Hash64": reflect.TypeOf((*original.Hash64)(nil)).Elem(),
}
