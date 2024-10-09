package cmd

import (
	"github.com/Charlie-Root/smcli/host"
	"github.com/spf13/cobra"
)

var UserCmd = &cobra.Command{
	Use:   "user <username> <password> <host>",
	Short: "Add new user",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		
		hostName := args[2]

		host.SelectHost(hostName)
		host.ServerCreateUser(args[0], args[1])
	},
}
