package cmd

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	common "github.com/liatrio/compgen/cmd/common"
	"github.com/spf13/cobra"
)

var shortDescription = "Build a new Open Telemetry receiver component"
var longDescription = `
receiverName: A full module path (https://go.dev/ref/mod#glos-module-path)
  E.g. 'github.com/liatrio/liatrio-otel-collector/pkg/receiver/myreceiver'

outputDir: A full or relative path to a directory that contains receivers
	E.g. receiver/`

// ReceiverCmd represents the receiver command
var ReceiverCmd = &cobra.Command{
	Use:   "receiver [flags] receiverName outputDir",
	Short: shortDescription,
	Long:  fmt.Sprint(shortDescription, "\n", longDescription),
	Args:  cobra.MinimumNArgs(2),
	Run:   run,
}

//go:embed templates/*
var Templates embed.FS

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
	packageName := args[0]
	name := packageName[strings.LastIndex(packageName, "/")+1:]
	modulePath := filepath.Join(args[1], name)

	err := os.MkdirAll(modulePath, os.ModePerm)
	if err != nil {
		panic(err)
	}

	data := common.TemplateData{Name: name, PackageName: packageName}
	common.Render(Templates, modulePath, data)
	common.Tidy(modulePath)
}
