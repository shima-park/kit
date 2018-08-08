package transport

import (
	"io/ioutil"
	"path/filepath"
	"text/template"

	"ezrpro.com/micro/kit/pkg/cst"
	gen "ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/generator/assignment"
	"ezrpro.com/micro/kit/pkg/utils"
)

type TransportGenerator struct {
	cst  cst.ConcreteSyntaxTree
	opts Options
}

func NewTransportGenerator(t cst.ConcreteSyntaxTree, opts ...Option) gen.Generator {
	options := newOptions(opts...)

	return &TransportGenerator{
		cst:  t,
		opts: options,
	}
}

func (g *TransportGenerator) Generate() error {
	for tplName, readWriter := range g.opts.readWriterMap {
		tplBody, err := ioutil.ReadAll(readWriter.template)
		if err != nil {
			return err
		}

		pbCST, err := getProtobufCST(g.opts.baseServiceName, g.cst.PackageName())
		if err != nil {
			return err
		}

		t := template.New(string(tplName)).Funcs(map[string]interface{}{
			"ToLowerFirstCamelCase":     utils.ToLowerFirstCamelCase,
			"ToCamelCase":               utils.ToCamelCase,
			"BasePath":                  filepath.Base,
			"GenerateAssignmentSegment": assignment.NewGeneratorFactory(g.cst, pbCST).Generate,
			"NewSimpleAlias":            assignment.NewSimpleAlias,
			"NewObjectAlias":            assignment.NewObjectAlias(g.cst, pbCST),
		})
		t, err = t.Parse(string(tplBody))
		if err != nil {
			return err
		}

		serviceIface, err := gen.FilterInterface(g.cst.Interfaces(), g.opts.serviceSuffix)
		if err != nil {
			return err
		}

		// protobuf的interface是以server结尾
		pbServiceIface, err := gen.FilterInterface(pbCST.Interfaces(), utils.GetProtobufServiceSuffix())
		if err != nil {
			return err
		}

		err = t.Execute(readWriter.writer, map[string]interface{}{
			"BaseServiceName":        g.opts.baseServiceName,
			"PackageName":            g.opts.transportPackageName,
			"ServiceName":            serviceIface.Name,
			"ServiceMethods":         serviceIface.Methods,
			"ServiceImportPath":      utils.GetServiceImportPath(g.opts.baseServiceName),
			"EndpointImportPath":     utils.GetEndpointImportPath(g.opts.baseServiceName),
			"ProtobufImportPath":     utils.GetProtobufImportPath(g.opts.baseServiceName),
			"RequestAndResponseList": gen.GetRequestAndResponseList(g.cst),
			"ProtobufCST": map[string]interface{}{
				"PackageName":            pbCST.PackageName(),
				"ServiceName":            pbServiceIface.Name,
				"RequestAndResponseList": gen.GetRequestAndResponseList(pbCST),
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func getProtobufCST(baseServiceName, servicePackageName string) (cst.ConcreteSyntaxTree, error) {
	pbGoPath := utils.GetProtobufFilePath(baseServiceName)

	pbGoFile := filepath.Join(pbGoPath, servicePackageName+".pb.go")
	pbCST, err := cst.New(pbGoFile)
	if err != nil {
		return nil, err
	}
	return pbCST, nil
}
