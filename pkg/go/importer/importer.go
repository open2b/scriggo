// Go version: go1.11.5

package importer

import "scrigo"
import "reflect"
import original "go/importer"

var Package = scrigo.Package{
	"Default": original.Default,
	"For": original.For,
	"Lookup": reflect.TypeOf((original.Lookup)(nil)),
}
