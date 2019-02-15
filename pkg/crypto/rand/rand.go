// Go version: go1.11.5

package rand

import original "crypto/rand"
import "scrigo"

var Package = scrigo.Package{
	"Int": original.Int,
	"Prime": original.Prime,
	"Read": original.Read,
	"Reader": &original.Reader,
}
