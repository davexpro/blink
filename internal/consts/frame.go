package consts

import (
	"time"

	"github.com/bytedance/sonic"

	"github.com/davexpro/blink/internal/util"
)

type Frame struct {
	Body      []byte `json:"body"`
	Timestamp int64  `json:"timestamp"`
	Nonce     string `json:"nonce"`
	Signature string `json:"signature"`
}

func NewFrame(body interface{}) *Frame {
	bs, _ := sonic.Marshal(body)
	return &Frame{
		Body:      bs,
		Timestamp: time.Now().UnixMilli(),
		Nonce:     util.RandString(32),
		Signature: "",
	}
}

func (f *Frame) Marshal() ([]byte, error) {
	return sonic.Marshal(f)
}
