package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

type options struct {
	inventory string
	source    string
	rack      string
	site      string
	tag       string
	role      string
	parallel  int
	timeout   time.Duration
	output    string
	dryRun    bool
	yes       bool
	verbose   bool
}

var opts options

var RootCmd = &cobra.Command{
	Use:   "sm-cli",
	Short: "Manage Supermicro servers through their BMC (Redfish)",
	Long: `Manage Supermicro servers through their BMC (Redfish).

Hosts come from an inventory file (--source file, default) or from NetBox
(--source netbox, needs NETBOX_URL and NETBOX_TOKEN). Name hosts as arguments,
or select them from NetBox with --rack, --site, --tag or --role.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// ErrFailed means one or more hosts failed; the details were already printed.
var ErrFailed = errors.New("one or more hosts failed")

func init() {
	f := RootCmd.PersistentFlags()
	f.StringVar(&opts.inventory, "inventory", "inventory.yaml", `inventory file, "-" for stdin (--source file)`)
	f.StringVar(&opts.source, "source", envOr("SMCLI_SOURCE", "file"), "where hosts come from: file or netbox (env SMCLI_SOURCE)")
	f.StringVar(&opts.rack, "rack", "", "netbox: select devices in this rack")
	f.StringVar(&opts.site, "site", "", "netbox: select devices at this site (slug)")
	f.StringVar(&opts.tag, "tag", "", "netbox: select devices with this tag (slug)")
	f.StringVar(&opts.role, "role", "", "netbox: select devices with this role (slug)")
	f.IntVar(&opts.parallel, "parallel", 5, "hosts to work on at the same time")
	f.DurationVar(&opts.timeout, "timeout", 15*time.Second, "timeout per request")
	f.StringVarP(&opts.output, "output", "o", "text", "output format: text or json")
	f.BoolVar(&opts.dryRun, "dry-run", false, "show changes without sending them")
	f.BoolVarP(&opts.yes, "yes", "y", false, "allow changes on more than one host")
	f.BoolVarP(&opts.verbose, "verbose", "v", false, "log requests to stderr (passwords masked)")

	RootCmd.AddCommand(PowerCmd, BootCmd, MediaCmd, UserCmd, AclCmd, InfoCmd, SensorsCmd, SelCmd)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Execute runs the CLI and returns the process exit code.
func Execute(version string) int {
	RootCmd.Version = version
	if err := RootCmd.Execute(); err != nil {
		if !errors.Is(err, ErrFailed) {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		return 1
	}
	return 0
}
