package main

import (
	"fmt"
	"sort"

	"github.com/elasticlic/els-cli/config"
	"github.com/urfave/cli"
)

// ELSCLI represents
type ELSCLI struct {
	ca     cli.App
	config config.Config
}

// NewELSCLI creates a new instance of the ELS CLI App. Call Run() to execute
// the App.
func NewELSCLI(ca cli.App, c config.Config, args []string) {
	a := &ELSCLI{
		ca:     ca,
		config: c,
	}
}

func (e *ELSCLI) initCLIAPP() {
	ca.Name = "els-cli"
	ca.Usage = "blah"
	app.Action = func(c *cli.Context) error {
		fmt.Printf("Hello %q", c.Args().Get(0))
		return nil
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "profile, p",
			Value:  "default",
			Usage:  "Use profile `PROFILE` from ~/.els-cli/credentials",
			EnvVar: "ELSCLI_PROFILE",
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))

}

// Run parses the command line arguments and tries to identify and execute a
// command.
func (e *ELSCLI) Run(cliArgs []string) {
	e.initCLIAPP()
	e.ca.Run(cliArgs)
}
