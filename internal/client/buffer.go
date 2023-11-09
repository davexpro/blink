package client

import (
	"bytes"
	"sync"
)

const (
	_MaxBuffer = 1048576 // 1MB buffer size
)

var (
	bufferPool = sync.Pool{}
)

func newBuffer() *bytes.Buffer {
	if ret := bufferPool.Get(); ret != nil {
		return ret.(*bytes.Buffer)
	} else {
		return bytes.NewBuffer(make([]byte, 0, _MaxBuffer))
	}
}

func freeBuffer(p *bytes.Buffer) {
	p.Reset()
	bufferPool.Put(p)
}
