package cmd

import (
	"github.com/Charlie-Root/smcli/host"
	"github.com/spf13/cobra"
)

// Parent ACL command (no direct functionality)
var AclCmd = &cobra.Command{
	Use:   "acl",
	Short: "Manage ACL settings",
}

// Subcommand to enable ACL on the host
var EnableAclCmd = &cobra.Command{
	Use:   "enable <host>",
	Short: "Enable ACL on the specified host",
	Args:  cobra.ExactArgs(1), // Expect exactly one argument for the host
	Run: func(cmd *cobra.Command, args []string) {
		hostName := args[0]
		host.SelectHost(hostName)
		host.ServerEnableACL() // Call to enable ACL on the host
	},
}

// Subcommand to add an ACL entry
var AddAclCmd = &cobra.Command{
	Use:   "add <address> <prefix> <policy> <host>",
	Short: "Add new ACL entry",
	Args:  cobra.ExactArgs(4), // Expect address, prefix, policy, and host as arguments
	Run: func(cmd *cobra.Command, args []string) {
		address := args[0]
		prefix := args[1]
		policy := args[2]
		hostName := args[3]

		host.SelectHost(hostName)
		host.ServerCreateACL(address, prefix, policy) // Call to add ACL entry
	},
}

func init() {
	// Register subcommands under the acl command
	AclCmd.AddCommand(EnableAclCmd)
	AclCmd.AddCommand(AddAclCmd)

	// Register acl command to the root command (assuming RootCmd is the root cobra command)
	RootCmd.AddCommand(AclCmd)
}
