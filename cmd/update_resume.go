package cmd

import (
	"catalog/internal/dto"
	"catalog/internal/service"
	"catalog/thirt_part/supplier"
	"fmt"

	"github.com/spf13/cobra"
)

var simulateFailureResume bool

var updateResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume an incomplete catalog update",
	RunE: func(cmd *cobra.Command, args []string) error {
		sup := supplier.NewFakeSupplier()
		svc := service.NewUpdateService(sup)

		fmt.Println("Resuming catalog update...")
		result, err := svc.ResumeUpdate()
		if err != nil {
			return err
		}

		printUpdateResult(result)
		return nil
	},
}

func printUpdateResult(result *dto.UpdateResult) {
	fmt.Printf("\nUpdate %s completed with status: %s\n", result.UpdateID, result.Status)
	fmt.Printf("Total batches:    %d\n", result.TotalBatches)
	fmt.Printf("Completed:        %d\n", result.BatchesCompleted)
	fmt.Printf("Skipped:          %d\n", result.BatchesSkipped)
	fmt.Printf("\nProducts added:   %d\n", result.ProductsAdded)
	fmt.Printf("Products updated: %d\n", result.ProductsUpdated)
	fmt.Printf("Products withdrawn: %d\n", result.ProductsWithdrawn)
	fmt.Printf("Products skipped (older version): %d\n", result.ProductsSkippedOlder)
}

func init() {
	updateResumeCmd.Flags().BoolVar(&simulateFailureResume, "simulate-failure", false, "simulate a supplier outage on batch-003")
	updateCmd.AddCommand(updateResumeCmd)
}
