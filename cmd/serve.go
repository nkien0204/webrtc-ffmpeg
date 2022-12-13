package cmd

import (
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the application",
	Run:   runServeCmd,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	//serveCmd.Flags().Int("system-crypt-cost", bcrypt.DefaultCost, "User bcrypt cost. Default is 10")
	//err := viper.BindPFlag("system-crypt-cost", serveCmd.Flags().Lookup("system-crypt-cost"))
	//if err != nil {
	//	panic(err)
	//}
}

// runServeCmd serves both gRPC and REST gateway.
// From now, this command will be used mainly for development.
func runServeCmd(cmd *cobra.Command, args []string) {
	err := cmd.Help()
	if err != nil {
		panic(err)
	}
}
