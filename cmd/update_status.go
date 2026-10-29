package cmd

import (
	"catalog/internal/service"
	"fmt"

	"github.com/spf13/cobra"
)

var updateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of the current or latest update",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := service.NewUpdateService(nil)

		// Try in-progress first, then latest
		update, err := svc.GetInProgressUpdate()
		if err != nil {
			return err
		}
		if update == nil {
			update, err = svc.GetLatestUpdate()
			if err != nil {
				return err
			}
		}
		if update == nil {
			fmt.Println("No updates found.")
			return nil
		}

		fmt.Printf("Update ID:    %s\n", update.ID)
		fmt.Printf("Status:       %s\n", update.Status)
		fmt.Printf("Started:      %s\n", update.StartedAt.Format("2006-01-02 15:04:05"))
		if update.CompletedAt != nil {
			fmt.Printf("Completed:    %s\n", update.CompletedAt.Format("2006-01-02 15:04:05"))
		}

		batches, err := svc.ListBatches(update.ID)
		if err != nil {
			return err
		}

		fmt.Printf("Total batches: %d\n", len(batches))
		completed := 0
		failed := 0
		pending := 0
		for _, b := range batches {
			switch b.Status {
			case "completed":
				completed++
			case "failed":
				failed++
			default:
				pending++
			}
		}
		fmt.Printf("  Completed:  %d\n", completed)
		fmt.Printf("  Failed:     %d\n", failed)
		fmt.Printf("  Pending:    %d\n", pending)

		if failed > 0 {
			fmt.Println("\nFailed batches:")
			for _, b := range batches {
				if b.Status == "failed" {
					fmt.Printf("  %s: %s\n", b.ID, b.Error)
				}
			}
		}

		return nil
	},
}

func init() {
	updateCmd.AddCommand(updateStatusCmd)
}
