package consts

type Frame struct {
	Body      []byte `json:"body"`
	Timestamp int64  `json:"timestamp"`
	DeviceID  string `json:"device_id,omitempty"` // only client
	Nonce     string `json:"nonce,omitempty"`     // only server
	Signature string `json:"signature,omitempty"` // only server
}
