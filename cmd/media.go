package cmd

import (
	"fmt"

	"github.com/Charlie-Root/smcli/host"
	"github.com/spf13/cobra"
)

var mediaISO string

var MediaCmd = &cobra.Command{
	Use:       "media [insert|eject|status] [host...]",
	Short:     "Manage virtual media",
	Args:      cobra.MinimumNArgs(1),
	ValidArgs: []string{"insert", "eject", "status"},
	RunE: func(cmd *cobra.Command, args []string) error {
		action, hosts := args[0], args[1:]
		switch action {
		case "status":
			return run(hosts, false, func(c *host.Client) (host.Result, error) { return c.MediaStatus() })
		case "eject":
			return run(hosts, true, func(c *host.Client) (host.Result, error) { return c.MediaEject() })
		case "insert":
			return run(hosts, true, func(c *host.Client) (host.Result, error) {
				iso := c.Host.ISOImage
				if mediaISO != "" {
					iso = mediaISO
				}
				return c.MediaInsert(iso)
			})
		}
		return fmt.Errorf("unknown media action %q (want insert, eject or status)", action)
	},
}

func init() {
	MediaCmd.Flags().StringVar(&mediaISO, "iso", "", "ISO image URL (overrides iso_image from the inventory)")
}
