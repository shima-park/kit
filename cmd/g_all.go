package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"ezrpro.com/micro/kit/pkg/cst"
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

		var (
			genFuncs []GenerateFunc
			err      error
		)
		// 如果使用的接口定义是proto生成的pb.go,则先分析pb.go
		// 找出service和方法定义，通过该信息生成service.go
		// 再向下生成其他组件
		if utils.IsProtobufSourceFile(sourceFile) {
			pbGoFilePath := sourceFile
			tree, err := cst.New(sourceFile)
			if err != nil {
				logrus.Error("Your source file has an error", err)
				return
			}
			var (
				serviceName string
				pbSvcSuffix = utils.GetProtobufServiceSuffix()
				methods     []string
			)
			for _, iface := range tree.Interfaces() {
				if strings.HasSuffix(iface.Name, pbSvcSuffix) {
					serviceName = strings.ToLower(strings.TrimSuffix(iface.Name, pbSvcSuffix))
					for _, method := range iface.Methods {
						methods = append(methods, method.Name)
					}

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
			newService(serviceName, methods, tree)
			// 拼接service.go目录
			sourcePath := utils.GetServiceFilePath(serviceName)
			// 替换sourceFile
			sourceFile = filepath.Join(sourcePath, fmt.Sprintf("%s.go", service.ServiceTemplate.String()))

			pbGoABSPath, err := filepath.Abs(pbGoFilePath)
			if err != nil {
				logrus.Error("failed to get abs path of pb.go", err)
				return
			}

			// 将读取的pb文件路径设置入全局读取protobuf的配置中
			// 在后续生成的文件中，将pb的导入目录设置为该目录
			utils.SetProtobufPath(utils.GetImportPathByFileAbsPath(pbGoABSPath))
			tg := &TransportGenerator{
				pbGoFilePath: pbGoFilePath,
			}
			genFuncs = []GenerateFunc{
				generateEndpoint,
				tg.generateTransport,
				generateServer,
				generateClient,
			}
		} else {
			tg := &TransportGenerator{}
			genFuncs = []GenerateFunc{
				generateProtobuf,
				generateEndpoint,
				tg.generateTransport,
				generateServer,
				generateClient,
			}
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
