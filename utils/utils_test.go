package utils

import (
	"bytes"
	"testing"

	"golang.org/x/text/encoding/unicode"
)

func TestToUTF8_PlainUTF8(t *testing.T) {
	input := []byte("# Hello, World!\n")
	got := ToUTF8(input)
	if !bytes.Equal(got, input) {
		t.Errorf("plain UTF-8 should pass through unchanged\ngot:  %q\nwant: %q", got, input)
	}
}

func TestToUTF8_UTF8BOM(t *testing.T) {
	input := append([]byte{0xEF, 0xBB, 0xBF}, []byte("# Hello\n")...)
	got := ToUTF8(input)
	want := []byte("# Hello\n")
	if !bytes.Equal(got, want) {
		t.Errorf("UTF-8 BOM should be stripped\ngot:  %q\nwant: %q", got, want)
	}
}

func TestToUTF8_UTF16LE(t *testing.T) {
	text := "# Hello\n"
	enc := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder()
	encoded, err := enc.Bytes([]byte(text))
	if err != nil {
		t.Fatalf("failed to encode test input: %v", err)
	}

	got := ToUTF8(encoded)
	if !bytes.Equal(got, []byte(text)) {
		t.Errorf("UTF-16 LE with BOM should be converted to UTF-8\ngot:  %q\nwant: %q", got, text)
	}
}

func TestToUTF8_UTF16BE(t *testing.T) {
	text := "# Hello\n"
	enc := unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewEncoder()
	encoded, err := enc.Bytes([]byte(text))
	if err != nil {
		t.Fatalf("failed to encode test input: %v", err)
	}

	got := ToUTF8(encoded)
	if !bytes.Equal(got, []byte(text)) {
		t.Errorf("UTF-16 BE with BOM should be converted to UTF-8\ngot:  %q\nwant: %q", got, text)
	}
}

func TestToUTF8_Empty(t *testing.T) {
	got := ToUTF8([]byte{})
	if len(got) != 0 {
		t.Errorf("empty input should return empty output, got: %q", got)
	}
}

func TestToUTF8_Nil(t *testing.T) {
	got := ToUTF8(nil)
	if got != nil {
		t.Errorf("nil input should return nil, got: %q", got)
	}
}

func TestToUTF8_UTF16LEWithMultibyteChars(t *testing.T) {
	// Test UTF-16 LE with non-ASCII characters (e.g. "# Héllo\n")
	text := "# H\u00e9llo\n"
	enc := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder()
	encoded, err := enc.Bytes([]byte(text))
	if err != nil {
		t.Fatalf("failed to encode test input: %v", err)
	}

	got := ToUTF8(encoded)
	if !bytes.Equal(got, []byte(text)) {
		t.Errorf("UTF-16 LE with multibyte chars should be converted correctly\ngot:  %q\nwant: %q", got, text)
	}
}
