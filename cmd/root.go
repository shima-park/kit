package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "kit",
	Short: "Kit is go-kit source generator",
	Long:  `Kit是一个基于go-kit搭建微服务框架的通用层源码生成工具`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "If you want to see the debug logs.")
	rootCmd.PersistentFlags().BoolP("force", "f", false, "Force overide existing files without asking.")
	rootCmd.PersistentFlags().StringP("folder", "b", "", "If you want to specify the base folder of the project.")
	viper.BindPFlag("gk_folder", rootCmd.PersistentFlags().Lookup("folder"))
	viper.BindPFlag("gk_force", rootCmd.PersistentFlags().Lookup("force"))
	viper.BindPFlag("gk_debug", rootCmd.PersistentFlags().Lookup("debug"))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}

func checkProtoc() bool {
	_, err := exec.LookPath("protoc")
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("Please install protoc first and than rerun the command")
		if runtime.GOOS == "windows" {
			fmt.Println(
				`Install proto3.
https://github.com/google/protobuf/releases
Update protoc Go bindings via
> go get -u github.com/golang/protobuf/proto
> go get -u github.com/golang/protobuf/protoc-gen-go

See also
https://github.com/grpc/grpc-go/tree/master/examples`,
			)
		} else if runtime.GOOS == "darwin" {
			fmt.Println(
				`Install proto3 from source macOS only.
> brew install autoconf automake libtool
> git clone https://github.com/google/protobuf
> ./autogen.sh ; ./configure ; make ; make install

Update protoc Go bindings via
> go get -u github.com/golang/protobuf/{proto,protoc-gen-go}

See also
https://github.com/grpc/grpc-go/tree/master/examples`,
			)
		} else {
			fmt.Println(`Install proto3
sudo apt-get install -y git autoconf automake libtool curl make g++ unzip
git clone https://github.com/google/protobuf.git
cd protobuf/
./autogen.sh
./configure
make
make check
sudo make install
sudo ldconfig # refresh shared library cache.`)
		}
		return false
	}
	return true
}
