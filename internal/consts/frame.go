package consts

type Frame struct {
	Body      []byte `json:"body"`
	Timestamp int64  `json:"timestamp"`
	Nonce     string `json:"nonce"`
	Sign      string `json:"sign"`
}
