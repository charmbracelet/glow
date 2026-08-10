package utils

import (
	"encoding/binary"
	"unicode/utf16"
)

// DecodeUTF16BOM converts UTF-16 text with a byte-order mark to UTF-8.
// Data without a UTF-16 BOM, including malformed UTF-16 with an odd payload,
// is returned unchanged.
func DecodeUTF16BOM(data []byte) []byte {
	if len(data) < 2 || (len(data)-2)%2 != 0 {
		return data
	}

	var order binary.ByteOrder
	switch {
	case data[0] == 0xff && data[1] == 0xfe:
		order = binary.LittleEndian
	case data[0] == 0xfe && data[1] == 0xff:
		order = binary.BigEndian
	default:
		return data
	}

	codeUnits := make([]uint16, (len(data)-2)/2)
	for i := range codeUnits {
		codeUnits[i] = order.Uint16(data[2+i*2:])
	}
	return []byte(string(utf16.Decode(codeUnits)))
}
