package internal

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"unicode/utf16"

	"golang.org/x/crypto/md4"
)

func encDecInit(serverKey string) (cipher.Block, error) {
	if len(serverKey) == 0 {
		Fatal("invalid key", nil)
	}
	key := []byte(serverKey)
	return aes.NewCipher(key)
}

// Decrypt will decrypt a secret
func Decrypt(serverKey, item string) (string, error) {
	block, err := encDecInit(serverKey)
	if err != nil {
		return "", err
	}
	ciphertext, err := hex.DecodeString(item)
	if err != nil {
		return "", err
	}
	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return string(ciphertext), nil
}

// Encrypt will encrypt a secret
func Encrypt(serverKey, item string) (string, error) {
	block, err := encDecInit(serverKey)
	if err != nil {
		return "", err
	}
	bytes := []byte(item)
	ciphertext := make([]byte, aes.BlockSize+len(bytes))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], bytes)
	return hex.EncodeToString(ciphertext), nil
}

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
func MD4(item string) string {
	h := md4.New()
	h.Write(utf16le(item))
	return fmt.Sprintf("%x", string(h.Sum(nil)))
}
