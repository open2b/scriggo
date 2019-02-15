// Go version: go1.11.5

package bzip2

import original "compress/bzip2"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"NewReader": original.NewReader,
	"StructuralError": reflect.TypeOf(""),
}
