package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"ezrpro.com/micro/kit/pkg/cst"
	"ezrpro.com/micro/kit/pkg/generator/endpoint"
	"ezrpro.com/micro/kit/pkg/generator/service"
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

		serviceSuffix := utils.SelectServiceSuffix(sourceFile)
		err = generateEndpoint(cst, serviceSuffix)
		if err != nil {
			logrus.Error(err)
			return
		}
	},
}

func generateEndpoint(cst cst.ConcreteSyntaxTree, serviceSuffix string) error {
	baseServiceName := service.GetBaseServiceName(cst.PackageName(), serviceSuffix)
	endpointPath := utils.GetEndpointFilePath(baseServiceName)
	endpointPackageName := filepath.Base(endpointPath)
	var options = []endpoint.Option{
		endpoint.WithBaseServiceName(baseServiceName),
		endpoint.WithEndpointPackageName(endpointPackageName),
		endpoint.WithServiceSuffix(serviceSuffix),
	}
	for templateName, template := range endpoint.TemplateMap {
		filename := filepath.Join(endpointPath, fmt.Sprintf("%s.go", templateName.String()))

		file, err := createFile(filename)
		if err != nil {
			logrus.Error("Create file ", filename, " error:", err)
			return err
		}
		defer formatAndGoimports(filename)
		defer file.Close()

		options = append(options,
			endpoint.WithReadWriter(
				templateName,
				strings.NewReader(template),
				file),
		)
	}

	gen := endpoint.NewEndpointGenerator(
		cst,
		options...,
	)

	err := gen.Generate()
	if err != nil {
		logrus.Error(err)
		return err
	}

	return nil
}

func init() {
	generateCmd.AddCommand(endpointCmd)

	endpointCmd.Flags().StringP("source", "s", "", "Need to analyze the source file of ast")
	viper.BindPFlag("g_e_source_file", endpointCmd.Flags().Lookup("source"))
}
