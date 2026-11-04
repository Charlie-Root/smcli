package cmd

import (
	"fmt"

	"github.com/Charlie-Root/smcli/host"
	"github.com/spf13/cobra"
)

var BootCmd = &cobra.Command{
	Use:       "boot [cd|pxe] [host...]",
	Short:     "Boot from CD or PXE once (UEFI)",
	Args:      cobra.MinimumNArgs(1),
	ValidArgs: []string{"cd", "pxe"},
	RunE: func(cmd *cobra.Command, args []string) error {
		target, hosts := args[0], args[1:]
		if target != "cd" && target != "pxe" {
			return fmt.Errorf("unknown boot target %q (want cd or pxe)", target)
		}
		return run(hosts, true, func(c *host.Client) (host.Result, error) { return c.BootOnce(target) })
	},
}
