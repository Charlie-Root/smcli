package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Charlie-Root/smcli/host"
	"github.com/spf13/cobra"
)

var UserCmd = &cobra.Command{
	Use:   "user <username> <password|-> [host...]",
	Short: "Add an Administrator user",
	Long:  `Add an Administrator user. Pass "-" as password to read it from stdin, which keeps it out of the process list.`,
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		username, password, hosts := args[0], args[1], args[2:]
		if password == "-" {
			line, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil && line == "" {
				return fmt.Errorf("reading password from stdin: %w", err)
			}
			password = strings.TrimRight(line, "\r\n")
		}
		if password == "" {
			return fmt.Errorf("empty password")
		}
		return run(hosts, true, func(c *host.Client) (host.Result, error) { return c.CreateUser(username, password) })
	},
}
