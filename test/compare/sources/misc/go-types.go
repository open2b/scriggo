// runcompare

package main

import (
	"bytes"
	"time"
)

func main() {
	var m interface{} = bytes.Reader{}
	_ = m.(bytes.Reader)
	var m2 interface{} = time.Month(3)
	_ = m2.(time.Month)
	_ = time.Date(2000, 5, 1, 0, 0, 0, 0, time.UTC)
}
