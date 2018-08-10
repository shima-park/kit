package transport

var DefaultGRPCTemplate = `
{{$servicePackageName := BasePath .ServiceImportPath}}
{{$endpointPackageName := BasePath .EndpointImportPath}}
{{$protobufPackageName := BasePath .ProtobufImportPath}}
{{$baseServiceName := .BaseServiceName}}
{{$requestAndResponseList := .RequestAndResponseList}}
{{$pbRequestAndResponseList := .ProtobufCST.RequestAndResponseList}}
package {{.PackageName}}

import (
	"context"
	"errors"

	"google.golang.org/grpc"

	"github.com/go-kit/kit/tracing/opentracing"
	grpctransport "github.com/go-kit/kit/transport/grpc"

	{{$servicePackageName}} "{{.ServiceImportPath}}"
        {{$endpointPackageName}} "{{.EndpointImportPath}}"
        {{$protobufPackageName}} "{{.ProtobufImportPath}}"
        "ezrpro.com/micro/spiderconn"
)

type grpcServer struct {
{{range $index, $method := .ServiceMethods}}
	{{ToLowerFirstCamelCase $method.Name}} grpctransport.Handler
{{end}}
}

// NewGRPCServer makes a set of endpoints available as a gRPC AddServer.
func NewGRPCServer(opts ...Option) {{.ProtobufCST.PackageName}}.{{.ProtobufCST.ServiceName}} {
	options := newOptions(opts...)

	serverOptions := []grpctransport.ServerOption{
		grpctransport.ServerErrorLogger(options.logger),
	}
	serverOptions = append(serverOptions, options.grpcServerOptions...)

	return &grpcServer{
{{range $index, $method := .ServiceMethods}}
		{{ToLowerFirstCamelCase $method.Name}}: grpctransport.NewServer(
			options.endpoints.{{$method.Name}}Endpoint.Do,
			decodeGRPC{{$method.Name}}Request,
			encodeGRPC{{$method.Name}}Response,
			append(
				serverOptions,
				grpctransport.ServerBefore(
					opentracing.GRPCToContext(options.otTracer, options.endpoints.{{$method.Name}}Endpoint.Name(), options.logger),
				),
			)...,
		),
{{end}}
	}
}
{{range $index, $method := .ServiceMethods}}
func (s *grpcServer) {{$method.Name}}(ctx context.Context, req *{{$protobufPackageName}}.{{$method.Name}}Request) (*{{$protobufPackageName}}.{{$method.Name}}Response, error) {
	_, resp, err := s.{{ToLowerFirstCamelCase $method.Name}}.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.(*{{$protobufPackageName}}.{{$method.Name}}Response), nil
}
{{end}}

// NewGRPCClient returns an AddService backed by a gRPC server at the other end
// of the conn. The caller is responsible for constructing the conn, and
// eventually closing the underlying transport. We bake-in certain middlewares,
// implementing the client library pattern.
func NewGRPCClient(conn *grpc.ClientConn, opts ...ClientOption) {{$servicePackageName}}.{{.ServiceName}} {
	options := newClientOption(opts...)

	// Each individual endpoint is an grpc/transport.Client (which implements
	// endpoint.Endpoint) that gets wrapped with various middlewares. If you
	// made your own client library, you'd do this work there, so your server
	// could rely on a consistent set of client behavior.
{{range $index, $method := .ServiceMethods}}
	var {{ToLowerFirstCamelCase $method.Name}}Wrapper spiderconn.EndpointWrapper
	{
		method := "{{$method.Name}}"
		{{ToLowerFirstCamelCase $method.Name}}Endpoint := grpctransport.NewClient(
			conn,
			"{{$protobufPackageName}}.{{ToCamelCase $baseServiceName}}",
			method,
			encodeGRPC{{$method.Name}}Request,
			decodeGRPC{{$method.Name}}Response,
			{{$protobufPackageName}}.{{$method.Name}}Response{},
			append(options.clientOptions, grpctransport.ClientBefore(opentracing.ContextToGRPC(options.otTracer, options.logger)))...,
		).Endpoint()
		for _, middlewareCreator := range options.middlewareCreators {
			{{ToLowerFirstCamelCase $method.Name}}Endpoint = middlewareCreator(method)({{ToLowerFirstCamelCase $method.Name}}Endpoint)
		}
		{{ToLowerFirstCamelCase $method.Name}}Wrapper = spiderconn.NewWrapper(method, {{ToLowerFirstCamelCase $method.Name}}Endpoint)
	}
{{end}}
	// Returning the endpoint.Set as a {{$servicePackageName}}.{{.ServiceName}} relies on the
	// endpoint.Set implementing the {{.ServiceName}} methods. That's just a simple bit
	// of glue code.
	return {{$endpointPackageName}}.Set{
{{range $index, $method := .ServiceMethods}}
		{{$method.Name}}Endpoint: {{ToLowerFirstCamelCase $method.Name}}Wrapper,
{{end}}
	}
}

{{range .RequestAndResponseList}}
{{if .Request}}
// decodeGRPC{{.Request.Name}} is a transport/grpc.DecodeRequestFunc that converts a
// gRPC {{.Request.Name}} to a user-domain {{.Request.Name}}. Primarily useful in a server.
func decodeGRPC{{.Request.Name}}(_ context.Context, grpcReq interface{}) (interface{}, error) {
        if grpcReq == nil {
            return nil, nil
        }
	req := grpcReq.(*{{$protobufPackageName}}.{{.Request.Name}})
        {{if not .Request}}_ = req{{end}}
	return &{{$servicePackageName}}.{{.Request.Name}}{
            {{$alias := NewObjectAlias "req" $protobufPackageName .Request.Name true}}
            {{GenerateAssignmentSegment .Request $pbRequestAndResponseList $alias}}
        }, nil
}
{{end}}

{{if .Response}}
// decodeGRPC{{.Response.Name}} is a transport/grpc.DecodeResponseFunc that converts a
// gRPC {{.Response.Name}} to a user-domain {{.Response.Name}}. Primarily useful in a client.
func decodeGRPC{{.Response.Name}}(_ context.Context, grpcResponse interface{}) (interface{}, error) {
        if grpcResponse == nil {
            return nil, nil
        }
	resp := grpcResponse.(*{{$protobufPackageName}}.{{.Response.Name}})
	return &{{$servicePackageName}}.{{.Response.Name}}{
            {{$alias := NewObjectAlias "resp" $protobufPackageName .Response.Name true}}
            {{GenerateAssignmentSegment .Response $pbRequestAndResponseList $alias}}
        }, nil
}
{{end}}
{{end}}


{{range .ProtobufCST.RequestAndResponseList}}
{{if .Request}}
// encodeGRPC{{.Request.Name}} is a transport/grpc.EncodeRequestFunc that converts a
// user-domain {{.Request.Name}} to a gRPC {{.Request.Name}}. Primarily useful in a client.
func encodeGRPC{{.Request.Name}}(_ context.Context, request interface{}) (interface{}, error) {
        if request == nil {
            return nil, nil
        }
	req := request.(*{{$servicePackageName}}.{{.Request.Name}})
        {{if not .Request}}_ = req{{end}}
	return &{{$protobufPackageName}}.{{.Request.Name}}{
            {{$alias := NewObjectAlias "req" $servicePackageName .Request.Name true}}
            {{GenerateAssignmentSegment .Request $requestAndResponseList $alias}}
        }, nil
}
{{end}}

{{if .Response}}
// encodeGRPC{{.Response.Name}} is a transport/grpc.EncodeResponseFunc that converts a
// user-domain {{.Response.Name}} to a gRPC {{.Response.Name}}. Primarily useful in a server.
func encodeGRPC{{.Response.Name}}(_ context.Context, response interface{}) (interface{}, error) {
        if response == nil {
            return nil, nil
        }
	resp := response.(*{{$servicePackageName}}.{{.Response.Name}})
	return &{{$protobufPackageName}}.{{.Response.Name}}{
            {{$alias := NewObjectAlias "resp" $servicePackageName .Response.Name true}}
            {{GenerateAssignmentSegment .Response $requestAndResponseList $alias}}
        }, nil
}
{{end}}

{{end}}
`

