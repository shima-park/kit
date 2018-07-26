package cmd

import (
	"path/filepath"

	"ezrpro.com/micro/kit/pkg/cst"
	gen "ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/generator/endpoint"
	"ezrpro.com/micro/kit/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var endpointCmd = &cobra.Command{
	Use:     "endpoint",
	Short:   "generate source code of go-kit endpoint",
	Aliases: []string{"e"},
	Run: func(cmd *cobra.Command, args []string) {
		sourceFile := viper.GetString("g_e_source_file")
		if sourceFile == "" {
			logrus.Error("You must provide a source file for analyze of ast")
			return
		}

		cst, err := cst.New(sourceFile)
		if err != nil {
			logrus.Error(err)
			return
		}

		generateEndpoint(cst)
	},
}

func generateEndpoint(cst cst.ConcreteSyntaxTree) error {
	endpointPath := utils.GetEndpointFilePath(cst.PackageName())

	filename := filepath.Join(endpointPath, cst.PackageName()+".go")

	file, err := createFile(filename)
	if err != nil {
		logrus.Error("Create file ", filename, " error:", err)
		return err
	}
	defer file.Close()

	gen := endpoint.NewEndpointGenerator(
		cst,
		endpoint.WithWriter(file),
		endpoint.WithTemplateConfig(
			gen.NewTemplateConfig(cst),
		),
	)

	err = gen.Generate()
	if err != nil {
		logrus.Error(err)
		return err
	}

	formatAndGoimports(filename)
	return nil
}

func init() {
	generateCmd.AddCommand(endpointCmd)

	endpointCmd.Flags().StringP("source", "s", "", "Need to analyze the source file of ast")
	viper.BindPFlag("g_e_source_file", endpointCmd.Flags().Lookup("source"))
}
