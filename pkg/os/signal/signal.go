// Go version: go1.11.5

package signal

import original "os/signal"
import "scrigo"

var Package = scrigo.Package{
	"Ignore": original.Ignore,
	"Ignored": original.Ignored,
	"Notify": original.Notify,
	"Reset": original.Reset,
	"Stop": original.Stop,
}
