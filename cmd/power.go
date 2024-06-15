package cmd

import (
	"fmt"

	"github.com/Charlie-Root/smcli/host"
	"github.com/spf13/cobra"
)

var PowerCmd = &cobra.Command{
	Use:   "power [on|off|restart]",
	Short: "Manage power",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		action := args[0]
		hostName := args[1]

		host.SelectHost(hostName)
		host.GetRedfishPath()

		switch action {
		case "on":
			host.ServerPowerOn()
		case "off":
			host.ServerPowerOff()
		case "restart":
			host.ServerRestart()
		default:
			fmt.Println("Unknown power command:", action)
		}
	},
}
