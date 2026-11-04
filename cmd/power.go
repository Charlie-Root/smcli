package cmd

import (
	"fmt"

	"github.com/Charlie-Root/smcli/host"
	"github.com/spf13/cobra"
)

var PowerCmd = &cobra.Command{
	Use:   "power [on|off|restart|shutdown|status] [host...]",
	Short: "Show or change the power state",
	Long: `Show or change the power state.

  on        power on
  off       force power off
  restart   force restart
  shutdown  graceful shutdown (ACPI, the OS shuts down)
  status    show the power state`,
	Args:      cobra.MinimumNArgs(1),
	ValidArgs: []string{"on", "off", "restart", "shutdown", "status"},
	RunE: func(cmd *cobra.Command, args []string) error {
		action, hosts := args[0], args[1:]
		switch action {
		case "status":
			return run(hosts, false, func(c *host.Client) (host.Result, error) { return c.PowerStatus() })
		case "on", "off", "restart", "shutdown":
			return run(hosts, true, func(c *host.Client) (host.Result, error) { return c.Reset(action) })
		}
		return fmt.Errorf("unknown power action %q (want on, off, restart, shutdown or status)", action)
	},
}
