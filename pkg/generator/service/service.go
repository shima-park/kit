package service

import (
	"io/ioutil"
	"strings"

	"ezrpro.com/micro/kit/pkg/cst"
	gen "ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/utils"
	"github.com/alecthomas/template"
)

type ServiceGenerator struct {
	opts Options
}

func NewServiceGenerator(opts ...Option) gen.Generator {
	options := newOptions(opts...)

	if !strings.HasSuffix(options.serviceName, options.serviceSuffix) {
		options.serviceName = utils.ToCamelCase(options.serviceName + options.serviceSuffix)
	}

	for i, method := range options.methods {
		options.methods[i] = utils.ToCamelCase(method)
	}

	return &ServiceGenerator{
		opts: options,
	}
}

func (g *ServiceGenerator) Generate() error {
	for tplName, readWriter := range g.opts.readWriterMap {

		tplBody, err := ioutil.ReadAll(readWriter.template)
		if err != nil {
			return err
		}

		var (
			reqAndResps  []gen.ReqAndResp
			refStructMap map[string]*cst.Struct
			constMap     []cst.Constant
		)
		if g.opts.csTree != nil {
			reqAndResps = gen.GetRequestAndResponseList(g.opts.csTree)
			if len(reqAndResps) > 0 {
				for _, reqAndResp := range reqAndResps {
					reqRefMap := gen.GetReferenceStructMap(g.opts.csTree, reqAndResp.Request)
					respRefMap := gen.GetReferenceStructMap(g.opts.csTree, reqAndResp.Response)
					refStructMap = gen.MergeStructMap(reqRefMap, respRefMap, refStructMap)
				}

			}
			constMap = g.opts.csTree.Consts()
		}

		t := template.New(string(tplName)).Funcs(
			map[string]interface{}{
				"GenType": func(pkg string, f cst.Field) string {
					t := f.Type
					typ := t.BaseType
					if t.ElementType != nil {
						typ = *t.ElementType
					} else if t.ValueType != nil {
						typ = *t.ValueType
					}
					if typ.GoType == cst.StructType {
						typ.X = pkg
					}
					return typ.String()
				},
			},
		)
		t, err = t.Parse(string(tplBody))
		if err != nil {
			return err
		}

		packageName := strings.ToLower(g.opts.serviceName)

		var data = map[string]interface{}{
			"PackageName":         packageName,
			"ServiceName":         g.opts.serviceName,
			"InterfaceMethods":    g.opts.methods,
			"RequestAndResponses": reqAndResps,
			"ReferenceStructMap":  refStructMap,
			"ConstMap":            constMap,
		}

		err = t.Execute(readWriter.writer, data)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetBaseServiceName(packageName, serviceSuffix string) string {
	packageName = strings.ToLower(packageName)
	serviceSuffix = strings.ToLower(serviceSuffix)
	return strings.TrimSuffix(packageName, serviceSuffix)
}
