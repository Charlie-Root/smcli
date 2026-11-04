package cmd

import (
	"time"

	"github.com/Charlie-Root/smcli/host"
	"github.com/spf13/cobra"
)

var InfoCmd = &cobra.Command{
	Use:   "info [host...]",
	Short: "Show model, serial, BIOS/BMC firmware, power state and health",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(args, false, func(c *host.Client) (host.Result, error) { return c.Info() })
	},
}

var sensorsProblemsOnly bool

var SensorsCmd = &cobra.Command{
	Use:   "sensors [host...]",
	Short: "Show temperatures, fans, PSUs and voltages",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(args, false, func(c *host.Client) (host.Result, error) { return c.Sensors(sensorsProblemsOnly) })
	},
}

var (
	selSince   time.Duration
	selLimit   int
	selManager bool
)

var SelCmd = &cobra.Command{
	Use:   "sel [host...]",
	Short: "Show the hardware event log, newest first",
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(args, false, func(c *host.Client) (host.Result, error) { return c.SEL(selSince, selLimit, selManager) })
	},
}

func init() {
	SensorsCmd.Flags().BoolVar(&sensorsProblemsOnly, "problems", false, "only show sensors that are not OK")
	SelCmd.Flags().DurationVar(&selSince, "since", 24*time.Hour, "only entries newer than this (0 = all)")
	SelCmd.Flags().IntVar(&selLimit, "limit", 50, "maximum entries per host (0 = all)")
	SelCmd.Flags().BoolVar(&selManager, "manager", false, "also read the BMC's own logs")
}
