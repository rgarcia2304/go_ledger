package main
import(
	"github.com/rgarcia2304/go_ledger/internal/ledger"
	"github.com/spf13/cobra"
	"fmt"
	"context"
)
var(
	accountName string
	accountType string
	accountCurrency string
)

var createAccountCmd = &cobra.Command{
	Use: "create-account",
	Aliases: []string{"ca"},
	Short: "Creates a new ledger account",
	RunE: func(cmd *cobra.Command, args []string) error{
		acc, err := svc.CreateAccount(context.Background(), ledger.CreateAccountRequest{
			Name: accountName, 
			AccountType: accountType,
			Currency: accountCurrency,
		})

		if err != nil{
			return err
		}
		fmt.Printf("Account created successfully\nID: %s\nName %s\nType: %s\nCurrency: %s\n"			,acc.ID, acc.Name, acc.AccountType, acc.Currency)
		return nil

	},
}

func init(){
	createAccountCmd.Flags().StringVar(&accountName, "name", "", "Account name (required)")
	createAccountCmd.Flags().StringVar(&accountType, "type", "", "Account type: asset, liability, equity, revenue, expense (required)")
    	createAccountCmd.Flags().StringVar(&accountCurrency, "currency", "USD", "Account currency (required)")
    	createAccountCmd.MarkFlagRequired("name")
    	createAccountCmd.MarkFlagRequired("type")
    	createAccountCmd.MarkFlagRequired("currency")
    	rootCmd.AddCommand(createAccountCmd)
}
