// Go version: go1.11.5

package ring

import original "container/ring"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"New": original.New,
	"Ring": reflect.TypeOf(original.Ring{}),
}
