package cmd

import (
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Manage catalog updates",
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
