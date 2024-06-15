package cmd

import (
	"github.com/spf13/cobra"
)

var Verbose bool

var RootCmd = &cobra.Command{
	Use: "sm-cli",
}
