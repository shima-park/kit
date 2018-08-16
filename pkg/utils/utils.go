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
	viper.SetDefault("gk_service_path_format", "{{.Path}}/pkg/{{.ServiceName}}service")
	viper.SetDefault("gk_protobuf_path_format", "{{.Path}}/pkg/{{.ServiceName}}pb")
	viper.SetDefault("gk_endpoint_path_format", "{{.Path}}/pkg/{{.ServiceName}}endpoint")
	viper.SetDefault("gk_transport_path_format", "{{.Path}}/pkg/{{.ServiceName}}transport")
	viper.SetDefault("gk_server_path_format", "{{.Path}}/pkg/{{.ServiceName}}server")
	viper.SetDefault("gk_client_path_format", "{{.Path}}/pkg/{{.ServiceName}}client")
	viper.SetDefault("gk_impl_path_format", "{{.Path}}/pkg/{{.ServiceName}}impl")
	viper.SetDefault("gk_service_suffix", "Service")
	viper.SetDefault("gk_protobuf_service_suffix", "Server")
	viper.SetDefault("gk_request_suffix", "Request")
	viper.SetDefault("gk_response_suffix", "Response")
	viper.SetDefault("gk_protobuf_path", "")
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

func GetGoSrc() string {
	return filepath.Join(GetGOPATH(), "src")
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

func GetPWDImportPath() string {
	return strings.Replace(GetPWD(), GetGoSrc(), "", -1)
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

func getServerPath(path, serviceName string) string {
	return getPath("gk_server_path_format", path, serviceName)
}

func getClientPath(path, serviceName string) string {
	return getPath("gk_client_path_format", path, serviceName)
}

func getImplPath(path, serviceName string) string {
	return getPath("gk_impl_path_format", path, serviceName)
}

func GetServiceSuffix() string {
	return viper.GetString("gk_service_suffix")
}

func GetProtobufServiceSuffix() string {
	return viper.GetString("gk_protobuf_service_suffix")
}

func IsProtobufSourceFile(sourceFile string) bool {
	return strings.HasSuffix(sourceFile, "pb.go")
}

func SelectServiceSuffix(sourceFile string) string {
	if IsProtobufSourceFile(sourceFile) {
		return GetProtobufServiceSuffix()
	}
	return GetServiceSuffix()
}

func GetRequestSuffix() string {
	return viper.GetString("gk_request_suffix")
}

func GetResponseSuffix() string {
	return viper.GetString("gk_response_suffix")
}

func GetServiceImportPath(svc string) string {
	return getServicePath(
		strings.TrimLeft(GetPWDImportPath(), string(filepath.Separator)),
		svc)
}

func GetProtobufImportPath(svc string) string {
	if GetProtobufPath() != "" {
		return GetProtobufPath()
	}
	return getProtobufPath(
		strings.TrimLeft(GetPWDImportPath(), string(filepath.Separator)),
		svc)
}

func GetProtobufPath() string {
	return viper.GetString("gk_protobuf_path")
}

func SetProtobufPath(path string) {
	viper.Set("gk_protobuf_path", path)
}

func GetEndpointImportPath(svc string) string {
	return getEndpointPath(
		strings.TrimLeft(GetPWDImportPath(), string(filepath.Separator)),
		svc)
}

func GetTransportImportPath(svc string) string {
	return getTransportPath(
		strings.TrimLeft(GetPWDImportPath(), string(filepath.Separator)),
		svc)
}

func GetServerImportPath(svc string) string {
	return getServerPath(
		strings.TrimLeft(GetPWDImportPath(), string(filepath.Separator)),
		svc)
}

func GetClientImportPath(svc string) string {
	return getClientPath(
		strings.TrimLeft(GetPWDImportPath(), string(filepath.Separator)),
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

func GetServerFilePath(svc string) string {
	return getServerPath(
		GetPWD(),
		svc)
}

func GetClientFilePath(svc string) string {
	return getClientPath(
		GetPWD(),
		svc)
}

func GetImplFilePath(svc string) string {
	return getImplPath(
		GetPWD(),
		svc)
}

// GetImportPathByFileAbsPath
// 转换/Users/liuxingwang/go/src/ezrpro.com/micro/demo/pkg/addpb/addservice.pb.go
// 成 ezrpro.com/micro/demo/pkg/addpb
func GetImportPathByFileAbsPath(fileAbsPath string) string {
	base := filepath.Base(fileAbsPath)
	s := strings.Replace(fileAbsPath, GetGoSrc(), "", -1)
	s = strings.Replace(s, base, "", -1)
	s = strings.TrimLeft(s, string(filepath.Separator))
	s = strings.TrimRight(s, string(filepath.Separator))
	return s
}

// path := "/Users/liuxingwang/go/src/ezrpro.com/micro/demo/model/misc.go:9:2"
// path = GetPackageNameByFileAbsPath(path)
// path = model
func GetPackageNameByFileAbsPath(path string) string {
	path = filepath.Dir(path)
	path = strings.TrimPrefix(path, GetGoSrc())
	return filepath.Base(path)
}
