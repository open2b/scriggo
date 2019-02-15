// Go version: go1.11.5

package build

import "scrigo"
import "reflect"
import original "go/build"

var Package = scrigo.Package{
	"ArchChar": original.ArchChar,
	"Context": reflect.TypeOf(original.Context{}),
	"Default": &original.Default,
	"Import": original.Import,
	"ImportDir": original.ImportDir,
	"ImportMode": reflect.TypeOf(original.ImportMode(uint(0))),
	"IsLocalImport": original.IsLocalImport,
	"MultiplePackageError": reflect.TypeOf(original.MultiplePackageError{}),
	"NoGoError": reflect.TypeOf(original.NoGoError{}),
	"Package": reflect.TypeOf(original.Package{}),
	"ToolDir": &original.ToolDir,
}
