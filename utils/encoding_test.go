package utils

import "testing"

func TestDecodeUTF16BOM(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{
			name: "little endian",
			data: []byte{0xff, 0xfe, '#', 0, ' ', 0, 'T', 0, 'i', 0, 't', 0, 'l', 0, 'e', 0},
			want: "# Title",
		},
		{
			name: "big endian",
			data: []byte{0xfe, 0xff, 0, '#', 0, ' ', 0, 'T', 0, 'i', 0, 't', 0, 'l', 0, 'e'},
			want: "# Title",
		},
		{
			name: "utf8 remains unchanged",
			data: []byte("# Title"),
			want: "# Title",
		},
		{
			name: "odd utf16 payload remains unchanged",
			data: []byte{0xff, 0xfe, '#'},
			want: string([]byte{0xff, 0xfe, '#'}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(DecodeUTF16BOM(tt.data)); got != tt.want {
				t.Fatalf("DecodeUTF16BOM() = %q, want %q", got, tt.want)
			}
		})
	}
}
