package server

var DefaultServerTemplate = `
{{$servicePackageName := BasePath .ServiceImportPath}}
{{$endpointPackageName := BasePath .EndpointImportPath}}
{{$protobufPackageName := BasePath .ProtobufImportPath}}
{{$transportPackageName := BasePath .TransportImportPath}}
package addserver

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
		addtransport.WithEndpointOptions(
			append(
				options.endpointOptions,
				addendpoint.WithServiceOptions(
					options.serviceOptions...,
				),
			)...,
		),
	)

	svc := spiderconn.Service{
		ID:               uuid.NewUUID().String(),
		Name:             options.name,
		RegisterInterval: options.registerInterval,
		TTL:              options.ttl,
	}

	if options.grpcAddr != "" {
		svc.Tags = []string{options.version, spiderconn.TransportTypeGRPC}
		err := addGRPCServer(options, svc)
		if err != nil {
			options.logger.Log("err", err)
			os.Exit(1)
		}
	}

	if options.httpAddr != "" {
		svc.Tags = []string{options.version, spiderconn.TransportTypeHTTP}
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
		registrarCreator spiderconn.RegistrarCreator = options.registrarCreator
		consulAddr       string                      = options.registrarAddress
		transportOptions []{{$transportPackageName}}.Option       = options.transportOptions
	)

	if grpcServer == nil {
		grpcServer = grpc.NewServer(grpc.UnaryInterceptor(kitgrpc.Interceptor))
	}

	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Log("transport", "gRPC", "during", "Listen", "err", err)
		return err
	}

	registrar, err := registrarCreator(
		consulAddr,
		logger,
		grpcListener.Addr().String(),
		svc, true)
	if err != nil {
		logger.Log("registrar", "consul", "err", err)
		return err
	}

	group.Add(func() error {
		logger.Log("transport", "gRPC", "addr", grpcListener.Addr())

		transport := {{$transportPackageName}}.NewGRPCServer(transportOptions...)

		addpb.RegisterAddServer(grpcServer, transport)
		// 默认注册上 grpc healthcheck
		hv1.RegisterHealthServer(grpcServer, health.NewGRPCCheckServer())

		registrar.Register()

		return grpcServer.Serve(grpcListener)
	}, func(error) {
		registrar.Deregister()

		grpcListener.Close()
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
		registrarCreator spiderconn.RegistrarCreator = options.registrarCreator
		consulAddr       string                      = options.registrarAddress
		transportOptions []{{$transportPackageName}}.Option       = options.transportOptions
	)

	if httpMux == nil {
		httpMux = http.NewServeMux()
	}

	httpListener, err := net.Listen("tcp", httpAddr)
	if err != nil {
		logger.Log("transport", "HTTP", "during", "Listen", "err", err)
		return err
	}

	registrar, err := registrarCreator(
		consulAddr,
		logger,
		httpListener.Addr().String(),
		svc, false)
	if err != nil {
		logger.Log("registrar", "consul", "err", err)
		return err
	}

	group.Add(func() error {
		logger.Log("transport", "HTTP", "addr", httpListener.Addr())

		registrar.Register()

		transport := {{$transportPackageName}}.NewHTTPHandler(transportOptions...)

		if httpPattern != "" {
			httpMux.Handle(httpPattern, transport)
			return http.Serve(httpListener, httpMux)
		}
		return http.Serve(httpListener, transport)
	}, func(error) {
		registrar.Deregister()

		httpListener.Close()
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

	httpAddr    string
	httpPattern string
	httpMux     *http.ServeMux
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

	if options.logger == nil {
		options.logger = spiderconn.DefaultLogger
	}

	if options.group == nil {
		options.group = &group.Group{}
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

	if options.registrarCreator == nil {
		options.registrarCreator = spiderconn.NewConsulRegistrar
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
}`
