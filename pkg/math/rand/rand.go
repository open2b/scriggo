// Go version: go1.11.5

package rand

import original "math/rand"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"ExpFloat64": original.ExpFloat64,
	"Float32": original.Float32,
	"Float64": original.Float64,
	"Int": original.Int,
	"Int31": original.Int31,
	"Int31n": original.Int31n,
	"Int63": original.Int63,
	"Int63n": original.Int63n,
	"Intn": original.Intn,
	"New": original.New,
	"NewSource": original.NewSource,
	"NewZipf": original.NewZipf,
	"NormFloat64": original.NormFloat64,
	"Perm": original.Perm,
	"Rand": reflect.TypeOf(original.Rand{}),
	"Read": original.Read,
	"Seed": original.Seed,
	"Shuffle": original.Shuffle,
	"Source": reflect.TypeOf((*original.Source)(nil)).Elem(),
	"Source64": reflect.TypeOf((*original.Source64)(nil)).Elem(),
	"Uint32": original.Uint32,
	"Uint64": original.Uint64,
	"Zipf": reflect.TypeOf(original.Zipf{}),
}
