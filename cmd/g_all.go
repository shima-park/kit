package cmd

import (
	"ezrpro.com/micro/kit/pkg/cst"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type GenerateFunc func(cst cst.ConcreteSyntaxTree) error

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

		cst, err := cst.New(sourceFile)
		if err != nil {
			logrus.Error(err)
			return
		}

		genFuncs := []GenerateFunc{
			generateProtobuf,
			generateEndpoint,
		}
		genFuncs = append(genFuncs, generateTransportFuncs(AllTransportTypes...)...)

		for _, genFunc := range genFuncs {
			if err = genFunc(cst); err != nil {
				logrus.Error(err)
				return
			}
		}

	},
}

func init() {
	generateCmd.AddCommand(allCmd)

	allCmd.Flags().StringP("source", "s", "", "Need to analyze the source file of ast")
	viper.BindPFlag("g_a_source_file", allCmd.Flags().Lookup("source"))
}
