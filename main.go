package main

import (
	"flag"
	"fmt"
	"io/ioutil"
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
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("Missing Markdown file. Usage: ./gold -s STYLE.json FILE.md")
		os.Exit(1)
	}
	f, err := os.Open(args[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	b, _ := ioutil.ReadAll(f)
	r, err := gold.NewTermRenderer(*s)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	out := bf.Run(b, bf.WithRenderer(r))
	fmt.Printf("%s", string(out))
}
