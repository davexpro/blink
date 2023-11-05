package main

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/davexpro/blink"
	"github.com/davexpro/blink/internal/client"
)

var (
	date  string
	magic string
)

func main() {
	// init cli
	app := &cli.App{
		Name:     "blink-cli",
		Usage:    "./blink-cli run",
		Version:  blink.Version + " (" + magic + ") " + date,
		Writer:   os.Stdout,
		Commands: client.Commands,
	}

	// init rand seed
	rand.New(rand.NewSource(time.Now().UnixNano()))

	// run the cli
	err := app.Run(os.Args)
	if err != nil {
		log.Println(err.Error())
	}
}
