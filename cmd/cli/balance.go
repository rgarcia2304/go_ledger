package main
import(
	"github.com/spf13/cobra"
	"fmt"
	"context"
	"github.com/google/uuid"
)
var(
	accountID string
)

var getBalanceCmd = &cobra.Command{
	Use: "get-balance",
	Aliases: []string{"gb"},
	Short: "Get balance for this account",
	RunE: func(cmd *cobra.Command, args []string) error{
		parsedID, err := uuid.Parse(accountID)
		if err != nil{
			return err
		}
		bal, err := svc.GetBalance(context.Background(), parsedID)
		if err != nil{
			return err
		}
		fmt.Printf("Account \nID: %s\nAccount Balance: %v\n", accountID, bal)
		return nil

	},
}

func init(){
	getBalanceCmd.Flags().StringVar((&accountID), "accountID", "", "Account ID (required)")
    	getBalanceCmd.MarkFlagRequired("accountID")
    	rootCmd.AddCommand(getBalanceCmd)
}
