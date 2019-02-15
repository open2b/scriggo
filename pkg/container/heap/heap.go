// Go version: go1.11.5

package heap

import "reflect"
import original "container/heap"
import "scrigo"

var Package = scrigo.Package{
	"Fix": original.Fix,
	"Init": original.Init,
	"Interface": reflect.TypeOf((*original.Interface)(nil)).Elem(),
	"Pop": original.Pop,
	"Push": original.Push,
	"Remove": original.Remove,
}
