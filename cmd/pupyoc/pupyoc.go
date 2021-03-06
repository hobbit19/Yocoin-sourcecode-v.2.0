// Authored and revised by YOC team, 2017-2018
// License placeholder #1

// puppyoc is a command to assemble and maintain private networks.
package main

import (
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/Yocoin15/Yocoin_Sources/log"
	"gopkg.in/urfave/cli.v1"
)

// main is just a boring entry point to set up the CLI app.
func main() {
	app := cli.NewApp()
	app.Name = "puppyoc"
	app.Usage = "assemble and maintain private YoCoin networks"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "network",
			Usage: "name of the network to administer (no spaces or hyphens, please)",
		},
		cli.IntFlag{
			Name:  "loglevel",
			Value: 3,
			Usage: "log level to emit to the screen",
		},
	}
	app.Action = func(c *cli.Context) error {
		// Set up the logger to print everything and the random generator
		log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(c.Int("loglevel")), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))
		rand.Seed(time.Now().UnixNano())

		network := c.String("network")
		if strings.Contains(network, " ") || strings.Contains(network, "-") {
			log.Crit("No spaces or hyphens allowed in network name")
		}
		// Start the wizard and relinquish control
		makeWizard(c.String("network")).run()
		return nil
	}
	app.Run(os.Args)
}
