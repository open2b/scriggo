// Go version: go1.11.5

package sort

import "scrigo"
import "reflect"
import original "sort"

var Package = scrigo.Package{
	"Float64Slice": reflect.TypeOf((original.Float64Slice)(nil)),
	"Float64s": original.Float64s,
	"Float64sAreSorted": original.Float64sAreSorted,
	"IntSlice": reflect.TypeOf((original.IntSlice)(nil)),
	"Interface": reflect.TypeOf((*original.Interface)(nil)).Elem(),
	"Ints": original.Ints,
	"IntsAreSorted": original.IntsAreSorted,
	"IsSorted": original.IsSorted,
	"Reverse": original.Reverse,
	"Search": original.Search,
	"SearchFloat64s": original.SearchFloat64s,
	"SearchInts": original.SearchInts,
	"SearchStrings": original.SearchStrings,
	"Slice": original.Slice,
	"SliceIsSorted": original.SliceIsSorted,
	"SliceStable": original.SliceStable,
	"Sort": original.Sort,
	"Stable": original.Stable,
	"StringSlice": reflect.TypeOf((original.StringSlice)(nil)),
	"Strings": original.Strings,
	"StringsAreSorted": original.StringsAreSorted,
}
