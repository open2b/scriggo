// Go version: go1.11.5

package tabwriter

import original "text/tabwriter"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"NewWriter": original.NewWriter,
	"Writer": reflect.TypeOf(original.Writer{}),
}
