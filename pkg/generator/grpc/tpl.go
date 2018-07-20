package grpc

var tpl = `
syntax = "proto3";

package {{.PackageName}};

service {{.ServiceName}} {
{{range $i, $v := .InterfaceMethods}}
  rpc {{$v.Name}} ({{$v.RequestParamName}}) returns ($v.ResponseParamName}}) {}
{{end}}
}

{{range $i, $v := .Messages}}
message {{$v.Name}} {
  {{range $i, $v := .Messages}}
    {{$v.Type}} {{$v.Name}} = {{$i}};
  {{end}}
}
{{end}}
`
