package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"ezrpro.com/micro/kit/pkg/generator/service"
	"ezrpro.com/micro/kit/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serviceCmd = &cobra.Command{
	Use:     "service",
	Short:   "Generate new service",
	Aliases: []string{"s"},
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := strings.ToLower(viper.GetString("ns_service"))
		if serviceName == "" {
			logrus.Error("You must provide a name for the service e.g.(./kit n s -s=add -p=addservice -m=Sum,Concat)")
			cmd.Help()
			return
		}

		methods := strings.Split(viper.GetString("ns_methods"), ",")
		if viper.GetString("ns_methods") == "" || len(methods) == 0 {
			logrus.Error("You must provide a method list e.g.(./kit n s -s=add -p=addservice -m=Sum,Concat)")
			cmd.Help()
			return
		}

		servicePath := utils.GetServiceFilePath(serviceName)

		var options = []service.Option{
			service.WithServiceName(serviceName),
			service.WithMethods(methods),
		}
		for templateName, template := range service.TemplateMap {
			filename := filepath.Join(servicePath, fmt.Sprintf("%s.go", templateName.String()))
			file, err := createFile(filename)
			if err != nil {
				logrus.Error(err)
				return
			}
			defer GoimportsAndformat(filename)
			defer file.Close()

			options = append(options,
				service.WithReadWriter(
					templateName,
					strings.NewReader(template),
					file),
			)
		}

		gen := service.NewServiceGenerator(
			options...,
		)
		err := gen.Generate()
		if err != nil {
			logrus.Error(err)
			return
		}
	},
}

func init() {
	newCmd.AddCommand(serviceCmd)

	serviceCmd.Flags().StringP("service", "s", "", "The name of the service")
	serviceCmd.Flags().StringP("methods", "m", "", "List of methods of service")
	viper.BindPFlag("ns_service", serviceCmd.Flags().Lookup("service"))
	viper.BindPFlag("ns_methods", serviceCmd.Flags().Lookup("methods"))
}
