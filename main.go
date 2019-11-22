package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/charmbracelet/gold"
)

func readerFromArgument(s string) (io.ReadCloser, error) {
	if s == "-" {
		return os.Stdin, nil
	}

	if isGitHubURL(s) {
		resp, err := findGitHubREADME(s)
		if err != nil {
			return nil, err
		}
		return resp.Body, nil
	}

	return os.Open(s)
}

func main() {
	s := flag.String("s", "", "style json path")
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("Missing Markdown file. Usage: ./gold -s STYLE.json FILE.md")
		os.Exit(1)
	}

	in, err := readerFromArgument(args[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer in.Close()

	b, _ := ioutil.ReadAll(in)
	out, err := gold.RenderBytes(b, *s)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("%s", string(out))
}
