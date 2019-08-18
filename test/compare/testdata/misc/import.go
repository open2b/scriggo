// compile

package main

import "bufio"
import _ "bytes"
import . "crypto/sha1"
import p "encoding/base64"

import ( "errors" )
import ( "fmt"; )
import ( "io/ioutil"
)
import ( "log";
)
import (
	"math"
)
import (
	"math/rand";
)

func main() {
	_ = bufio.MaxScanTokenSize
	_ = BlockSize
	_ = p.StdPadding
	_ = errors.New
	_ = fmt.Errorf
	_ = ioutil.Discard
	_ = log.Ldate
	_ = math.E
	_ = rand.ExpFloat64
}
