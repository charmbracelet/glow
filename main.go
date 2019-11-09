package main

import (
	"flag"
	"fmt"
	"os"

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
	s := flag.String("s", "", "style json path")
	flag.Parse()
	r, err := gold.NewTermRenderer(*s)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	out := bf.Run([]byte(tmd), bf.WithRenderer(r))
	fmt.Println(string(out))
}
