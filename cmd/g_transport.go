package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"ezrpro.com/micro/kit/pkg/cst"
	"ezrpro.com/micro/kit/pkg/generator/service"
	"ezrpro.com/micro/kit/pkg/generator/transport"
	"ezrpro.com/micro/kit/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	AllTransportTypes = []string{"grpc", "thrift", "http"}
)

var transportCmd = &cobra.Command{
	Use:     "transport",
	Short:   "generate source code of go-kit transport",
	Aliases: []string{"t"},
	Run: func(cmd *cobra.Command, args []string) {
		//		transportType := viper.GetString("g_t_transport_type")
		sourceFile := viper.GetString("g_t_source_file")
		if sourceFile == "" {
			logrus.Error("You must provide a source file for analyze of ast")
			return
		}

		err := generateTransport(sourceFile)
		if err != nil {
			logrus.Error(err)
			return
		}
	},
}

func generateTransport(sourceFile string) error {
	cst, err := cst.New(sourceFile)
	if err != nil {
		return err
	}
	serviceSuffix := utils.SelectServiceSuffix(sourceFile)
	baseServiceName := service.GetBaseServiceName(cst.PackageName(), serviceSuffix)
	transportPath := utils.GetTransportFilePath(baseServiceName)
	transportPackageName := filepath.Base(transportPath)
	var options = []transport.Option{
		transport.WithBaseServiceName(baseServiceName),
		transport.WithTransportPackageName(transportPackageName),
		transport.WithServiceSuffix(serviceSuffix),
	}
	for templateName, template := range transport.TemplateMap {
		filename := filepath.Join(transportPath, fmt.Sprintf("%s.go", templateName.String()))

		file, err := createFile(filename)
		if err != nil {
			return errors.New("Create file " + filename + " error:" + err.Error())
		}
		defer GoimportsAndformat(filename)
		defer file.Close()

		options = append(options,
			transport.WithReadWriter(
				templateName,
				strings.NewReader(template),
				file),
		)
	}

	gen := transport.NewTransportGenerator(
		cst,
		options...,
	)

	err = gen.Generate()
	if err != nil {
		return err
	}

	return nil
}

func init() {
	generateCmd.AddCommand(transportCmd)

	transportCmd.Flags().StringP("source", "s", "", "Source file defined by the service interface")
	viper.BindPFlag("g_t_source_file", transportCmd.Flags().Lookup("source"))

	transportCmd.Flags().StringP("transport", "t", "grpc", "Transport type(all, grpc, thrift, http)")
	viper.BindPFlag("g_t_transport_type", transportCmd.Flags().Lookup("transport"))
}
