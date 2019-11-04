package main

import (
	"fmt"

	"github.com/magicnumbers/gold"
	bf "gopkg.in/russross/blackfriday.v2"
)

const tmd = `
# Test

This is a test yo.

## Dude!

* one
* two
* three

[yoda](http://flame.com)
`

func main() {
	r := gold.TermRenderer{}
	out := bf.Run([]byte(tmd), bf.WithRenderer(&r))
	fmt.Println(string(out))
}
