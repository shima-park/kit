package server

var DefaultServerTemplate = `
{{$servicePackageName := BasePath .ServiceImportPath}}
{{$endpointPackageName := BasePath .EndpointImportPath}}
{{$protobufPackageName := BasePath .ProtobufImportPath}}
{{$transportPackageName := BasePath .TransportImportPath}}
package {{.PackageName}}

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	{{$servicePackageName}} "{{.ServiceImportPath}}"
        {{$endpointPackageName}} "{{.EndpointImportPath}}"
        {{$protobufPackageName}} "{{.ProtobufImportPath}}"
	{{$transportPackageName}} "{{.TransportImportPath}}"
	"ezrpro.com/micro/spiderconn"
	"ezrpro.com/micro/spiderconn/health"
	"github.com/go-kit/kit/log"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/oklog/oklog/pkg/group"
	"github.com/pborman/uuid"
	"google.golang.org/grpc"
	hv1 "google.golang.org/grpc/health/grpc_health_v1"
)

func New(opts ...Option) *group.Group {
	options := newOptions(opts...)

	options.transportOptions = append(
		options.transportOptions,
		{{$transportPackageName}}.WithEndpointOptions(
			append(
				options.endpointOptions,
				{{$endpointPackageName}}.WithServiceOptions(
					options.serviceOptions...,
				),
			)...,
		),
	)

	svc := spiderconn.Service{
		Name:             options.name,
		RegisterInterval: options.registerInterval,
		TTL:              options.ttl,
	}

	if options.grpcAddr != "" {
		svc.ID = uuid.NewUUID().String()
		svc.Tags = append(
			options.tags,
			[]string{options.version, spiderconn.TransportTypeGRPC}...,
		)
		err := addGRPCServer(options, svc)
		if err != nil {
			options.logger.Log("err", err)
			os.Exit(1)
		}
	}

	if options.httpAddr != "" {
		svc.ID = uuid.NewUUID().String()
		svc.Tags = append(
			options.tags,
			[]string{options.version, spiderconn.TransportTypeHTTP}...,
		)
		err := addHTTPServer(options, svc)
		if err != nil {
			options.logger.Log("err", err)
			os.Exit(1)
		}
	}

	cancelInterrupt := make(chan struct{})
	options.group.Add(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		select {
		case sig := <-c:
			return fmt.Errorf("received signal %s", sig)
		case <-cancelInterrupt:
			return nil
		}
	}, func(error) {
		close(cancelInterrupt)
	})

	return options.group
}

func addGRPCServer(options Options, svc spiderconn.Service) error {
	var (
		logger           log.Logger                  = options.logger
		group            *group.Group                = options.group
		grpcAddr         string                      = options.grpcAddr
		grpcServer       *grpc.Server                = options.grpcServer
                grpcListener     net.Listener                = options.grpcListener
		registrarCreator spiderconn.RegistrarCreator = options.registrarCreator
		consulAddr       string                      = options.registrarAddress
		transportOptions []{{$transportPackageName}}.Option       = options.transportOptions
		isRegister bool = registrarCreator != nil // 注册器不为空时注册当前服务
		isListener bool = grpcListener == nil     // net.Listener为nil时，新建Listener监听，并将服务注册入group。否则外部自行启动监听
		isServe      bool = grpcServer == nil       // grpcServer为nil时，新建grpcServer，并将接口实现和健康检查注册
		registrar  sd.Registrar
		err        error
	)

        if isListener {
		grpcListener, err = net.Listen("tcp", grpcAddr)
		if err != nil {
			logger.Log("transport", "gRPC", "during", "Listen", "err", err)
			return err
		}

		logger.Log("transport", "gRPC", "addr", grpcListener.Addr())
	}

	if isRegister {
		registrar, err = registrarCreator(
			consulAddr,
			logger,
			grpcListener.Addr().String(),
			svc, true)
		if err != nil {
			logger.Log("registrar", "consul", "err", err)
			return err
		}
	}

	if isServe {
		grpcServer = grpc.NewServer(grpc.UnaryInterceptor(kitgrpc.Interceptor))
		// 默认注册上 grpc healthcheck
		hv1.RegisterHealthServer(grpcServer, health.NewGRPCCheckServer())
	}

	transport := {{$transportPackageName}}.NewGRPCServer(transportOptions...)
	{{$protobufPackageName}}.Register{{ToCamelCase .BaseServiceName}}Server(grpcServer, transport)

	errCh := make(chan error, 1)
	group.Add(func() error {
		if isRegister {
			registrar.Register()
		}

		if isListener && isServe {
			errCh <- grpcServer.Serve(grpcListener)
			return <-errCh
		}
		return <-errCh
	}, func(err error) {
		if isRegister {
			registrar.Deregister()
		}

		if isListener {
			grpcListener.Close()
		}

		errCh <- err
	})
	return nil
}

func addHTTPServer(options Options, svc spiderconn.Service) error {
	var (
		logger           log.Logger                  = options.logger
		group            *group.Group                = options.group
		httpAddr         string                      = options.httpAddr
                httpPattern      string                      = options.httpPattern
		httpMux          *http.ServeMux              = options.httpMux
		httpListener     net.Listener                = options.httpListener
		registrarCreator spiderconn.RegistrarCreator = options.registrarCreator
		consulAddr       string                      = options.registrarAddress
		transportOptions []{{$transportPackageName}}.Option       = options.transportOptions

		isRegister       bool                        = registrarCreator != nil
		isListener       bool                        = httpListener == nil
		isServe          bool                        = httpMux == nil
		registrar        sd.Registrar
		err              error
	)

	if isListener {
		httpListener, err = net.Listen("tcp", httpAddr)
		if err != nil {
			logger.Log("transport", "HTTP", "during", "Listen", "err", err)
			return err
		}
		logger.Log("transport", "HTTP", "addr", httpListener.Addr())
	}

	if isRegister {
		registrar, err = registrarCreator(
			consulAddr,
			logger,
			httpListener.Addr().String(),
			svc, false)
		if err != nil {
			logger.Log("registrar", "consul", "err", err)
			return err
		}
	}

	if isServe {
		httpMux = http.NewServeMux()
	}

	transport := {{$transportPackageName}}.NewHTTPHandler(transportOptions...)
	if httpPattern != "" {
		httpMux.Handle(httpPattern, transport)
		return http.Serve(httpListener, httpMux)
	}

	errCh := make(chan error, 1)
	group.Add(func() error {
		if isRegister {
			registrar.Register()
		}

		if isListener && isServe {
			errCh <- http.Serve(httpListener, httpMux)
                        return <-errCh
		}

		return <-errCh
	}, func(error) {
		if isRegister {
			registrar.Deregister()
		}

		if isListener {
			httpListener.Close()
		}

		errCh <- err
	})
	return nil
}`

