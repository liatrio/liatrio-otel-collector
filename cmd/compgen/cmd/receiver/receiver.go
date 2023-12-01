package cmd

import (
	"log"
	"strings"

	common "github.com/liatrio/compgen/cmd/common"
	"github.com/spf13/cobra"
)

// ReceiverCmd represents the receiver command
var ReceiverCmd = &cobra.Command{
	Use:   "receiver",
	Short: "Build a new Open Telemetry receiver component",
	Long:  `Build a new Open Telemetry receiver component`,
	Run:   run,
}

func init() {
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// receiverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// receiverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func run(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		log.Fatal("A receiver name is required but was not supplied.")
	}
	name := args[0]

	shortName := name[strings.LastIndex(name, "/")+1:]
	modulePath := common.PackageDir + "/receiver/" + shortName

	common.InitNewModule(modulePath, name)
	data := common.TemplateData{Name: shortName, PackageName: name}
	common.RenderTemplates("cmd/receiver/templates", modulePath, data)
	common.CompleteModule(modulePath)
}
