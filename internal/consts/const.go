package consts

import "time"

const (
	BlinkIdentifier = "blink v1 R!CH dave@suprich.org"

	HeaderClientKey     = "x-blink-key"
	HeaderClientVersion = "x-blink-version"

	HTTPAbortMessage = "404 page not found"

	HeartbeatDuration = time.Second * 15
)
