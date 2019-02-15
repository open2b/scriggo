// Go version: go1.11.5

package subtle

import original "crypto/subtle"
import "scrigo"

var Package = scrigo.Package{
	"ConstantTimeByteEq": original.ConstantTimeByteEq,
	"ConstantTimeCompare": original.ConstantTimeCompare,
	"ConstantTimeCopy": original.ConstantTimeCopy,
	"ConstantTimeEq": original.ConstantTimeEq,
	"ConstantTimeLessOrEq": original.ConstantTimeLessOrEq,
	"ConstantTimeSelect": original.ConstantTimeSelect,
}
