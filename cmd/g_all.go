package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type GenerateFunc func(sourceFile string) error

var allCmd = &cobra.Command{
	Use:     "all",
	Short:   "generate all source code of go-kit",
	Aliases: []string{"a"},
	Run: func(cmd *cobra.Command, args []string) {
		sourceFile := viper.GetString("g_a_source_file")
		if sourceFile == "" {
			logrus.Error("You must provide a source file for analyze of ast")
			return
		}

		var (
			genFuncs []GenerateFunc
			err      error
		)
		genFuncs = []GenerateFunc{
			generateProtobuf,
			generateEndpoint,
			generateTransport,
			generateServer,
			generateClient,
		}
		if err != nil {
			logrus.Error(err)
			return
		}

		for _, genFunc := range genFuncs {
			if err = genFunc(sourceFile); err != nil {
				logrus.Fatal(err)
			}
		}

	},
}

func init() {
	generateCmd.AddCommand(allCmd)

	allCmd.Flags().StringP("source", "s", "", "Source file defined by the service interface")
	allCmd.Flags().StringP("pkg", "p", "", "If you want to replace package of source file ")
	viper.BindPFlag("g_a_package", allCmd.Flags().Lookup("pkg"))
	viper.BindPFlag("g_a_source_file", allCmd.Flags().Lookup("source"))
}
