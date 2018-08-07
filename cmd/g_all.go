package cmd

import (
	"ezrpro.com/micro/kit/pkg/cst"
	"ezrpro.com/micro/kit/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type GenerateFunc func(cst cst.ConcreteSyntaxTree, serviceSuffix string) error

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
			ctree    cst.ConcreteSyntaxTree
			err      error
		)
		if !utils.IsProtobufSourceFile(sourceFile) {
			ctree, err = cst.New(sourceFile)
			genFuncs = []GenerateFunc{
				generateProtobuf,
				generateEndpoint,
				generateTransport,
				generateServer,
				generateClient,
			}
		} else {
			pkg := viper.GetString("g_a_package")
			if pkg == "" {
				logrus.Error("You must provide a package name for generate code")
				return
			}
			ctree, err = cst.New(
				sourceFile,
				cst.WithPackageName(pkg),
			)
			genFuncs = []GenerateFunc{
				generateEndpoint,
				generateTransport,
				generateServer,
				generateClient,
			}
		}
		if err != nil {
			logrus.Error(err)
			return
		}

		for _, genFunc := range genFuncs {
			if err = genFunc(ctree, utils.SelectServiceSuffix(sourceFile)); err != nil {
				logrus.Fatal(err)
			}
		}

	},
}

func init() {
	generateCmd.AddCommand(allCmd)

	allCmd.Flags().StringP("source", "s", "", "Need to analyze the source file of ast")
	allCmd.Flags().StringP("pkg", "p", "", "If you want to replace package of source file ")
	viper.BindPFlag("g_a_package", allCmd.Flags().Lookup("pkg"))
	viper.BindPFlag("g_a_source_file", allCmd.Flags().Lookup("source"))
}
