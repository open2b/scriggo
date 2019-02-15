// Go version: go1.11.5

package sync

import original "sync"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Cond": reflect.TypeOf(original.Cond{}),
	"Locker": reflect.TypeOf((*original.Locker)(nil)).Elem(),
	"Map": reflect.TypeOf(original.Map{}),
	"Mutex": reflect.TypeOf(original.Mutex{}),
	"NewCond": original.NewCond,
	"Once": reflect.TypeOf(original.Once{}),
	"Pool": reflect.TypeOf(original.Pool{}),
	"RWMutex": reflect.TypeOf(original.RWMutex{}),
	"WaitGroup": reflect.TypeOf(original.WaitGroup{}),
}
