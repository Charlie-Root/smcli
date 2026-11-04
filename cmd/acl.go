package cmd

import (
	"fmt"
	"strconv"

	"github.com/Charlie-Root/smcli/host"
	"github.com/spf13/cobra"
)

var AclCmd = &cobra.Command{
	Use:   "acl",
	Short: "Manage the BMC IP access control list (Supermicro)",
}

var EnableAclCmd = &cobra.Command{
	Use:   "enable [host...]",
	Short: "Enable IP access control",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(args, true, func(c *host.Client) (host.Result, error) { return c.EnableACL() })
	},
}

var AddAclCmd = &cobra.Command{
	Use:   "add <address> <prefix> <policy> [host...]",
	Short: "Add an IP access control rule",
	Args:  cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		address, policy, hosts := args[0], args[2], args[3:]
		prefix, err := strconv.Atoi(args[1])
		if err != nil || prefix < 0 || prefix > 128 {
			return fmt.Errorf("invalid prefix length %q", args[1])
		}
		return run(hosts, true, func(c *host.Client) (host.Result, error) { return c.AddACL(address, prefix, policy) })
	},
}

func init() {
	AclCmd.AddCommand(EnableAclCmd, AddAclCmd)
}
