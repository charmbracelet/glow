package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/charmbracelet/gold"
)

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
	out, err := gold.RenderBytes(b, *s)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("%s", string(out))
}
