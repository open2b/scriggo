// Go version: go1.11.5

package debug

import original "runtime/debug"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"FreeOSMemory": original.FreeOSMemory,
	"GCStats": reflect.TypeOf(original.GCStats{}),
	"PrintStack": original.PrintStack,
	"ReadGCStats": original.ReadGCStats,
	"SetGCPercent": original.SetGCPercent,
	"SetMaxStack": original.SetMaxStack,
	"SetMaxThreads": original.SetMaxThreads,
	"SetPanicOnFault": original.SetPanicOnFault,
	"SetTraceback": original.SetTraceback,
	"Stack": original.Stack,
	"WriteHeapDump": original.WriteHeapDump,
}
