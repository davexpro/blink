package main

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/davexpro/blink"
	"github.com/davexpro/blink/internal/server"
)

var (
	date  string
	magic string
)

func main() {
	log.Printf("blink server magic: %s date: %s", magic, date)
	// init cli
	app := &cli.App{
		Name:     "blink-srv",
		Usage:    "./blink-srv run",
		Version:  blink.Version + " (" + magic + ") " + date,
		Writer:   os.Stdout,
		Commands: server.Commands,
	}

	// init rand seed
	rand.New(rand.NewSource(time.Now().UnixNano()))

	// run the cli
	err := app.Run(os.Args)
	if err != nil {
		log.Println(err.Error())
	}
}
