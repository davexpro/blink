package client

import "github.com/urfave/cli/v2"

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
			Action:      runClient,
			Flags:       flags,
		},
	}
)

func runClient(c *cli.Context) error {
	h := NewBlinkClient()
	return h.Run()
}
