package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"ezrpro.com/micro/kit/pkg/cst"
	"ezrpro.com/micro/kit/pkg/generator/client"
	"ezrpro.com/micro/kit/pkg/generator/service"
	"ezrpro.com/micro/kit/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var clientCmd = &cobra.Command{
	Use:     "client",
	Short:   "generate source code of go-kit client",
	Aliases: []string{"c"},
	Run: func(cmd *cobra.Command, args []string) {
		sourceFile := viper.GetString("g_c_source_file")
		if sourceFile == "" {
			logrus.Error("You must provide a source file for analyze of ast")
			return
		}

		err := generateClient(sourceFile)
		if err != nil {
			logrus.Error(err)
			return
		}
	},
}

func generateClient(sourceFile string) error {
	cst, err := cst.New(sourceFile)
	if err != nil {
		return err
	}
	serviceSuffix := utils.SelectServiceSuffix(sourceFile)
	baseServiceName := service.GetBaseServiceName(cst.PackageName(), serviceSuffix)
	clientPath := utils.GetClientFilePath(baseServiceName)
	clientPackageName := filepath.Base(clientPath)
	var options = []client.Option{
		client.WithBaseServiceName(baseServiceName),
		client.WithClientPackageName(clientPackageName),
		client.WithServiceSuffix(serviceSuffix),
	}
	for templateName, template := range client.TemplateMap {
		filename := filepath.Join(clientPath, fmt.Sprintf("%s.go", templateName.String()))

		file, err := createFile(filename)
		if err != nil {
			return errors.New("Create file " + filename + " error:" + err.Error())
		}
		defer GoimportsAndformat(filename)
		defer file.Close()

		options = append(options,
			client.WithReadWriter(
				templateName,
				strings.NewReader(template),
				file),
		)
	}

	gen := client.NewClientGenerator(
		cst,
		options...,
	)

	err = gen.Generate()
	if err != nil {
		logrus.Error(err)
		return err
	}

	return nil
}

func init() {
	generateCmd.AddCommand(clientCmd)

	clientCmd.Flags().StringP("source", "s", "", "Source file defined by the service interface")
	viper.BindPFlag("g_c_source_file", clientCmd.Flags().Lookup("source"))
}
