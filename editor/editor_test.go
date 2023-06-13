package editor

import (
	"reflect"
	"testing"
)

func TestEditor(t *testing.T) {
	filename := "README.md"
	for k, v := range map[string][]string{
		"":             {"nano", filename},
		"nvim":         {"nvim", filename},
		"vim":          {"vim", filename},
		"vscode --foo": {"vscode", "--foo", filename},
		"nvim -a -b":   {"nvim", "-a", "-b", filename},
	} {
		t.Run(k, func(t *testing.T) {
			t.Setenv("EDITOR", k)
			cmd := Cmd("README.md")
			got := cmd.Args
			if !reflect.DeepEqual(got, v) {
				t.Fatalf("expected %v; got %v", v, got)
			}
		})
	}
}
