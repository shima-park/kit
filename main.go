package main

import (
	"os"
	"path/filepath"
	"strings"

	"ezrpro.com/micro/kit/cmd"
	"ezrpro.com/micro/kit/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	utils.SetDefaults()
	viper.AutomaticEnv()
	gosrc := utils.GetGOPATH() + string(filepath.Separator) + "src" + string(filepath.Separator)
	pwd, err := os.Getwd()
	if err != nil {
		logrus.Error(err)
		return
	}
	if !strings.HasPrefix(pwd, gosrc) {
		logrus.Error("The project must be in the $GOPATH/src folder for the generator to work.")
		return
	}
	cmd.Execute()
}
