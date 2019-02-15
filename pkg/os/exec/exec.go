// Go version: go1.11.5

package exec

import original "os/exec"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Cmd": reflect.TypeOf(original.Cmd{}),
	"Command": original.Command,
	"CommandContext": original.CommandContext,
	"ErrNotFound": &original.ErrNotFound,
	"Error": reflect.TypeOf(original.Error{}),
	"ExitError": reflect.TypeOf(original.ExitError{}),
	"LookPath": original.LookPath,
}
