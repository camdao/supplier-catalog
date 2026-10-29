package cmd

import (
	"catalog/internal/service"
	"catalog/third_part/supplier"
	"fmt"

	"github.com/spf13/cobra"
)

var updateStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a new catalog update from the supplier",
	RunE: func(cmd *cobra.Command, args []string) error {

		sup := supplier.NewFakeSupplier()
		update := service.NewUpdateService(sup)

		fmt.Println("Starting catalog update...")
		result, err := update.StartUpdate()
		if err != nil {
			return err
		}

		printUpdateResult(result)
		return nil
	},
}

func init() {
	updateCmd.AddCommand(updateStartCmd)
}
