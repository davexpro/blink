package util

import "github.com/bytedance/gopkg/lang/fastrand"

const (
	letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[fastrand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}
