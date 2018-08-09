package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	"ezrpro.com/micro/kit/pkg/cst"
	"ezrpro.com/micro/kit/pkg/generator/implement"
	"ezrpro.com/micro/kit/pkg/generator/service"
	"ezrpro.com/micro/kit/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var implCmd = &cobra.Command{
	Use:     "implement",
	Short:   "Generating an implementation of an interface",
	Aliases: []string{"i"},
	Run: func(cmd *cobra.Command, args []string) {
		sourceFile := viper.GetString("n_i_source_file")
		if sourceFile == "" {
			logrus.Error("You must provide a source file for analyze of ast e.g.(-s xxx/service.go)")
			return
		}

		err := generateImpl(sourceFile)
		if err != nil {
			logrus.Error(err)
			return
		}
	},
}

func generateImpl(sourceFile string) error {
	receiver := viper.GetString("n_i_receiver")
	if receiver == "" {
		return errors.New("You must provide a receiver of function e.g.(-r \"f *File\")")
	}

	iface := viper.GetString("n_i_interface")
	if iface == "" {
		return errors.New("You must provide an interface name that needs to be implemented e.g.(-i io.Writer)")
	}

	cst, err := cst.New(sourceFile)
	if err != nil {
		return err
	}
	serviceSuffix := utils.SelectServiceSuffix(sourceFile)
	baseServiceName := service.GetBaseServiceName(cst.PackageName(), serviceSuffix)
	implPath := utils.GetImplFilePath(baseServiceName)
	implPackageName := filepath.Base(implPath)
	filename := filepath.Join(implPath, fmt.Sprintf("%s.go", baseServiceName))

	file, err := createFile(filename)
	if err != nil {
		return errors.New("Create file " + filename + " error:" + err.Error())
	}
	defer GoimportsAndformat(filename)
	defer file.Close()

	gen := implement.NewImplementGenerator(
		receiver,
		iface,
		implement.WithSourceDirctory(filepath.Dir(sourceFile)),
		implement.WithWriter(file),
		implement.WithPackageName(implPackageName),
	)

	err = gen.Generate()
	if err != nil {
		return err
	}

	return nil
}

func init() {
	newCmd.AddCommand(implCmd)

	implCmd.Flags().StringP("source", "s", "", "Source file defined by the service interface")
	implCmd.Flags().StringP("receiver", "r", "", "Receiver of each function")
	implCmd.Flags().StringP("interface", "i", "", "Interface that needs to generate a method list")
	viper.BindPFlag("n_i_interface", implCmd.Flags().Lookup("interface"))
	viper.BindPFlag("n_i_receiver", implCmd.Flags().Lookup("receiver"))
	viper.BindPFlag("n_i_source_file", implCmd.Flags().Lookup("source"))
}
