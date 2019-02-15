// Go version: go1.11.5

package bytes

import original "bytes"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Buffer": reflect.TypeOf(original.Buffer{}),
	"Compare": original.Compare,
	"Contains": original.Contains,
	"ContainsAny": original.ContainsAny,
	"ContainsRune": original.ContainsRune,
	"Count": original.Count,
	"Equal": original.Equal,
	"EqualFold": original.EqualFold,
	"ErrTooLarge": &original.ErrTooLarge,
	"Fields": original.Fields,
	"FieldsFunc": original.FieldsFunc,
	"HasPrefix": original.HasPrefix,
	"HasSuffix": original.HasSuffix,
	"Index": original.Index,
	"IndexAny": original.IndexAny,
	"IndexByte": original.IndexByte,
	"IndexFunc": original.IndexFunc,
	"IndexRune": original.IndexRune,
	"Join": original.Join,
	"LastIndex": original.LastIndex,
	"LastIndexAny": original.LastIndexAny,
	"LastIndexByte": original.LastIndexByte,
	"LastIndexFunc": original.LastIndexFunc,
	"Map": original.Map,
	"NewBuffer": original.NewBuffer,
	"NewBufferString": original.NewBufferString,
	"NewReader": original.NewReader,
	"Reader": reflect.TypeOf(original.Reader{}),
	"Repeat": original.Repeat,
	"Replace": original.Replace,
	"Runes": original.Runes,
	"Split": original.Split,
	"SplitAfter": original.SplitAfter,
	"SplitAfterN": original.SplitAfterN,
	"SplitN": original.SplitN,
	"Title": original.Title,
	"ToLower": original.ToLower,
	"ToLowerSpecial": original.ToLowerSpecial,
	"ToTitle": original.ToTitle,
	"ToTitleSpecial": original.ToTitleSpecial,
	"ToUpper": original.ToUpper,
	"ToUpperSpecial": original.ToUpperSpecial,
	"Trim": original.Trim,
	"TrimFunc": original.TrimFunc,
	"TrimLeft": original.TrimLeft,
	"TrimLeftFunc": original.TrimLeftFunc,
	"TrimPrefix": original.TrimPrefix,
	"TrimRight": original.TrimRight,
	"TrimRightFunc": original.TrimRightFunc,
	"TrimSpace": original.TrimSpace,
	"TrimSuffix": original.TrimSuffix,
}
