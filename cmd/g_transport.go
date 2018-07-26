package cmd

import (
	"path/filepath"
	"strings"

	"ezrpro.com/micro/kit/pkg/cst"
	"ezrpro.com/micro/kit/pkg/generator"
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
		transportType := viper.GetString("g_t_transport_type")
		sourceFile := viper.GetString("g_t_source_file")
		if sourceFile == "" {
			logrus.Error("You must provide a source file for analyze of ast")
			return
		}

		cst, err := cst.New(sourceFile)
		if err != nil {
			logrus.Error(err)
			return
		}

		var transportTypes []string
		if transportType == "all" {
			transportTypes = AllTransportTypes
		} else {
			transportTypes = strings.Split(transportType, ",")
		}

		for _, tt := range transportTypes {
			transportType := strings.TrimSpace(strings.ToLower(tt))
			err = generateTransport(cst, transportType)
			if err != nil {
				logrus.Error(err)
				return
			}
		}

	},
}

func generateTransportFuncs(transportTypes ...string) []GenerateFunc {
	var funcs = make([]GenerateFunc, len(transportTypes))
	for i, _ := range transportTypes {
		transportType := transportTypes[i]
		funcs[i] = func(cst cst.ConcreteSyntaxTree) error {
			return generateTransport(cst, transportType)
		}
	}
	return funcs
}

func generateTransport(cst cst.ConcreteSyntaxTree, transportType string) error {
	var (
		gen           = generator.NoopGenerator
		transportPath = utils.GetTransportFilePath(cst.PackageName())
		filename      = filepath.Join(transportPath, transportType+".go")
	)

	switch strings.ToLower(transportType) {
	case "grpc":
		file, err := createFile(filename)
		if err != nil {
			return err
		}
		defer file.Close()

		gen = transport.NewTransportGenerator(
			cst,
			transport.WithWriter(file),
			transport.WithTemplateConfig(
				generator.NewTemplateConfig(cst),
			),
		)

		err = gen.Generate()
		if err != nil {
			return err
		}

		formatAndGoimports(filename)
	case "thrift":
	case "http":

	}

	return nil
}

func init() {
	generateCmd.AddCommand(transportCmd)

	transportCmd.Flags().StringP("source", "s", "", "Need to analyze the source file of ast")
	viper.BindPFlag("g_t_source_file", transportCmd.Flags().Lookup("source"))

	transportCmd.Flags().StringP("transport", "t", "grpc", "Transport type(all, grpc, thrift, http)")
	viper.BindPFlag("g_t_transport_type", transportCmd.Flags().Lookup("transport"))
}
