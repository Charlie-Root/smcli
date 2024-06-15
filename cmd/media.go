package cmd

import (
	"fmt"

	"github.com/Charlie-Root/smcli/host"
	"github.com/spf13/cobra"
)

var MediaCmd = &cobra.Command{
	Use:   "media [insert|eject|status]",
	Short: "Manage virtual media",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		action := args[0]
		hostName := args[1]

		host.SelectHost(hostName)
		host.GetRedfishPath()

		switch action {
		case "insert":
			host.ServerVirtualMediaInsert()
		case "eject":
			host.ServerVirtualMediaEject()
		case "status":
			host.ServerVirtualMediaStatus()
		default:
			fmt.Println("Unknown media command:", action)
		}
	},
}
