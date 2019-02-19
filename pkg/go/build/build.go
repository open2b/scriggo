// Go version: go1.11.5

package build

import original "go/build"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"AllowBinary": scrigo.Constant(original.AllowBinary, nil),
	"ArchChar": original.ArchChar,
	"Context": reflect.TypeOf(original.Context{}),
	"Default": &original.Default,
	"FindOnly": scrigo.Constant(original.FindOnly, nil),
	"IgnoreVendor": scrigo.Constant(original.IgnoreVendor, nil),
	"Import": original.Import,
	"ImportComment": scrigo.Constant(original.ImportComment, nil),
	"ImportDir": original.ImportDir,
	"ImportMode": reflect.TypeOf(original.ImportMode(uint(0))),
	"IsLocalImport": original.IsLocalImport,
	"MultiplePackageError": reflect.TypeOf(original.MultiplePackageError{}),
	"NoGoError": reflect.TypeOf(original.NoGoError{}),
	"Package": reflect.TypeOf(original.Package{}),
	"ToolDir": &original.ToolDir,
}
