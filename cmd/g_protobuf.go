package cmd

import (
	"path/filepath"

	"ezrpro.com/micro/kit/pkg/cst"
	"ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/generator/protobuf"
	"ezrpro.com/micro/kit/pkg/generator/service"
	"ezrpro.com/micro/kit/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var grpcCmd = &cobra.Command{
	Use:     "protobuf",
	Short:   "generate protobuf of go-kit grpc",
	Aliases: []string{"p"},
	Run: func(cmd *cobra.Command, args []string) {
		sourceFile := viper.GetString("g_s_source_file")
		if sourceFile == "" {
			logrus.Error("You must provide a source file for analyze of ast")
			return
		}

		err := generateProtobuf(sourceFile)
		if err != nil {
			logrus.Error(err)
			return
		}
	},
}

func generateProtobuf(sourceFile string) error {
	cst, err := cst.New(sourceFile)
	if err != nil {
		return err
	}
	serviceSuffix := utils.SelectServiceSuffix(sourceFile)
	baseServiceName := service.GetBaseServiceName(cst.PackageName(), serviceSuffix)
	protoPath := utils.GetProtobufFilePath(baseServiceName)
	filename := filepath.Join(protoPath, cst.PackageName()+".proto")

	file, err := createFile(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	gen := protobuf.NewProtobufGenerator(
		cst,
		protobuf.WithWriter(file),
		protobuf.WithServiceNameNormalizer(
			ServiceNameNormalizer{serviceSuffix: serviceSuffix},
		),
		protobuf.WithStructFilter(generator.DefaultStructFilter),
		protobuf.WithServiceSuffix(serviceSuffix),
	)

	err = gen.Generate()
	if err != nil {
		return err
	}

	err = generateProtobufGo(filename)
	if err != nil {
		return err
	}
	return nil
}

type ServiceNameNormalizer struct {
	serviceSuffix string
}

func (n ServiceNameNormalizer) Normalize(name string) string {
	if n.serviceSuffix == "" {
		n.serviceSuffix = utils.GetServiceSuffix()
	}
	return utils.ToCamelCase(service.GetBaseServiceName(name, n.serviceSuffix))
}

func init() {
	generateCmd.AddCommand(grpcCmd)

	grpcCmd.Flags().StringP("source", "s", "", "Need to analyze the source file of ast")
	viper.BindPFlag("g_s_source_file", grpcCmd.Flags().Lookup("source"))
}
