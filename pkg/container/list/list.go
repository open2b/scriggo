// Go version: go1.11.5

package list

import original "container/list"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Element": reflect.TypeOf(original.Element{}),
	"List": reflect.TypeOf(original.List{}),
	"New": original.New,
}
