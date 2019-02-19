// Go version: go1.11.5

package fmt

import original "fmt"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Errorf": original.Errorf,
	"Formatter": reflect.TypeOf((*original.Formatter)(nil)).Elem(),
	"Fprint": original.Fprint,
	"Fprintf": original.Fprintf,
	"Fprintln": original.Fprintln,
	"Fscan": original.Fscan,
	"Fscanf": original.Fscanf,
	"Fscanln": original.Fscanln,
	"GoStringer": reflect.TypeOf((*original.GoStringer)(nil)).Elem(),
	"Print": original.Print,
	"Printf": original.Printf,
	"Println": original.Println,
	"Scan": original.Scan,
	"ScanState": reflect.TypeOf((*original.ScanState)(nil)).Elem(),
	"Scanf": original.Scanf,
	"Scanln": original.Scanln,
	"Scanner": reflect.TypeOf((*original.Scanner)(nil)).Elem(),
	"Sprint": original.Sprint,
	"Sprintf": original.Sprintf,
	"Sprintln": original.Sprintln,
	"Sscan": original.Sscan,
	"Sscanf": original.Sscanf,
	"Sscanln": original.Sscanln,
	"State": reflect.TypeOf((*original.State)(nil)).Elem(),
	"Stringer": reflect.TypeOf((*original.Stringer)(nil)).Elem(),
}