var DefaultHTTPTemplate = `
{{$servicePackageName := BasePath .ServiceImportPath}}
{{$endpointPackageName := BasePath .EndpointImportPath}}
package {{.PackageName}}

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	{{$servicePackageName}} "{{.ServiceImportPath}}"
        {{$endpointPackageName}} "{{.EndpointImportPath}}"
	"ezrpro.com/micro/spiderconn"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/tracing/opentracing"
	httptransport "github.com/go-kit/kit/transport/http"
)

// NewHTTPHandler returns an HTTP handler that makes a set of endpoints
// available on predefined paths.
func NewHTTPHandler(opts ...Option) http.Handler {
	options := newOptions(opts...)

	m := http.NewServeMux()
{{range $index, $method := .ServiceMethods}}
	m.Handle("/{{ToLowerFirstCamelCase $method.Name}}", httptransport.NewServer(
		options.endpoints.{{$method.Name}}Endpoint.Do,
		decodeHTTP{{$method.Name}}Request,
		encodeHTTPGenericResponse,
		append(options.httpServerOptions, httptransport.ServerBefore(opentracing.HTTPToContext(options.otTracer, "{{$method.Name}}", options.logger)))...,
	))
{{end}}
	return m
}

// NewHTTPClient returns an AddService backed by an HTTP server living at the
// remote instance. We expect instance to come from a service discovery system,
// so likely of the form "host:port". We bake-in certain middlewares,
// implementing the client library pattern.
func NewHTTPClient(instance string, opts ...ClientOption) ({{$servicePackageName}}.{{.ServiceName}}, error) {
	// Quickly sanitize the instance string.
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}

	options := newClientOption(opts...)

{{range $index, $method := .ServiceMethods}}
	var {{ToLowerFirstCamelCase $method.Name}}Wrapper spiderconn.EndpointWrapper
	{
		method := "{{$method.Name}}"
		{{ToLowerFirstCamelCase $method.Name}}Endpoint := httptransport.NewClient(
			"POST",
			copyURL(u, "/{{ToLowerFirstCamelCase $method.Name}}"),
			encodeHTTPGenericRequest,
			decodeHTTP{{$method.Name}}Response,
			append(options.httpClientOptions, httptransport.ClientBefore(opentracing.ContextToHTTP(options.otTracer, options.logger)))...,
		).Endpoint()
		for _, middlewareCreator := range options.middlewareCreators {
			{{ToLowerFirstCamelCase $method.Name}}Endpoint = middlewareCreator(method)({{ToLowerFirstCamelCase $method.Name}}Endpoint)
		}
		{{ToLowerFirstCamelCase $method.Name}}Wrapper = spiderconn.NewWrapper(method, {{ToLowerFirstCamelCase $method.Name}}Endpoint)
	}
{{end}}
	// Returning the endpoint.Set as a service.Service relies on the
	// endpoint.Set implementing the Service methods. That's just a simple bit
	// of glue code.
	return {{$endpointPackageName}}.Set{
{{range $index, $method := .ServiceMethods}}
		{{$method.Name}}Endpoint: {{ToLowerFirstCamelCase $method.Name}}Wrapper,
{{end}}
	}, nil
}

func copyURL(base *url.URL, path string) *url.URL {
	next := *base
	next.Path = path
	return &next
}

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	w.WriteHeader(err2code(err))
	json.NewEncoder(w).Encode(errorWrapper{Error: err.Error()})
}

func err2code(err error) int {
	//switch err {
	//case addservice.ErrTwoZeroes, addservice.ErrMaxSizeExceeded, addservice.ErrIntOverflow:
	//	return http.StatusBadRequest
	//}
	// TODO
	if err != nil {
		return http.StatusInternalServerError
	}
	return http.StatusOK
}

func errorDecoder(r *http.Response) error {
	var w errorWrapper
	if err := json.NewDecoder(r.Body).Decode(&w); err != nil {
		return err
	}
	return errors.New(w.Error)
}

type errorWrapper struct {
	Error string ` + "`json:\"error\"`" + `
}

{{range .RequestAndResponseList}}
{{if .Request}}
// decodeHTTP{{.Request.Name}} is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded {{.Request.Name}} from the HTTP request body. Primarily useful in a
// server.
func decodeHTTP{{.Request.Name}}(_ context.Context, r *http.Request) (interface{}, error) {
	var req addservice.{{.Request.Name}}
	err := json.NewDecoder(r.Body).Decode(&req)
	return &req, err
}
{{end}}

{{if .Response}}
// decodeHTTP{{.Response.Name}} is a transport/http.DecodeResponseFunc that decodes a
// JSON-encoded {{.Response.Name}} from the HTTP response body. If the response has a
// non-200 status code, we will interpret that as an error and attempt to decode
// the specific error message from the response body. Primarily useful in a
// client.
func decodeHTTP{{.Response.Name}}(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errors.New(r.Status)
	}
	var resp addservice.{{.Response.Name}}
	err := json.NewDecoder(r.Body).Decode(&resp)
	return &resp, err
}
{{end}}
{{end}}

// encodeHTTPGenericRequest is a transport/http.EncodeRequestFunc that
// JSON-encodes any request to the request body. Primarily useful in a client.
func encodeHTTPGenericRequest(_ context.Context, r *http.Request, request interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	return nil
}

// encodeHTTPGenericResponse is a transport/http.EncodeResponseFunc that encodes
// the response as JSON to the response writer. Primarily useful in a server.
func encodeHTTPGenericResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if f, ok := response.(endpoint.Failer); ok && f.Failed() != nil {
		errorEncoder(ctx, f.Failed(), w)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}
`

