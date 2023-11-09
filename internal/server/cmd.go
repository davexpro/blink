package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/davexpro/blink/internal/pkg/blog"
)

var (
	flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "endpoint",
			Value: "",
		},
		&cli.StringFlag{
			Name:    "server_key",
			Aliases: []string{"k"},
			Value:   "MC4CAQAwBQYDK2VwBCIEIM2V07AyaaI4AHt2bx+gYKGWv3Sbxd0PSMP58lDSiRcM",
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
		{
			Name:        "config",
			Usage:       "",
			UsageText:   "",
			Description: "",
			Action:      runConfig,
		},
	}
)

func runServer(c *cli.Context) error {
	// 1. unmarshal server key
	srvKeyStr := c.String("server_key")
	srvKeyBytes, err := base64.StdEncoding.DecodeString(srvKeyStr)
	if err != nil {
		return err
	}

	srvKeyAny, err := x509.ParsePKCS8PrivateKey(srvKeyBytes)
	if err != nil {
		return err
	}

	srvKey, ok := srvKeyAny.(ed25519.PrivateKey)
	if !ok || len(srvKey) <= 0 {
		blog.Errorf("invalid server key, should be ed25519 private key")
		return nil
	}

	// init blink server
	h := NewBlinkServer(srvKey)
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

// runConfig config generate helper
func runConfig(ctx *cli.Context) error {
	return nil
}
