package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"ezrpro.com/micro/kit/pkg/cst"
	"ezrpro.com/micro/kit/pkg/generator/server"
	"ezrpro.com/micro/kit/pkg/generator/service"
	"ezrpro.com/micro/kit/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serverCmd = &cobra.Command{
	Use:     "server",
	Short:   "generate source code of go-kit server",
	Aliases: []string{"s"},
	Run: func(cmd *cobra.Command, args []string) {
		sourceFile := viper.GetString("g_s_source_file")
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
		err = generateServer(cst, serviceSuffix)
		if err != nil {
			logrus.Error(err)
			return
		}
	},
}

func generateServer(cst cst.ConcreteSyntaxTree, serviceSuffix string) error {
	baseServiceName := service.GetBaseServiceName(cst.PackageName(), serviceSuffix)
	serverPath := utils.GetServerFilePath(baseServiceName)
	serverPackageName := filepath.Base(serverPath)
	var options = []server.Option{
		server.WithBaseServiceName(baseServiceName),
		server.WithServerPackageName(serverPackageName),
		server.WithServiceSuffix(serviceSuffix),
	}
	for templateName, template := range server.TemplateMap {
		filename := filepath.Join(serverPath, fmt.Sprintf("%s.go", templateName.String()))

		file, err := createFile(filename)
		if err != nil {
			logrus.Error("Create file ", filename, " error:", err)
			return err
		}
		defer formatAndGoimports(filename)
		defer file.Close()

		options = append(options,
			server.WithReadWriter(
				templateName,
				strings.NewReader(template),
				file),
		)
	}

	gen := server.NewServerGenerator(
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
	generateCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringP("source", "s", "", "Need to analyze the source file of ast")
	viper.BindPFlag("g_s_source_file", serverCmd.Flags().Lookup("source"))
}
