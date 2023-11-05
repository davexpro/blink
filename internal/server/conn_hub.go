package server

type ConnHub struct {
}

var globalHub = NewConnHub()

func NewConnHub() *ConnHub {
	return &ConnHub{}
}
