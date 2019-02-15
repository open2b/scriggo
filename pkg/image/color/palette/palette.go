// Go version: go1.11.5

package palette

import original "image/color/palette"
import "scrigo"

var Package = scrigo.Package{
	"Plan9": &original.Plan9,
	"WebSafe": &original.WebSafe,
}
