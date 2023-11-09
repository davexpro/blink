package client

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"

	"github.com/urfave/cli/v2"

	"github.com/davexpro/blink/internal/pkg/blog"
)

var (
	flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "endpoint",
			Value: "ws://127.0.0.1:18888/ws",
		},
		&cli.StringFlag{
			Name:    "server_key",
			Aliases: []string{"k", "srv_key"},
			Value:   "MCowBQYDK2VwAyEACubq6oo/fFmvt7rer0MTYP2neD/WJn5E7ILUxypXNbk=",
		},
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c", "conf"},
			Value:   "client.json",
		},
		&cli.IntFlag{
			Name:    "threads",
			Aliases: []string{"t"},
			Value:   32,
			Usage:   "",
		},
	}
	Commands = []*cli.Command{
		{
			Name:        "run",
			Usage:       "",
			UsageText:   "",
			Description: "",
			Action:      runClient,
			Flags:       flags,
		},
		{
			Name:        "install",
			Usage:       "",
			UsageText:   "",
			Description: "",
			Action:      installClient,
			Flags:       flags,
		},
	}
)

func runClient(c *cli.Context) error {
	// 1. unmarshal server key
	srvKeyStr := c.String("server_key")
	srvKeyBytes, err := base64.StdEncoding.DecodeString(srvKeyStr)
	if err != nil {
		return err
	}

	srvKeyAny, err := x509.ParsePKIXPublicKey(srvKeyBytes)
	if err != nil {
		return err
	}

	srvKey, ok := srvKeyAny.(ed25519.PublicKey)
	if !ok || len(srvKey) <= 0 {
		blog.Errorf("invalid server key, should be ed25519 public key")
		return nil
	}

	h := NewBlinkClient(c.String("endpoint"), srvKey)
	return h.Run()
}

func installClient(*cli.Context) error {
	return nil
}
