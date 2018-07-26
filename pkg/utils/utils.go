package utils

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/alecthomas/template"
	"github.com/alioygur/godash"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func SetDefaults() {
	viper.SetDefault("gk_service_path_format", "{{.Path}}/pkg/service/{{.ServiceName}}")
	viper.SetDefault("gk_protobuf_path_format", "{{.Path}}/pkg/pb/{{.ServiceName}}")
	viper.SetDefault("gk_endpoint_path_format", "{{.Path}}/pkg/endpoint/{{.ServiceName}}")
	viper.SetDefault("gk_transport_path_format", "{{.Path}}/pkg/transport/{{.ServiceName}}")
}

func GetFileNameWithoutExt(filename string) string {
	_, filename = filepath.Split(filename)
	extension := filepath.Ext(filename)
	name := filename[0 : len(filename)-len(extension)]
	return name
}

func ToLowerFirstCamelCase(s string) string {
	if s == "" {
		return s
	}
	if len(s) == 1 {
		return strings.ToLower(string(s[0]))
	}
	return strings.ToLower(string(s[0])) + godash.ToCamelCase(s)[1:]
}

func ToUpperFirst(s string) string {
	if s == "" {
		return s
	}
	if len(s) == 1 {
		return strings.ToLower(string(s[0]))
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

func ToLowerSnakeCase(s string) string {
	return strings.ToLower(godash.ToSnakeCase(s))
}

func ToCamelCase(s string) string {
	return godash.ToCamelCase(s)
}

func GetGOPATH() string {
	if os.Getenv("GOPATH") != "" {
		return os.Getenv("GOPATH")
	}
	return defaultGOPATH()
}

func defaultGOPATH() string {
	env := "HOME"
	if runtime.GOOS == "windows" {
		env = "USERPROFILE"
	} else if runtime.GOOS == "plan9" {
		env = "home"
	}
	if home := os.Getenv(env); home != "" {
		def := filepath.Join(home, "go")
		if filepath.Clean(def) == filepath.Clean(runtime.GOROOT()) {
			// Don't set the default GOPATH to GOROOT,
			// as that will trigger warnings from the go tool.
			return ""
		}
		return def
	}
	return ""
}

func GetImportPath() string {
	gopath := filepath.Join(GetGOPATH(), "src")
	return strings.Replace(GetPWD(), gopath, "", -1)
}

func GetPWD() string {
	if viper.GetString("gk_folder") != "" {
		return viper.GetString("gk_folder")
	}

	pwd, err := os.Getwd()
	if err != nil {
		logrus.Fatal(err)
	}

	return pwd
}

func getPath(tmplName, path, serviceName string) string {
	tmpl, err := template.New(tmplName).
		Parse(viper.GetString(tmplName))
	if err != nil {
		logrus.Fatal(err)
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, map[string]interface{}{
		"Path":        path,
		"ServiceName": serviceName,
	})
	if err != nil {
		logrus.Fatal(err)
	}
	return string(buff.Bytes())
}

func getServicePath(path, serviceName string) string {
	return getPath("gk_service_path_format", path, serviceName)
}

func getProtobufPath(path, serviceName string) string {
	return getPath("gk_protobuf_path_format", path, serviceName)
}

func getEndpointPath(path, serviceName string) string {
	return getPath("gk_endpoint_path_format", path, serviceName)
}

func getTransportPath(path, serviceName string) string {
	return getPath("gk_transport_path_format", path, serviceName)
}

func GetServiceImportPath(svc string) string {
	return getServicePath(
		strings.TrimLeft(GetImportPath(), string(filepath.Separator)),
		svc)
}

func GetProtobufImportPath(svc string) string {
	return getProtobufPath(
		strings.TrimLeft(GetImportPath(), string(filepath.Separator)),
		svc)
}

func GetEndpointImportPath(svc string) string {
	return getEndpointPath(
		strings.TrimLeft(GetImportPath(), string(filepath.Separator)),
		svc)
}

func GetTransportImportPath(svc string) string {
	return getTransportPath(
		strings.TrimLeft(GetImportPath(), string(filepath.Separator)),
		svc)
}

func GetServiceFilePath(svc string) string {
	return getServicePath(
		GetPWD(),
		svc)
}

func GetProtobufFilePath(svc string) string {
	return getProtobufPath(
		GetPWD(),
		svc)
}

func GetEndpointFilePath(svc string) string {
	return getEndpointPath(
		GetPWD(),
		svc)
}

func GetTransportFilePath(svc string) string {
	return getTransportPath(
		GetPWD(),
		svc)
}
