package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/smallnest/rpcx/log"
	"github.com/spf13/viper"
	"golang.org/x/tools/imports"
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
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	logrus.Info("protoc ", strings.Join(args, " "))
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf(fmt.Sprint(err) + ": " + stderr.String())
	}

	return nil
}

func GoimportsAndformat(filepath string) {
	if err := goimports(filepath); err != nil {
		logrus.Error("goimports filename:", filepath, "error:", err)
	}
}

func goimports(filepath string) error {
	body, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}

	// imports.Process中包含了format.Source
	body, err = imports.Process(filepath, body, nil)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath, body, 0644)
	if err != nil {
		return err
	}

	return nil
}
