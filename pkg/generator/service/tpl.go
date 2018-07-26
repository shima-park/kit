package service

var defaultTemplate = `
package {{ToLower .ServiceName}}

import (
	"context"
        "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
)

// {{ToCamelCase .ServiceName}} describes the service.
type {{ToCamelCase .ServiceName}} interface {
{{if not .InterfaceMethods}}
    // Add your methods here
    // e.x: Foo(ctx context.Context, *FooRequest)(*FooResponse, err error)
{{else}}
    {{range .InterfaceMethods}}
        {{.}}(ctx context.Context,req *{{.}}Request)(resp *{{.}}Response, err error)
    {{end}}
{{end}}
}

// New returns a basic Service with all of the expected middlewares wired in.
func New(logger log.Logger, ints, chars metrics.Counter) {{ToCamelCase .ServiceName}} {
	var svc {{ToCamelCase .ServiceName}}
	{
		svc = NewBasicService()
	}
	return svc
}

{{if .InterfaceMethods}}
    {{range .InterfaceMethods}}
        type {{.}}Request struct{

        }

        type {{.}}Response struct{

        }
    {{end}}
{{end}}

// NewBasicService returns a naïve, stateless implementation of Service.
func NewBasicService() {{ToCamelCase .ServiceName}} {
	return basicService{}
}

type basicService struct{}

{{if .InterfaceMethods}}
    {{range .InterfaceMethods}}
        // {{.}} implements Service.
        func (s basicService) {{.}}(ctx context.Context,req *{{.}}Request) (resp *{{.}}Response,err error) {
        	return
        }
    {{end}}
{{end}}
`
