// Go version: go1.11.5

package tabwriter

import original "text/tabwriter"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"AlignRight": scrigo.Constant(original.AlignRight, nil),
	"Debug": scrigo.Constant(original.Debug, nil),
	"DiscardEmptyColumns": scrigo.Constant(original.DiscardEmptyColumns, nil),
	"Escape": scrigo.Constant(original.Escape, nil),
	"FilterHTML": scrigo.Constant(original.FilterHTML, nil),
	"NewWriter": original.NewWriter,
	"StripEscape": scrigo.Constant(original.StripEscape, nil),
	"TabIndent": scrigo.Constant(original.TabIndent, nil),
	"Writer": reflect.TypeOf(original.Writer{}),
}
