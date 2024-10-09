package main

import (
	"log"

	"github.com/Charlie-Root/smcli/cmd"
	"github.com/Charlie-Root/smcli/config"
	"github.com/Charlie-Root/smcli/host"
)

func main() {
	config.LoadConfig("inventory.yaml")

	rootCmd := cmd.RootCmd
	rootCmd.PersistentFlags().BoolVarP(&cmd.Verbose, "verbose", "v", false, "verbose output")

	host.SetVerbose(cmd.Verbose)

	rootCmd.AddCommand(cmd.PowerCmd)
	rootCmd.AddCommand(cmd.MediaCmd)
	rootCmd.AddCommand(cmd.BootCmd)
	rootCmd.AddCommand(cmd.UserCmd)
	rootCmd.AddCommand(cmd.AclCmd)


	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
