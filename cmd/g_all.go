package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"ezrpro.com/micro/kit/pkg/cst"
	"ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/generator/service"
	"ezrpro.com/micro/kit/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type GenerateFunc func(sourceFile string) error

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

		if utils.IsProtobufSourceFile(sourceFile) {
			cst, err := cst.New(sourceFile)
			if err != nil {
				logrus.Error("Your source file has an error", err)
				return
			}
			var (
				serviceName string
				pbSvcSuffix = utils.GetProtobufServiceSuffix()
				methods     []string
				reqAndResps []generator.ReqAndResp
			)
			for _, iface := range cst.Interfaces() {
				if strings.HasSuffix(iface.Name, pbSvcSuffix) {
					serviceName = strings.ToLower(strings.TrimSuffix(iface.Name, pbSvcSuffix))
					for _, method := range iface.Methods {
						methods = append(methods, method.Name)
					}
					reqAndResps = generator.GetRequestAndResponseList(cst)
					break
				}
			}

			if serviceName == "" {
				logrus.Error("Can't find out service name")
				return
			}

			if len(methods) == 0 {
				logrus.Error("The service method must be provided")
				return
			}

			// 先通过pb.go生成service相关代码
			newService(serviceName, methods, reqAndResps)
			// 拼接service.go目录
			sourcePath := utils.GetServiceFilePath(serviceName)
			sourceFile = filepath.Join(sourcePath, fmt.Sprintf("%s.go", service.ServiceTemplate.String()))
		}

		var (
			genFuncs []GenerateFunc
			err      error
		)
		genFuncs = []GenerateFunc{
			generateProtobuf,
			generateEndpoint,
			generateTransport,
			generateServer,
			generateClient,
		}
		if err != nil {
			logrus.Error(err)
			return
		}

		for _, genFunc := range genFuncs {
			if err = genFunc(sourceFile); err != nil {
				logrus.Fatal(err)
			}
		}

	},
}

func init() {
	generateCmd.AddCommand(allCmd)

	allCmd.Flags().StringP("source", "s", "", "Source file defined by the service interface")
	allCmd.Flags().StringP("pkg", "p", "", "If you want to replace package of source file ")
	viper.BindPFlag("g_a_package", allCmd.Flags().Lookup("pkg"))
	viper.BindPFlag("g_a_source_file", allCmd.Flags().Lookup("source"))
}
