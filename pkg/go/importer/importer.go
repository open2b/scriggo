// Go version: go1.11.5

package importer

import original "go/importer"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Default": original.Default,
	"For": original.For,
	"Lookup": reflect.TypeOf((original.Lookup)(nil)),
}
