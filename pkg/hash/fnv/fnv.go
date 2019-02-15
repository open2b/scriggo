// Go version: go1.11.5

package fnv

import original "hash/fnv"
import "scrigo"

var Package = scrigo.Package{
	"New128": original.New128,
	"New128a": original.New128a,
	"New32": original.New32,
	"New32a": original.New32a,
	"New64": original.New64,
	"New64a": original.New64a,
}
