package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"fmt"

	"github.com/urfave/cli/v2"
)

var (
	flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "endpoint",
			Value: "",
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
			Action:      runServer,
			Flags:       flags,
		},
		{
			Name:        "genkey",
			Usage:       "",
			UsageText:   "",
			Description: "",
			Action:      runGenKey,
			Flags:       flags,
		},
	}
)

func runServer(c *cli.Context) error {
	h := NewBlinkServer()
	return h.Run()
}

func runGenKey(*cli.Context) error {
	pub, pri, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	pubKey, _ := x509.MarshalPKIXPublicKey(pub)
	priKey, _ := x509.MarshalPKCS8PrivateKey(pri)

	fmt.Println("pubKey:", base64.StdEncoding.EncodeToString(pubKey))
	fmt.Println("priKey:", base64.StdEncoding.EncodeToString(priKey))
	return nil
}
