package util

import "github.com/bytedance/sonic"

func MustMarshalString(v interface{}) string {
	str, _ := sonic.MarshalString(v)
	return str
}
