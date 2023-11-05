package client

import "github.com/urfave/cli/v2"

var (
	flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "endpoint",
			Value: "ws://127.0.0.1:18888/ws",
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
	}
)

func runClient(c *cli.Context) error {
	h := NewBlinkClient(c.String("endpoint"))
	return h.Run()
}
