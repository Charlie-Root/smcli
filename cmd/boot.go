package cmd

import (
	"fmt"

	"github.com/Charlie-Root/smcli/host"
	"github.com/spf13/cobra"
)

var BootCmd = &cobra.Command{
	Use:   "boot [cd|pxe]",
	Short: "Set boot order",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		action := args[0]
		hostName := args[1]

		host.SelectHost(hostName)
		host.GetRedfishPath()

		switch action {
		case "cd":
			host.ServerSetBootOnceCD()
		case "pxe":
			host.ServerSetBootOncePXE()
		default:
			fmt.Println("Unknown boot command:", action)
		}
	},
}
