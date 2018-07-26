package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/smallnest/rpcx/log"
	"github.com/spf13/viper"
)

func createFile(filename string) (writerCloser io.WriteCloser, err error) {
	dir, _ := filepath.Split(filename)

	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return
	}

	_, err = os.Stat(filename)
	if err == nil {
		if viper.GetBool("gk_force") {
			log.Info("remove file:", filename)
			err = os.Remove(filename)
			if err != nil {
				return
			}
		} else {
			err = fmt.Errorf("File(%s) already exists", filename)
			return
		}
	}
	log.Info("create file:", filename)
	return os.Create(filename)
}

func generateProtobufGo(protoPath string) error {
	genPbPath, _ := filepath.Split(protoPath)
	args := []string{
		"-I", genPbPath,
		protoPath,
		"--go_out=plugins=grpc:" + genPbPath,
	}
	//protoc -I ./ --go_out=plugins=grpc:./ ./test.proto
	cmd := exec.Command("protoc", args...)
	logrus.Info("protoc ", strings.Join(args, " "))
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func formatAndGoimports(filepath string) {
	if err := goimports(filepath); err != nil {
		logrus.Error("goimports filename:", filepath, "error:", err)
	}

	if err := gofmt(filepath); err != nil {
		logrus.Error("gofmt filename:", filepath, "error:", err)
	}
}

func gofmt(filepath string) error {
	args := []string{
		"-w", filepath,
	}
	cmd := exec.Command("gofmt", args...)
	logrus.Info("gofmt ", strings.Join(args, " "))
	return cmd.Run()
}

func goimports(filepath string) error {
	args := []string{
		"-w", filepath,
	}
	cmd := exec.Command("goimports", args...)
	logrus.Info("goimports ", strings.Join(args, " "))
	return cmd.Run()
}