var DefaultOptionsTemplate = `
{{$servicePackageName := BasePath .ServiceImportPath}}
{{$endpointPackageName := BasePath .EndpointImportPath}}
{{$transportPackageName := BasePath .TransportImportPath}}
package {{.PackageName}}

import (
	"context"
	"net/http"
	"time"

	{{$servicePackageName}} "{{.ServiceImportPath}}"
        {{$endpointPackageName}} "{{.EndpointImportPath}}"
	{{$transportPackageName}} "{{.TransportImportPath}}"
	"ezrpro.com/micro/spiderconn"
	"github.com/go-kit/kit/log"
	"github.com/oklog/oklog/pkg/group"
	"google.golang.org/grpc"
)

type Options struct {
	name             string
	version          string
	ttl              time.Duration
	registerInterval time.Duration
        tags             []string

	logger log.Logger
	//	otTracer opentracing.Tracer

	registrarCreator spiderconn.RegistrarCreator
	registrarAddress string

	transportOptions []{{$transportPackageName}}.Option
	endpointOptions  []{{$endpointPackageName}}.Option
	serviceOptions   []{{$servicePackageName}}.Option

	group *group.Group
	ctx   context.Context

	grpcAddr   string
	grpcServer *grpc.Server
        grpcListener net.Listener

	httpAddr    string
	httpPattern string
	httpMux     *http.ServeMux
        httpListener net.Listener
}

type Option func(*Options)

func newOptions(opts ...Option) Options {
	var options Options
	for _, opt := range opts {
		opt(&options)
	}

	if options.ctx == nil {
		options.ctx = context.Background()
	}

        if options.group == nil {
		options.group = &group.Group{}
	}

	if options.logger == nil {
		options.logger = spiderconn.DefaultLogger
	}

	if options.name == "" {
		options.name = spiderconn.DefaultServiceName
	}

	if options.version == "" {
		options.version = spiderconn.DefaultServiceVersion
	}

	if options.ttl <= time.Second {
		options.ttl = spiderconn.DefaultTTL
	}

	if options.registerInterval <= time.Second {
		options.registerInterval = spiderconn.DefaultRegisterInterval
	}

        if options.grpcAddr == "" && options.httpAddr == "" {
		switch spiderconn.DefaultTransport {
		case spiderconn.TransportTypeGRPC:
			options.grpcAddr = spiderconn.DefaultServiceAddress
		case spiderconn.TransportTypeHTTP:
			options.httpAddr = spiderconn.DefaultServiceAddress
		}
	}

	return options
}

func WithName(name string) Option {
	return func(o *Options) {
		o.name = name
	}
}

func WithVersion(version string) Option {
	return func(o *Options) {
		o.version = version
	}
}

func WithTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.ttl = ttl
	}
}

func WithRegisterInterval(interval time.Duration) Option {
	return func(o *Options) {
		o.registerInterval = interval
	}
}

func WithTags(tags []string) Option {
	return func(o *Options) {
		o.tags = tags
	}
}

func WithLogger(logger log.Logger) Option {
	return func(o *Options) {
		o.logger = logger
		o.transportOptions = append(o.transportOptions, {{$transportPackageName}}.WithLogger(logger))
	}
}

func WithGroup(group *group.Group) Option {
	return func(o *Options) {
		o.group = group
	}
}

func WithRegistrarCreator(c spiderconn.RegistrarCreator) Option {
	return func(o *Options) {
		o.registrarCreator = c
	}
}

func WithRegistrarAddress(addr string) Option {
	return func(o *Options) {
		o.registrarAddress = addr
                if o.registrarCreator == nil {
			o.registrarCreator = spiderconn.NewConsulRegistrar
		}
	}
}

func WithTransportOptions(opts ...{{$transportPackageName}}.Option) Option {
	return func(o *Options) {
		o.transportOptions = append(o.transportOptions, opts...)
	}
}

func WithEndpointOptions(opts ...{{$endpointPackageName}}.Option) Option {
	return func(o *Options) {
		o.endpointOptions = append(o.endpointOptions, opts...)
	}
}

func WithServiceOptions(opts ...{{$servicePackageName}}.Option) Option {
	return func(o *Options) {
		o.serviceOptions = append(o.serviceOptions, opts...)
	}
}

func WithGrpcAddr(grpcAddr string) Option {
	return func(o *Options) {
		o.grpcAddr = grpcAddr
	}
}

func WithGrpcServer(grpcServer *grpc.Server) Option {
	return func(o *Options) {
		o.grpcServer = grpcServer
	}
}

func WithGrpcListener(grpcListener net.Listener) Option {
	return func(o *Options) {
		o.grpcListener = grpcListener
	}
}

func WithHTTPAddr(httpAddr string) Option {
	return func(o *Options) {
		o.httpAddr = httpAddr
	}
}

func WithHTTPPattern(pattern string) Option {
	return func(o *Options) {
		o.httpPattern = pattern
	}
}

func WithHTTPMux(mux *http.ServeMux) Option {
	return func(o *Options) {
		o.httpMux = mux
	}
}

func WithHTTPListener(httpListener net.Listener) Option {
	return func(o *Options) {
		o.httpListener = httpListener
	}
}
`
