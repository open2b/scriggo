// Go version: go1.11.5

package suffixarray

import original "index/suffixarray"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Index": reflect.TypeOf(original.Index{}),
	"New": original.New,
}
