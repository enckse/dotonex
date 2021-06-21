package core

import (
	"fmt"
	"unicode/utf16"

	"golang.org/x/crypto/md4"
)

func utf16le(s string) []byte {
	codes := utf16.Encode([]rune(s))
	b := make([]byte, len(codes)*2)
	for i, r := range codes {
		b[i*2] = byte(r)
		b[i*2+1] = byte(r >> 8)
	}
	return b
}

// MD4 computes an md4 hash
func MD4(item string) (string, error) {
	h := md4.New()
	if _, err := h.Write(utf16le(item)); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", string(h.Sum(nil))), nil
}
