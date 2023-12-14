package cmd

import (
	"log"
	"os"

	receiver "github.com/liatrio/compgen/cmd/receiver"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "compgen",
	Short: "A tool for building new otel components.",
	Long:  `Compgen is a tool for building new receivers, processors, and exporters for Open Telemetry.`,
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	log.Default().SetFlags(log.Lshortfile)

	rootCmd.AddCommand(receiver.ReceiverCmd)
}
