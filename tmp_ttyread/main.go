package main

import (
	"fmt"
	"os"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"golang.org/x/term"
)

func main() {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Println("open err:", err)
		return
	}
	state, err := term.MakeRaw(int(tty.Fd()))
	if err != nil {
		fmt.Println("raw err:", err)
		return
	}
	defer func() { _ = term.Restore(int(tty.Fd()), state) }()

	cr, err := uv.NewCancelReader(tty)
	if err != nil {
		fmt.Println("cancelreader err:", err)
		return
	}
	buf := make([]byte, 64)
	go func() {
		time.Sleep(3 * time.Second)
		cr.Cancel()
	}()
	n, err := cr.Read(buf)
	fmt.Fprintf(os.Stderr, "read n=%d err=%v data=%q\n", n, err, buf[:n])
}
