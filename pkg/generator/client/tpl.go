package client

var DefaultClientTemplate = `
{{$servicePackageName := BasePath .ServiceImportPath}}
{{$endpointPackageName := BasePath .EndpointImportPath}}
{{$transportPackageName := BasePath .TransportImportPath}}
package {{.PackageName}}

import (
	"errors"
	"io"
	"time"

	{{$servicePackageName}} "{{.ServiceImportPath}}"
        {{$endpointPackageName}} "{{.EndpointImportPath}}"
	{{$transportPackageName}} "{{.TransportImportPath}}"
	"ezrpro.com/micro/spiderconn"
	"ezrpro.com/micro/spiderconn/middleware"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/lb"
	"google.golang.org/grpc"
)

func New(opts ...Option) (addservice.AddService, error) {
	var options Options
	options = newOptions(opts...)

	switch options.transport {
	case spiderconn.TransportTypeGRPC:
		return newGrpcClient(options.grpcAddr, options.middlewareCreators)
	case spiderconn.TransportTypeHTTP:
		return newHTTPClient(options.httpAddr, options.middlewareCreators)
	default:
		options.transport = spiderconn.DefaultTransport

		if options.instancer == nil &&
			options.grpcAddr == "" && options.httpAddr == "" {

			if options.consulAddr == "" {
				options.consulAddr = spiderconn.DefaultConsulAddress
			}

			var err error
			options.instancer, err = spiderconn.NewConsulSDInstancer(
				options.consulAddr,
				options.logger,
				options.serviceName,
				[]string{options.transport, options.version},
			)
			if err != nil {
				options.logger.Log(
					"service", options.serviceName,
					"consul", options.consulAddr,
					"transport", options.transport,
					"err", err)
				return nil, err
			}
		}

		return newSDClient(
			options.logger,
			options.instancer,
			options.transport,
			options.serviceName,
			options.middlewareCreators,
		)

	}
}

func newGrpcClient(addr string, middlewareCreators []middleware.Creator) ({{$servicePackageName}}.{{.ServiceName}}, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		return nil, err
	}

	return {{$transportPackageName}}.NewGRPCClient(
		conn,
		{{$transportPackageName}}.WithClientMiddlewares(middlewareCreators...),
	), nil
}

func newHTTPClient(addr string, middlewareCreators []middleware.Creator) ({{$servicePackageName}}.{{.ServiceName}}, error) {
	return {{$transportPackageName}}.NewHTTPClient(
		addr,
		{{$transportPackageName}}.WithClientMiddlewares(middlewareCreators...),
	)
}

func newSDClient(logger log.Logger, instancer sd.Instancer, transport, svcName string, middlewareCreators []middleware.Creator) ({{$servicePackageName}}.{{.ServiceName}}, error) {
	var (
		retryMax     = 3
		retryTimeout = 500 * time.Millisecond
	)

	var (
		endpoints  {{$endpointPackageName}}.Set
		factoryFor func(makeEndpoint func({{$servicePackageName}}.{{.ServiceName}}) spiderconn.EndpointWrapper) sd.Factory
	)
	switch transport {
	case spiderconn.TransportTypeGRPC:
		factoryFor = grpcFactoryFor
	case spiderconn.TransportTypeHTTP:
		factoryFor = httpFactoryFor
	default:
		return nil, errors.New("Unsupport transport:" + transport)
	}

{{range $index, $method := .ServiceMethods}}
        var {{ToLowerFirstCamelCase $method.Name}}Endpoint endpoint.Endpoint
        {
	method := "{{$method.Name}}"
	factory := factoryFor({{$endpointPackageName}}.Make{{$method.Name}}Endpoint)
	endpointer := sd.NewEndpointer(instancer, factory, logger)
	balancer := lb.NewRoundRobin(endpointer)
	retry := lb.Retry(retryMax, retryTimeout, balancer)
        {{ToLowerFirstCamelCase $method.Name}}Endpoint = retry
	for _, middlewareCreator := range middlewareCreators {
		{{ToLowerFirstCamelCase $method.Name}}Endpoint = middlewareCreator(method)({{ToLowerFirstCamelCase $method.Name}}Endpoint)
	}
	endpoints.{{$method.Name}}Endpoint = spiderconn.NewWrapper(method, {{ToLowerFirstCamelCase $method.Name}}Endpoint)
        }
{{end}}
	return endpoints, nil
}

func grpcFactoryFor(makeEndpoint func({{$servicePackageName}}.{{.ServiceName}}) spiderconn.EndpointWrapper) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		conn, err := grpc.Dial(instance, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
		if err != nil {
			return nil, nil, err
		}

		svc := {{$transportPackageName}}.NewGRPCClient(conn)

		return makeEndpoint(svc).Do, conn, nil
	}
}

func httpFactoryFor(makeEndpoint func({{$servicePackageName}}.{{.ServiceName}}) spiderconn.EndpointWrapper) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		svc, err := {{$transportPackageName}}.NewHTTPClient(instance)
		if err != nil {
			return nil, nil, err
		}

		return makeEndpoint(svc).Do, nil, nil
	}
}
`