var DefaultOptionsTemplate = `
{{$endpointPackageName := BasePath .EndpointImportPath}}
package {{.PackageName}}

import (
	{{$endpointPackageName}} "{{.EndpointImportPath}}"
	"ezrpro.com/micro/spiderconn"
	"ezrpro.com/micro/spiderconn/middleware"
	"github.com/go-kit/kit/log"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	httptransport "github.com/go-kit/kit/transport/http"
	stdopentracing "github.com/opentracing/opentracing-go"
)

type Options struct {
	endpoints         *{{$endpointPackageName}}.Set
	logger            log.Logger
	otTracer          stdopentracing.Tracer
	endpointOptions   []{{$endpointPackageName}}.Option
	grpcServerOptions []grpctransport.ServerOption
	httpServerOptions []httptransport.ServerOption
}

type Option func(*Options)

func newOptions(opts ...Option) Options {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}

	if options.endpoints == nil {
		options.endpoints = {{$endpointPackageName}}.New(options.endpointOptions...)
	}

	if options.logger == nil {
		options.logger = spiderconn.DefaultLogger
	}

	if options.otTracer == nil {
		options.otTracer = spiderconn.DefaultTracer
	}
	return options
}

func WithEndpoints(endpoints *{{$endpointPackageName}}.Set) Option {
	return func(o *Options) {
		o.endpoints = endpoints
	}
}

func WithLogger(logger log.Logger) Option {
	return func(o *Options) {
		o.logger = logger
	}
}

func WithTracer(tracer stdopentracing.Tracer) Option {
	return func(o *Options) {
		o.otTracer = tracer
	}
}

func WithEndpointOptions(opts ...{{$endpointPackageName}}.Option) Option {
	return func(o *Options) {
		o.endpointOptions = append(o.endpointOptions, opts...)
	}
}

func WithGrpcServerOptions(grpcServerOptions ...grpctransport.ServerOption) Option {
	return func(o *Options) {
		o.grpcServerOptions = append(o.grpcServerOptions, grpcServerOptions...)
	}
}

func WithHTTPServerOptions(serverOptions ...httptransport.ServerOption) Option {
	return func(o *Options) {
		o.httpServerOptions = append(o.httpServerOptions, serverOptions...)
	}
}

type ClientOptions struct {
	logger             log.Logger
	otTracer           stdopentracing.Tracer
	middlewareCreators []middleware.Creator
	clientOptions      []grpctransport.ClientOption
	httpClientOptions  []httptransport.ClientOption
}

type ClientOption func(*ClientOptions)

func newClientOption(opts ...ClientOption) ClientOptions {
	var options ClientOptions
	for _, opt := range opts {
		opt(&options)
	}

	if options.logger == nil {
		options.logger = spiderconn.DefaultLogger
	}

	if options.otTracer == nil {
		options.otTracer = spiderconn.DefaultTracer
	}
	return options
}

func WithClientLogger(logger log.Logger) ClientOption {
	return func(o *ClientOptions) {
		o.logger = logger
	}
}

func WithClientTracer(tracer stdopentracing.Tracer) ClientOption {
	return func(o *ClientOptions) {
		o.otTracer = tracer
	}
}

func WithClientMiddlewares(middlewareCreators ...middleware.Creator) ClientOption {
	return func(o *ClientOptions) {
		o.middlewareCreators = append(o.middlewareCreators, middlewareCreators...)
	}
}

func WithClientTransportOptions(opts ...grpctransport.ClientOption) ClientOption {
	return func(o *ClientOptions) {
		o.clientOptions = append(o.clientOptions, opts...)
	}
}
`
