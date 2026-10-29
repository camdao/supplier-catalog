package cmd

import (
	"catalog/internal/service"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var changesUpdateID string

var changesCmd = &cobra.Command{
	Use:   "changes",
	Short: "View change history",
}

var changesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List changes from a catalog update",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := service.NewChangesService()

		// If no update-id given, use the latest update
		uid := changesUpdateID
		if uid == "" {
			latest, err := svc.GetLatestUpdate()
			if err != nil {
				return err
			}
			if latest == nil {
				fmt.Println("No updates found.")
				return nil
			}
			uid = latest.ID
			fmt.Printf("Showing changes for latest update: %s\n\n", uid)
		}

		changes, err := svc.ListChanges(uid)
		if err != nil {
			return err
		}

		if len(changes) == 0 {
			fmt.Println("No changes recorded.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "BATCH\tPRODUCT\tCHANGE\tVERSION\tDETAILS\n")
		fmt.Fprintf(w, "-----\t-------\t------\t-------\t-------\n")
		for _, c := range changes {
			versionStr := fmt.Sprintf("%d→%d", c.OldVersion, c.NewVersion)
			if c.OldVersion == 0 {
				versionStr = fmt.Sprintf("→%d", c.NewVersion)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				c.BatchID, c.ProductID, c.ChangeType, versionStr, c.Details)
		}
		w.Flush()

		// Summary
		added, updated, withdrawn, skipped := 0, 0, 0, 0
		for _, c := range changes {
			switch c.ChangeType {
			case "added":
				added++
			case "updated":
				updated++
			case "withdrawn":
				withdrawn++
			case "skipped_older":
				skipped++
			}
		}
		fmt.Printf("\nSummary: %d added, %d updated, %d withdrawn, %d skipped (older)\n",
			added, updated, withdrawn, skipped)

		return nil
	},
}

func init() {
	changesListCmd.Flags().StringVar(&changesUpdateID, "update-id", "", "update ID to show changes for (default: latest)")

	changesCmd.AddCommand(changesListCmd)
	rootCmd.AddCommand(changesCmd)
}
