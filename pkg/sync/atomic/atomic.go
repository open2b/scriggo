// Go version: go1.11.5

package atomic

import original "sync/atomic"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"AddInt32": original.AddInt32,
	"AddInt64": original.AddInt64,
	"AddUint32": original.AddUint32,
	"AddUint64": original.AddUint64,
	"AddUintptr": original.AddUintptr,
	"CompareAndSwapInt32": original.CompareAndSwapInt32,
	"CompareAndSwapInt64": original.CompareAndSwapInt64,
	"CompareAndSwapPointer": original.CompareAndSwapPointer,
	"CompareAndSwapUint32": original.CompareAndSwapUint32,
	"CompareAndSwapUint64": original.CompareAndSwapUint64,
	"CompareAndSwapUintptr": original.CompareAndSwapUintptr,
	"LoadInt32": original.LoadInt32,
	"LoadInt64": original.LoadInt64,
	"LoadPointer": original.LoadPointer,
	"LoadUint32": original.LoadUint32,
	"LoadUint64": original.LoadUint64,
	"LoadUintptr": original.LoadUintptr,
	"StoreInt32": original.StoreInt32,
	"StoreInt64": original.StoreInt64,
	"StorePointer": original.StorePointer,
	"StoreUint32": original.StoreUint32,
	"StoreUint64": original.StoreUint64,
	"StoreUintptr": original.StoreUintptr,
	"SwapInt32": original.SwapInt32,
	"SwapInt64": original.SwapInt64,
	"SwapPointer": original.SwapPointer,
	"SwapUint32": original.SwapUint32,
	"SwapUint64": original.SwapUint64,
	"SwapUintptr": original.SwapUintptr,
	"Value": reflect.TypeOf(original.Value{}),
}
