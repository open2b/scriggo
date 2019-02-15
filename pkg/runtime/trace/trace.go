// Go version: go1.11.5

package trace

import "scrigo"
import "reflect"
import original "runtime/trace"

var Package = scrigo.Package{
	"IsEnabled": original.IsEnabled,
	"Log": original.Log,
	"Logf": original.Logf,
	"NewTask": original.NewTask,
	"Region": reflect.TypeOf(original.Region{}),
	"Start": original.Start,
	"StartRegion": original.StartRegion,
	"Stop": original.Stop,
	"Task": reflect.TypeOf(original.Task{}),
	"WithRegion": original.WithRegion,
}
