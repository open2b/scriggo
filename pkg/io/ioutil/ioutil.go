// Go version: go1.11.5

package ioutil

import original "io/ioutil"
import "scrigo"

var Package = scrigo.Package{
	"Discard": &original.Discard,
	"NopCloser": original.NopCloser,
	"ReadAll": original.ReadAll,
	"ReadDir": original.ReadDir,
	"ReadFile": original.ReadFile,
	"TempDir": original.TempDir,
	"TempFile": original.TempFile,
	"WriteFile": original.WriteFile,
}
