package cmd

import (
	"path/filepath"

	"ezrpro.com/micro/kit/pkg/cst"
	"ezrpro.com/micro/kit/pkg/generator/protobuf"
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

		cst, err := cst.New(sourceFile)
		if err != nil {
			logrus.Error(err)
			return
		}

		err = generateProtobuf(cst)
		if err != nil {
			logrus.Error(err)
			return
		}
	},
}

func generateProtobuf(cst cst.ConcreteSyntaxTree) error {
	protoPath := utils.GetProtobufFilePath(cst.PackageName())

	filename := filepath.Join(protoPath, cst.PackageName()+".proto")

	file, err := createFile(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	gen := protobuf.NewProtobufGenerator(
		cst,
		protobuf.WithWriter(file),
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

func init() {
	generateCmd.AddCommand(grpcCmd)

	grpcCmd.Flags().StringP("source", "s", "", "Need to analyze the source file of ast")
	viper.BindPFlag("g_s_source_file", grpcCmd.Flags().Lookup("source"))
}
