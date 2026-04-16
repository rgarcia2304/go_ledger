package main 

import(
	"fmt"
	"os"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "ledger", 
	Short: "A simple way to do accounting",
	Long: `A double entry ledger that keeps track of 
	all expenses in your organization`,
	Run: func(cmd *cobra.Command, args []string){

	},
}


func Execute(){
	if err := rootCmd.Execute(); err != nil{
			fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}
