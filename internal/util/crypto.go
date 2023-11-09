package util

import (
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/crypto/chacha20poly1305"
)

var (
	ErrInvalidCipherText = errors.New("invalid cipher text")
)

func MD5(v []byte) string {
	return strings.ToLower(fmt.Sprintf("%x", md5.Sum(v)))
}

func MD5Str(v string) string {
	return strings.ToLower(fmt.Sprintf("%x", md5.Sum([]byte(v))))
}

func Chacha20Encrypt(aeadKey, plainText []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(aeadKey)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aead.NonceSize())
	_, _ = rand.Read(nonce) // 生成随机 nonce

	encrypted := aead.Seal(nil, nonce, plainText, nil)
	encrypted = append(nonce, encrypted...) // 将 nonce 和密文组合
	return encrypted, nil
}

func Chacha20Decrypt(aeadKey, cipherText []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(aeadKey)
	if err != nil {
		return nil, err
	}
	if len(cipherText) <= aead.NonceSize() {
		return nil, ErrInvalidCipherText
	}

	nonce := cipherText[:aead.NonceSize()]
	decrypted, err := aead.Open(nil, nonce, cipherText[aead.NonceSize():], nil)
	if err != nil {
		return nil, err
	}
	return decrypted, nil
}
