// run

package main

import (
	"fmt"
	"net/url"
)

func main() {
	{
		u, _ := url.Parse("//username1@host")
		fmt.Println((*u).User)
	}
	{
		u, _ := url.Parse("//username2@host")
		fmt.Println(u.User)
	}
}
