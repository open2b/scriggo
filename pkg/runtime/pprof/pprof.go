// Go version: go1.11.5

package pprof

import original "runtime/pprof"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Do": original.Do,
	"ForLabels": original.ForLabels,
	"Label": original.Label,
	"LabelSet": reflect.TypeOf(original.LabelSet{}),
	"Labels": original.Labels,
	"Lookup": original.Lookup,
	"NewProfile": original.NewProfile,
	"Profile": reflect.TypeOf(original.Profile{}),
	"Profiles": original.Profiles,
	"SetGoroutineLabels": original.SetGoroutineLabels,
	"StartCPUProfile": original.StartCPUProfile,
	"StopCPUProfile": original.StopCPUProfile,
	"WithLabels": original.WithLabels,
	"WriteHeapProfile": original.WriteHeapProfile,
}
