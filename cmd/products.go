package cmd

import (
	"catalog/internal/service"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	productCategory      string
	productShowWithdrawn bool
)

var productsCmd = &cobra.Command{
	Use:   "products",
	Short: "Browse the local product catalog",
}

var productsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List products in the catalog",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := service.NewProductService()

		products, err := svc.ListProducts(productCategory, productShowWithdrawn)
		if err != nil {
			return err
		}

		if len(products) == 0 {
			fmt.Println("No products found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID\tNAME\tPRICE\tCATEGORY\tVERSION\tSTATUS\n")
		fmt.Fprintf(w, "--\t----\t-----\t--------\t-------\t------\n")
		for _, p := range products {
			status := "active"
			if p.Withdrawn {
				status = "WITHDRAWN"
			}
			fmt.Fprintf(w, "%s\t%s\t$%.2f\t%s\tv%d\t%s\n",
				p.ID, p.Name, float64(p.Price)/100, p.Category, p.Version, status)
		}
		w.Flush()

		fmt.Printf("\nTotal: %d products\n", len(products))
		return nil
	},
}

var productsShowCmd = &cobra.Command{
	Use:   "show [product-id]",
	Short: "Show details for a single product",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := service.NewProductService()

		p, err := svc.GetProduct(args[0])
		if err != nil {
			return err
		}
		if p == nil {
			return fmt.Errorf("product %s not found", args[0])
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID:\t%s\n", p.ID)
		fmt.Fprintf(w, "Name:\t%s\n", p.Name)
		fmt.Fprintf(w, "Description:\t%s\n", p.Description)
		fmt.Fprintf(w, "Price:\t$%.2f\n", float64(p.Price)/100)
		fmt.Fprintf(w, "Category:\t%s\n", p.Category)
		fmt.Fprintf(w, "Version:\t%d\n", p.Version)
		status := "Active"
		if p.Withdrawn {
			status = "WITHDRAWN"
		}
		fmt.Fprintf(w, "Status:\t%s\n", status)
		fmt.Fprintf(w, "Created:\t%s\n", p.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(w, "Updated:\t%s\n", p.UpdatedAt.Format("2006-01-02 15:04:05"))
		w.Flush()
		return nil
	},
}

func init() {
	productsListCmd.Flags().StringVar(&productCategory, "category", "", "filter by category")
	productsListCmd.Flags().BoolVar(&productShowWithdrawn, "withdrawn", false, "include withdrawn products")

	productsCmd.AddCommand(productsListCmd)
	productsCmd.AddCommand(productsShowCmd)
	rootCmd.AddCommand(productsCmd)
}
