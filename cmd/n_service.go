package cmd

import (
	"path/filepath"
	"strings"

	"ezrpro.com/micro/kit/pkg/generator/service"
	"ezrpro.com/micro/kit/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/smallnest/rpcx/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serviceCmd = &cobra.Command{
	Use:     "service",
	Short:   "Generate new service",
	Aliases: []string{"s"},
	Run: func(cmd *cobra.Command, args []string) {
		serviceName := strings.ToLower(viper.GetString("ns_name"))
		if serviceName == "" {
			logrus.Error("You must provide a name for the service e.g.(./kit n s -n=Add -m=Sum,Concat)")
			cmd.Help()
			return
		}

		methods := strings.Split(viper.GetString("ns_methods"), ",")

		servicePath := utils.GetServiceFilePath(serviceName)

		filename := filepath.Join(servicePath, serviceName+".go")
		log.Info("create file:", filename)
		file, err := createFile(filename)
		if err != nil {
			logrus.Error(err)
			return
		}
		defer file.Close()

		gen := service.NewServiceGenerator(
			service.WithServiceName(serviceName),
			service.WithWriter(file),
			service.WithMethods(methods),
		)
		err = gen.Generate()
		if err != nil {
			logrus.Error(err)
			return
		}

		formatAndGoimports(filename)
	},
}

func init() {
	newCmd.AddCommand(serviceCmd)

	serviceCmd.Flags().StringP("name", "n", "", "Name of service")
	serviceCmd.Flags().StringP("methods", "m", "", "List of methods of service")
	viper.BindPFlag("ns_name", serviceCmd.Flags().Lookup("name"))
	viper.BindPFlag("ns_methods", serviceCmd.Flags().Lookup("methods"))
}
