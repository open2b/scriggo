// Go version: go1.11.5

package palette

import "scrigo"
import original "image/color/palette"

var Package = scrigo.Package{
	"Plan9": &original.Plan9,
	"WebSafe": &original.WebSafe,
}
