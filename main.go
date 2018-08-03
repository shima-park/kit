package main

import (
	"strings"

	"ezrpro.com/micro/kit/cmd"
	"ezrpro.com/micro/kit/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	utils.SetDefaults()
	viper.AutomaticEnv()

	if !strings.HasPrefix(utils.GetPWD(), utils.GetGoSrc()) {
		logrus.Error("The project must be in the $GOPATH/src folder for the generator to work.")
		return
	}
	cmd.Execute()
}