var DefaultOptionsTemplate = `
package addclient

import (
	"ezrpro.com/micro/spiderconn"
	"ezrpro.com/micro/spiderconn/middleware"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
)

const (
	defaultServiceName    = "go.server"
	defaultServcieVersion = "1.0.0"
)

type Options struct {
	logger log.Logger

	// 默认使用consul
	instancer sd.Instancer

	// consulAddr, grpcAddr, httpAddr 三选一
	// transport, version 为consul获取可用连接时的tags
	transport   string
	serviceName string
	version     string
	consulAddr  string

	// http直连
	httpAddr string

	// grpc直连
	grpcAddr string

	middlewareCreators []middleware.Creator
}

type Option func(*Options)

func newOptions(opts ...Option) Options {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}

	if options.serviceName == "" {
		options.serviceName = spiderconn.DefaultServiceName
	}

	if options.version == "" {
		options.version = spiderconn.DefaultServiceVersion
	}

	if options.logger == nil {
		options.logger = spiderconn.DefaultLogger
	}

	return options
}

func WithInstancer(instancer sd.Instancer) Option {
	return func(o *Options) {
		o.instancer = instancer
	}
}

func WithLogger(logger log.Logger) Option {
	return func(o *Options) {
		o.logger = logger
	}
}

func WithServcieName(name string) Option {
	return func(o *Options) {
		o.serviceName = name
	}
}

func WithVersion(version string) Option {
	return func(o *Options) {
		o.version = version
	}
}

func WithTransportGRPC() Option {
	return func(o *Options) {
		o.transport = spiderconn.TransportTypeGRPC
	}
}

func WithTransportHTTP() Option {
	return func(o *Options) {
		o.transport = spiderconn.TransportTypeHTTP
	}
}

func WithConsulAddress(addr string) Option {
	return func(o *Options) {
		o.consulAddr = addr
	}
}

func WithHTTPAddress(addr string) Option {
	return func(o *Options) {
		o.httpAddr = addr
		WithTransportHTTP()(o)
	}
}

func WithGrpcAddress(addr string) Option {
	return func(o *Options) {
		o.grpcAddr = addr
		WithTransportGRPC()(o)
	}
}

func WithMiddlewareCreator(middlewareCreators ...middleware.Creator) Option {
	return func(o *Options) {
		o.middlewareCreators = append(o.middlewareCreators, middlewareCreators...)
	}
}
`

var DefaultExportedTemplate = `
{{$servicePackageName := BasePath .ServiceImportPath}}
package addclient

import (
	"context"

	"time"

	{{$servicePackageName}} "{{.ServiceImportPath}}"
	"ezrpro.com/micro/spiderconn/middleware/circuitbreaker"
	"ezrpro.com/micro/spiderconn/middleware/limiter"
	"ezrpro.com/micro/spiderconn/middleware/tracer"
        "ezrpro.com/micro/spiderconn"
	"github.com/go-kit/kit/log"
	opentracing "github.com/opentracing/opentracing-go"
	"golang.org/x/time/rate"
)

var (
	defaultClient {{$servicePackageName}}.{{.ServiceName}}
)

{{range $index, $method := .ServiceMethods}}
func {{$method.Name}}(ctx context.Context, req *{{$servicePackageName}}.{{$method.Name}}Request) (resp *{{$servicePackageName}}.{{$method.Name}}Response, err error) {
	return GetDefaultClient().{{$method.Name}}(ctx, req)
}
{{end}}

func GetDefaultClient() {{$servicePackageName}}.{{.ServiceName}} {
	if defaultClient == nil {
		panic("{{$servicePackageName}} default client is not initialized")
	}
	return defaultClient
}

func SetDefaultClient(svc {{$servicePackageName}}.{{.ServiceName}}) {
	defaultClient = svc
}

func InitDefaultClient(opts ...Option) error {
	var err error
	defaultClient, err = New(opts...)
	if err != nil {
		return err
	}
	return nil
}

func init() {
	InitDefaultClient(
		WithConsulAddress(spiderconn.DefaultConsulAddress),
		WithLogger(spiderconn.DefaultLogger),
		WithMiddlewareCreator(
			tracer.Creator(spiderconn.DefaultTracer),
			limiter.ErroringLimiterCreator(spiderconn.DefaultLimiter),
			circuitbreaker.Creator(spiderconn.DefaultCircuitBreakerOptions...),
		),
	)
}
`
