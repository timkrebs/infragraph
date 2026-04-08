package cmd

import (
	"log"
	"os"

	cli "github.com/timkrebs/gocli"
)

func Run() {
	ui := &cli.ConcurrentUi{
		Ui: &cli.ColoredUi{
			InfoColor:  cli.UiColorGreen,
			ErrorColor: cli.UiColorRed,
			WarnColor:  cli.UiColorYellow,
			Ui: &cli.BasicUi{
				Reader:      os.Stdin,
				Writer:      os.Stdout,
				ErrorWriter: os.Stderr,
			},
		},
	}

	c := cli.NewCLI("infragraph", "0.1.0")
	c.Args = os.Args[1:]
	c.HelpWriter = os.Stdout

	c.Commands = map[string]cli.CommandFactory{
		"server start":   func() (cli.Command, error) { return &ServerStartCmd{Ui: ui}, nil },
		"server stop":    func() (cli.Command, error) { return &ServerStopCmd{Ui: ui}, nil },
		"status":         func() (cli.Command, error) { return &StatusCmd{Ui: ui}, nil },
		"resources list": func() (cli.Command, error) { return &ResourcesListCmd{Ui: ui}, nil },
		"graph query":    func() (cli.Command, error) { return &GraphQueryCmd{Ui: ui}, nil },
		"graph impact":   func() (cli.Command, error) { return &GraphImpactCmd{Ui: ui}, nil },
		"db migrate":     func() (cli.Command, error) { return &DbMigrateCmd{Ui: ui}, nil },
	}

	exitStatus, err := c.Run()
	if err != nil {
		log.Println(err)
	}
	os.Exit(exitStatus)
}
